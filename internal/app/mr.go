package app

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/thegriglat/platui/internal/forge"
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
)

type mrCache struct {
	MRs []forge.MR `json:"mrs"`
}

func (a *App) mrKey(itemID string) string { return a.key("mrs", itemID) }

func (a *App) mrTTL() time.Duration {
	if t := a.cfg().GitLab.MRCache; t != nil {
		return *t
	}
	return 0
}

// mrBranches — ветки задачи: по шаблону, текущая и локальные с тем же ID.
func (a *App) mrBranches(p plane.Project, it plane.WorkItem) []string {
	var out []string
	if n, err := a.BranchName(p, it); err == nil && n != "" {
		out = append(out, n)
	}
	g, err := a.git()
	if err != nil {
		return out
	}
	same := func(b string) bool {
		ref, ok := gitbranch.ParseBranch(a.branchTemplate(), b)
		return ok && strings.EqualFold(ref.Identifier, p.Identifier) && ref.Sequence == it.Sequence
	}
	if cur, err := g.Current(); err == nil && cur != "" && same(cur) && !slices.Contains(out, cur) {
		out = append(out, cur)
	}
	if locals, err := g.Branches(); err == nil {
		for _, b := range locals {
			if same(b) && !slices.Contains(out, b) {
				out = append(out, b)
			}
		}
	}
	return out
}

// MRs — MR задачи через glab; результат кэшируется на cfg.GitLab.MRCache (fresh — в обход кэша).
func (a *App) MRs(ctx context.Context, p plane.Project, it plane.WorkItem, fresh bool) ([]forge.MR, error) {
	ttl := a.mrTTL()
	if !fresh && ttl > 0 {
		var c mrCache
		if at, ok := a.load(a.mrKey(it.ID), &c); ok && a.d.Now().Sub(at) < ttl {
			a.mu.Lock()
			a.mrs[it.ID] = c.MRs
			a.mu.Unlock()
			return c.MRs, nil
		}
	}
	dir := ""
	if g, err := a.git(); err == nil {
		dir, _ = g.Toplevel()
	}
	key := p.Identifier + "-" + itoa(it.Sequence)
	mrs, err := a.d.Forge.FindMRs(ctx, dir, a.mrBranches(p, it), key)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	a.mrs[it.ID] = mrs
	a.mu.Unlock()
	if ttl > 0 {
		a.save(a.mrKey(it.ID), mrCache{MRs: mrs})
	}
	return mrs, nil
}

// CachedMRs — MR из памяти или кэша без вызова glab.
func (a *App) CachedMRs(itemID string) ([]forge.MR, bool) {
	a.mu.Lock()
	mrs, ok := a.mrs[itemID]
	a.mu.Unlock()
	if ok {
		return mrs, true
	}
	if a.mrTTL() <= 0 {
		return nil, false
	}
	var c mrCache
	if _, ok := a.load(a.mrKey(itemID), &c); !ok {
		return nil, false
	}
	return c.MRs, true
}
