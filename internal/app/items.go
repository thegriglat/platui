package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
)

// Changes are the differences between two task list snapshots (by ID).
type Changes struct {
	New, Updated []plane.WorkItem
	Removed      []string
}

// Empty reports no changes.
func (c Changes) Empty() bool { return len(c.New) == 0 && len(c.Updated) == 0 && len(c.Removed) == 0 }

// Diff compares snapshots; Updated means UpdatedAt changed.
func Diff(old, cur []plane.WorkItem) Changes {
	var c Changes
	prev := make(map[string]plane.WorkItem, len(old))
	for _, it := range old {
		prev[it.ID] = it
	}
	seen := make(map[string]bool, len(cur))
	for _, it := range cur {
		seen[it.ID] = true
		p, ok := prev[it.ID]
		switch {
		case !ok:
			c.New = append(c.New, it)
		case !p.UpdatedAt.Equal(it.UpdatedAt):
			c.Updated = append(c.Updated, it)
		}
	}
	for _, it := range old {
		if !seen[it.ID] {
			c.Removed = append(c.Removed, it.ID)
		}
	}
	return c
}

func (a *App) itemsKey(projectID string) string { return a.key("items", projectID) }

// CachedItems returns tasks from memory, otherwise from disk, with the save time.
func (a *App) CachedItems(projectID string) ([]plane.WorkItem, time.Time, bool) {
	a.mu.Lock()
	if its, ok := a.items[projectID]; ok {
		at := a.itemsAt[projectID]
		a.mu.Unlock()
		return slices.Clone(its), at, true
	}
	a.mu.Unlock()
	var its []plane.WorkItem
	at, ok := a.load(a.itemsKey(projectID), &its)
	if !ok {
		return nil, time.Time{}, false
	}
	var full time.Time
	if _, ok := a.load(a.key("itemsat", projectID), &full); ok && !full.IsZero() {
		at = full
	}
	return its, at, true
}

// commitItems commits the full list: updates the time and the sync mark too;
// write=false keeps the list file on disk (it has not changed).
func (a *App) commitItems(projectID string, cur []plane.WorkItem, write bool) {
	now := a.d.Now()
	if write {
		a.storeItems(projectID, cur, now)
	} else {
		a.mu.Lock()
		a.items[projectID] = slices.Clone(cur)
		a.itemsAt[projectID] = now
		a.mu.Unlock()
	}
	a.save(a.key("itemsat", projectID), now)
	var mark time.Time
	for _, it := range cur {
		if it.UpdatedAt.After(mark) {
			mark = it.UpdatedAt
		}
	}
	a.mu.Lock()
	a.synced[projectID] = mark
	a.mu.Unlock()
	a.save(a.key("itemssync", projectID), mark)
}

// syncMark is the newest updated_at as of the last sync; zero — unknown.
// Edits of single tasks do not move it: changes made by others before them would be skipped.
func (a *App) syncMark(projectID string) time.Time {
	a.mu.Lock()
	t, ok := a.synced[projectID]
	a.mu.Unlock()
	if !ok {
		a.load(a.key("itemssync", projectID), &t)
	}
	return t
}

// fetchItems reads the tasks: only the changed ones merged into old when there is a sync
// mark, all of them otherwise or when tasks were removed (the merged count exceeds the total).
func (a *App) fetchItems(ctx context.Context, projectID string, old []plane.WorkItem, haveOld bool) ([]plane.WorkItem, error) {
	if since := a.syncMark(projectID); haveOld && !since.IsZero() {
		upd, total, err := a.d.API.WorkItemsUpdated(ctx, projectID, since)
		if err != nil {
			return nil, err
		}
		if cur := mergeItems(old, upd); len(cur) == total {
			return cur, nil
		}
	}
	return a.d.API.WorkItems(ctx, projectID)
}

