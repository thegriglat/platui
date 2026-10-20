package tui

import (
	"slices"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/plane"
)

// row is a table row: task, nesting depth, number of children, bookmark.
type row struct {
	it       plane.WorkItem
	depth    int
	kids     int
	expanded bool
	pinned   bool
}

// bookmarkIDs are the bookmarks of the current project (in the order added).
func (s *session) bookmarkIDs() []string {
	if s.deps.App == nil || s.project == nil {
		return nil
	}
	return s.deps.App.Bookmarks(s.project.ID)
}

// buildRows builds table rows from visible(): bookmarks on top (no tree), the rest as a
// tree by ParentID, collapsed branches hidden. With an active search — a flat list without pinning.
func (s *session) buildRows(expanded map[string]bool) []row {
	vis := s.visible()
	if s.hits != nil {
		rows := make([]row, len(vis))
		for i, it := range vis {
			rows[i] = row{it: it}
		}
		return rows
	}
	byID := make(map[string]plane.WorkItem, len(vis))
	for _, it := range vis {
		byID[it.ID] = it
	}
	var rows []row
	pinned := map[string]bool{}
	for _, id := range s.bookmarkIDs() {
		if it, ok := byID[id]; ok && !pinned[id] {
			pinned[id] = true
			rows = append(rows, row{it: it, pinned: true})
		}
	}
	rest := slices.DeleteFunc(slices.Clone(vis), func(it plane.WorkItem) bool { return pinned[it.ID] })
	var walk func(ns []*app.Node)
	walk = func(ns []*app.Node) {
		for _, n := range ns {
			r := row{it: n.Item, depth: n.Depth, kids: len(n.Children), expanded: expanded[n.Item.ID]}
			rows = append(rows, r)
			if r.expanded {
				walk(n.Children)
			}
		}
	}
	walk(app.Tree(rest))
	return rows
}

// revealAncestors expands the ancestors of task id (if hidden by a collapsed branch); true if anything was expanded.
func (s *session) revealAncestors(id string, expanded map[string]bool) bool {
	parents := map[string]string{}
	for _, it := range s.visible() {
		parents[it.ID] = it.ParentID
	}
	changed := false
	for p, i := parents[id], 0; p != "" && i < len(parents); i++ {
		if _, ok := parents[p]; !ok {
			break
		}
		if !expanded[p] {
			expanded[p], changed = true, true
		}
		p = parents[p]
	}
	return changed
}

// treePrefix is the indent and branch marker before the title.
func (r row) treePrefix() string {
	p := ""
	for i := 0; i < r.depth; i++ {
		p += "  "
	}
	switch {
	case r.kids == 0:
	case r.expanded:
		p += "▾ "
	default:
		p += "▸ (" + itoa(r.kids) + ") "
	}
	return p
}
