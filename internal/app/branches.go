package app

import (
	"context"
	"slices"
	"strings"

	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
)

func (a *App) branchTemplate() string {
	if t := a.cfg().Branch.Template; t != "" {
		return t
	}
	return config.DefaultBranchTemplate
}

func (a *App) commitTemplate() string {
	if t := a.cfg().Branch.CommitTemplate; t != "" {
		return t
	}
	return config.DefaultCommitTemplate
}

func (a *App) remote() string {
	if r := a.cfg().Branch.Remote; r != "" {
		return r
	}
	return config.DefaultRemote
}

func gitItem(p plane.Project, it plane.WorkItem) gitbranch.Item {
	return gitbranch.Item{Identifier: p.Identifier, Sequence: it.Sequence, Title: it.Name}
}

// BranchName returns the task branch name from the template.
func (a *App) BranchName(p plane.Project, it plane.WorkItem) (string, error) {
	return gitbranch.Render(a.branchTemplate(), gitItem(p, it))
}

// BranchStatuses returns task branch statuses (key is the task ID): the template branch
// and local branches with the same task ID. Tasks without branches are skipped.
func (a *App) BranchStatuses(p plane.Project, items []plane.WorkItem) (map[string]gitbranch.BranchStatus, error) {
	g, err := a.git()
	if err != nil {
		return nil, err
	}
	locals, err := g.Branches()
	if err != nil {
		return nil, err
	}
	bySeq := map[int][]string{}
	for _, b := range locals {
		if ref, ok := gitbranch.ParseBranch(a.branchTemplate(), b); ok && strings.EqualFold(ref.Identifier, p.Identifier) {
			bySeq[ref.Sequence] = append(bySeq[ref.Sequence], b)
		}
	}
	cands := make(map[string][]string, len(items)) // task ID → branches, template branch first
	var names []string
	for _, it := range items {
		var c []string
		c = append(c, a.templateName(p, it))
		for _, b := range bySeq[it.Sequence] {
			if !slices.Contains(c, b) {
				c = append(c, b)
			}
		}
		cands[it.ID] = c
		names = append(names, c...)
	}
	if len(names) == 0 {
		return map[string]gitbranch.BranchStatus{}, nil
	}
	sts, err := g.Statuses(slices.Compact(slices.Sorted(slices.Values(names))), a.remote(), a.cfg().Branch.Base)
	if err != nil {
		return nil, err
	}
	res := map[string]gitbranch.BranchStatus{}
	for id, c := range cands {
		for _, n := range c {
			if st, ok := sts[n]; ok && (st.Local || st.Remote) {
				res[id] = st
				break
			}
		}
	}
	return res, nil
}

// Stale is a local task branch that can be removed.
type Stale struct {
	Branch string
	Item   plane.WorkItem
}

// StaleBranches returns branches of the project's tasks in completed|cancelled groups, except the current one.
func (a *App) StaleBranches(ctx context.Context, p plane.Project) ([]Stale, error) {
	g, err := a.git()
	if err != nil {
		return nil, err
	}
	branches, err := g.Branches()
	if err != nil {
		return nil, err
	}
	cur, _ := g.Current()
	m, err := a.Meta(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	items, _, ok := a.CachedItems(p.ID)
	if !ok {
		if items, _, err = a.RefreshItems(ctx, p.ID); err != nil {
			return nil, err
		}
	}
	done := map[string]bool{}
	for _, s := range m.States {
		if g := strings.ToLower(s.Group); g == "completed" || g == "cancelled" {
			done[s.ID] = true
		}
	}
	bySeq := map[int]plane.WorkItem{}
	for _, it := range items {
		bySeq[it.Sequence] = it
	}
	var out []Stale
	for _, b := range slices.Sorted(slices.Values(branches)) {
		ref, ok := gitbranch.ParseBranch(a.branchTemplate(), b)
		if b == cur || !ok || !strings.EqualFold(ref.Identifier, p.Identifier) {
			continue
		}
		if it, ok := bySeq[ref.Sequence]; ok && done[it.StateID] {
			out = append(out, Stale{Branch: b, Item: it})
		}
	}
	return out, nil
}

// DeleteBranches deletes branches; the result is an error per name.
func (a *App) DeleteBranches(names []string, force bool) map[string]error {
	res := make(map[string]error, len(names))
	g, err := a.git()
	for _, n := range names {
		if err != nil {
			res[n] = err
			continue
		}
		res[n] = g.Delete(n, force)
	}
	return res
}

// CopyItems holds the lines for the clipboard.
type CopyItems struct{ ID, URL, Title, Branch, Markdown string }

// CopyItems prepares the values to copy.
func (a *App) CopyItems(p plane.Project, it plane.WorkItem) CopyItems {
	id := p.Identifier + "-" + itoa(it.Sequence)
	url := a.d.API.WebURL(p, it)
	br, _ := a.BranchName(p, it)
	return CopyItems{ID: id, URL: url, Title: it.Name, Branch: br, Markdown: "[" + id + "](" + url + ")"}
}

// templateName builds the name from the template without running git (no ValidName):
// an invalid name simply won't be found among the branches.
func (a *App) templateName(p plane.Project, it plane.WorkItem) string {
	return strings.NewReplacer(
		"{identifier}", p.Identifier,
		"{sequence}", itoa(it.Sequence),
		"{slug}", gitbranch.Slug(it.Name, 40),
	).Replace(a.branchTemplate())
}
