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

// Changes — отличия двух снимков списка задач (по ID).
type Changes struct {
	New, Updated []plane.WorkItem
	Removed      []string
}

// Empty — изменений нет.
func (c Changes) Empty() bool { return len(c.New) == 0 && len(c.Updated) == 0 && len(c.Removed) == 0 }

// Diff сравнивает снимки; Updated — UpdatedAt изменился.
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

// CachedItems — задачи из памяти, иначе с диска; время сохранения.
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

// commitItems — полный список: обновляет и время.
func (a *App) commitItems(projectID string, cur []plane.WorkItem) {
	now := a.d.Now()
	a.storeItems(projectID, cur, now)
	a.save(a.key("itemsat", projectID), now)
}

// storeItems сохраняет список с заданным временем (правка одной задачи его не меняет).
func (a *App) storeItems(projectID string, cur []plane.WorkItem, at time.Time) {
	a.mu.Lock()
	a.items[projectID] = slices.Clone(cur)
	a.itemsAt[projectID] = at
	a.mu.Unlock()
	a.save(a.itemsKey(projectID), cur)
}

// RefreshItems загружает задачи, считает отличия от прошлого снимка (память,
// иначе кэш) и сохраняет. Прошлого снимка нет — Changes пусто.
func (a *App) RefreshItems(ctx context.Context, projectID string) ([]plane.WorkItem, Changes, error) {
	cur, err := a.d.API.WorkItems(ctx, projectID)
	if err != nil {
		return nil, Changes{}, err
	}
	a.online()
	var ch Changes
	if old, _, ok := a.CachedItems(projectID); ok {
		ch = Diff(old, cur)
	}
	a.commitItems(projectID, cur)
	return cur, ch, nil
}

// Reload — обновление в обход кэша; ошибка не трогает прежние данные.
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
	a.commitItems(projectID, cur)
	return m, cur, ch, nil
}

// NextPoll — интервал следующего опроса: при ошибке удваивается (429 — до 10m, иное — до 5m).
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

// Find ищет задачу по ключу проекта и номеру.
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

// ParseRef разбирает "PROJ-42", "proj-42" или "42" (с defaultProject).
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

// CurrentItem — задача, на которую указывает текущая ветка.
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
