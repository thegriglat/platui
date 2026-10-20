package app

import (
	"context"
	"slices"

	"github.com/thegriglat/platui/internal/plane"
)

// replaceItem replaces a task in the loaded list (memory or cache); no list: nothing happens.
func (a *App) replaceItem(projectID string, it plane.WorkItem) {
	its, at, ok := a.CachedItems(projectID)
	if !ok {
		return
	}
	i := slices.IndexFunc(its, func(x plane.WorkItem) bool { return x.ID == it.ID })
	if i < 0 {
		return
	}
	its[i] = it
	a.storeItems(projectID, its, at)
}

// Item loads a task from the network and replaces it in memory and cache.
func (a *App) Item(ctx context.Context, projectID, itemID string) (plane.WorkItem, error) {
	it, err := a.d.API.WorkItem(ctx, projectID, itemID)
	if err != nil {
		return plane.WorkItem{}, err
	}
	a.replaceItem(projectID, it)
	return it, nil
}

// Update applies a patch and replaces the task in memory and cache.
func (a *App) Update(ctx context.Context, projectID, itemID string, p plane.WorkItemPatch) (plane.WorkItem, error) {
	it, err := a.d.API.UpdateWorkItem(ctx, projectID, itemID, p)
	if err != nil {
		return plane.WorkItem{}, err
	}
	a.replaceItem(projectID, it)
	return it, nil
}

// Comments returns all task comments ordered by CreatedAt.
func (a *App) Comments(ctx context.Context, projectID, itemID string) ([]plane.Comment, error) {
	cs, err := a.d.API.Comments(ctx, projectID, itemID)
	if err != nil {
		return nil, err
	}
	cs = slices.Clone(cs)
	slices.SortStableFunc(cs, func(x, y plane.Comment) int { return x.CreatedAt.Compare(y.CreatedAt) })
	return cs, nil
}
