package tui

import (
	"slices"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/plane"
)

// row — строка таблицы: задача, глубина вложенности, число детей, закладка.
type row struct {
	it       plane.WorkItem
	depth    int
	kids     int
	expanded bool
	pinned   bool
}

// bookmarkIDs — закладки текущего проекта (порядок добавления).
func (s *session) bookmarkIDs() []string {
	if s.deps.App == nil || s.project == nil {
		return nil
	}
	return s.deps.App.Bookmarks(s.project.ID)
}

// buildRows строит строки таблицы из visible(): закладки сверху (без дерева), остальные —
// деревом по ParentID, свёрнутые ветки скрыты. При активном поиске — плоский список без закреплений.
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

// revealAncestors раскрывает предков задачи id (если она скрыта свёрнутой веткой); true — что-то раскрыто.
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

// treePrefix — отступ и маркер ветки перед названием.
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