// mergeItems replaces the tasks of old with those of upd and adds the new ones;
// the order is that of the API: newest created first.
func mergeItems(old, upd []plane.WorkItem) []plane.WorkItem {
	byID := make(map[string]plane.WorkItem, len(upd))
	for _, it := range upd {
		byID[it.ID] = it
	}
	cur := make([]plane.WorkItem, 0, len(old)+len(upd))
	for _, it := range old {
		if u, ok := byID[it.ID]; ok {
			it = u
			delete(byID, it.ID)
		}
		cur = append(cur, it)
	}
	if len(byID) == 0 {
		return cur
	}
	for _, it := range upd {
		if _, ok := byID[it.ID]; ok {
			cur = append(cur, it)
		}
	}
	slices.SortStableFunc(cur, func(a, b plane.WorkItem) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return cur
}

// storeItems saves the list with the given time (editing one task does not change it).
func (a *App) storeItems(projectID string, cur []plane.WorkItem, at time.Time) {
	a.mu.Lock()
	a.items[projectID] = slices.Clone(cur)
	a.itemsAt[projectID] = at
	a.mu.Unlock()
	a.save(a.itemsKey(projectID), cur)
}

// RefreshItems loads tasks, computes the differences from the previous snapshot (memory,
// otherwise cache) and saves. No previous snapshot: Changes is empty.
func (a *App) RefreshItems(ctx context.Context, projectID string) ([]plane.WorkItem, Changes, error) {
	old, _, ok := a.CachedItems(projectID)
	cur, err := a.fetchItems(ctx, projectID, old, ok)
	if err != nil {
		return nil, Changes{}, err
	}
	a.online()
	var ch Changes
	if ok {
		ch = Diff(old, cur)
	}
	a.commitItems(projectID, cur, !ok || !ch.Empty())
	return cur, ch, nil
}

// Reload refreshes bypassing the cache; an error leaves the previous data untouched.
func (a *App) Reload(ctx context.Context, projectID string) (Meta, []plane.WorkItem, Changes, error) {
	m, err := a.fetchMeta(ctx, projectID)
	if err != nil {
		return Meta{}, nil, Changes{}, err
	}
	cur, err := a.d.API.WorkItems(ctx, projectID)
	if err != nil {
		return Meta{}, nil, Changes{}, err
	}
	var ch Changes
	if old, _, ok := a.CachedItems(projectID); ok {
		ch = Diff(old, cur)
	}
	a.online()
	a.commitMeta(m)
	a.commitItems(projectID, cur, true)
	return m, cur, ch, nil
}

// NextPoll is the next polling interval: doubles on error (429 up to 10m, otherwise up to 5m).
func NextPoll(base, prev time.Duration, err error) time.Duration {
	if err == nil {
		return base
	}
	limit := 5 * time.Minute
	if errors.Is(err, plane.ErrRateLimited) {
		limit = 10 * time.Minute
	}
	return min(max(prev, base)*2, limit)
}

// Find looks up a task by project key and number.
func (a *App) Find(ctx context.Context, identifier string, seq int) (plane.Project, plane.WorkItem, error) {
	p, err := a.Project(ctx, identifier)
	if err != nil {
		return plane.Project{}, plane.WorkItem{}, err
	}
	if its, at, ok := a.CachedItems(p.ID); ok {
		for _, it := range its {
			if it.Sequence == seq {
				if off, _ := a.Offline(); off {
					a.markOffline(at)
				}
				return p, it, nil
			}
		}
	}
	it, err := a.d.API.WorkItemByKey(ctx, p.Identifier, seq)
	if errors.Is(err, plane.ErrNotFound) {
		return p, plane.WorkItem{}, fmt.Errorf("%w: %s-%d", ErrItemNotFound, p.Identifier, seq)
	}
	return p, it, err
}

// ParseRef parses "PROJ-42", "proj-42" or "42" (with defaultProject).
func ParseRef(s, defaultProject string) (identifier string, seq int, ok bool) {
	s = strings.TrimSpace(s)
	id, num := defaultProject, s
	if i := strings.LastIndex(s, "-"); i >= 0 {
		id, num = strings.TrimSpace(s[:i]), s[i+1:]
	}
	n, err := strconv.Atoi(num)
	if err != nil || n <= 0 || id == "" || strings.ContainsAny(num, "+- ") || strings.ContainsAny(id, "- /") {
		return "", 0, false
	}
	return strings.ToUpper(id), n, true
}

// CurrentItem is the task the current branch points to.
func (a *App) CurrentItem(ctx context.Context) (plane.Project, plane.WorkItem, error) {
	g, err := a.git()
	if err != nil {
		return plane.Project{}, plane.WorkItem{}, err
	}
	br, err := g.Current()
	if err != nil {
		return plane.Project{}, plane.WorkItem{}, err
	}
	ref, ok := gitbranch.ParseBranch(a.branchTemplate(), br)
	if !ok {
		return plane.Project{}, plane.WorkItem{}, ErrNoItemForBranch
	}
	return a.Find(ctx, ref.Identifier, ref.Sequence)
}

func itoa(n int) string { return strconv.Itoa(n) }
