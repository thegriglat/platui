package tui

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func init() { newTable = func(s *session) pane { return &table{} } }

// table is the task table of the current tab; the cursor is s.cursor (task ID).
type table struct {
	list     cursorList
	expanded map[string]bool // expanded parents (subtasks)
	proj     string
	rows     []row
}

// sync aligns the list with the table rows and the s.cursor position.
func (t *table) sync(s *session, h int) []string {
	id := ""
	if s.project != nil {
		id = s.project.ID
	}
	if t.expanded == nil || t.proj != id {
		t.expanded, t.proj = map[string]bool{}, id
	}
	t.rows = s.buildRows(t.expanded)
	if s.cursor != "" && !slices.ContainsFunc(t.rows, func(r row) bool { return r.it.ID == s.cursor }) &&
		s.revealAncestors(s.cursor, t.expanded) {
		t.rows = s.buildRows(t.expanded)
	}
	t.list.SetLen(len(t.rows))
	t.list.SetHeight(h)
	ids := make([]string, len(t.rows))
	for i, r := range t.rows {
		ids[i] = r.it.ID
		if r.it.ID == s.cursor {
			t.list.Set(i)
		}
	}
	if len(t.rows) > 0 {
		s.cursor = ids[t.list.Pos()]
	} else {
		s.cursor = ""
	}
	return ids
}

// fold expands (open) or collapses the subtasks under the cursor; on a descendant "←" goes to the parent.
func (t *table) fold(s *session, open bool) {
	i := t.list.Pos()
	if i >= len(t.rows) {
		return
	}
	r := t.rows[i]
	switch {
	case open && r.kids > 0:
		t.expanded[r.it.ID] = true
	case !open && r.kids > 0 && r.expanded:
		delete(t.expanded, r.it.ID)
	case !open && r.depth > 0 && r.it.ParentID != "":
		s.cursor = r.it.ParentID
	}
}

func (t *table) Update(s *session, msg tea.Msg) (pane, tea.Cmd) {
	if f, ok := msg.(foundMsg); ok {
		return t, s.onFound(f)
	}
	if c, ok := msg.(rowClickMsg); ok {
		ids := t.sync(s, max(t.list.h, 1))
		if c.row < 0 || c.row >= len(ids) {
			return t, nil
		}
		t.list.Set(c.row)
		s.cursor = ids[c.row]
		if it, ok := s.itemByID(s.cursor); ok && c.open {
			return t, openItem(s, it)
		}
		return t, nil
	}
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return t, nil
	}
	ids := t.sync(s, max(t.list.h, 1))
	name := keyName(k)
	if t.list.Key(name) && len(ids) > 0 {
		s.cursor = ids[t.list.Pos()]
		return t, nil
	}
	switch name {
	case "right", "left":
		t.fold(s, name == "right")
		t.sync(s, max(t.list.h, 1))
	case "enter":
		if len(ids) > 0 {
			if it, ok := s.itemByID(ids[t.list.Pos()]); ok {
				return t, openItem(s, it)
			}
		}
	}
	return t, nil
}

func (t *table) View(s *session, width, height int) string {
	t.sync(s, height-1)
	if len(t.rows) == 0 {
		msg := "no tasks"
		switch {
		case s.loading && !s.live:
			msg = "loading…"
		case s.query != "" || s.mine || s.filter > 0:
			msg = "no matches"
		}
		return s.st.muted.Render(msg)
	}
	l := s.layoutCols(width)
	rows := []string{s.headerRow(l)}
	from, to := t.list.Range()
	for i := from; i < to; i++ {
		rows = append(rows, s.itemRow(t.rows[i], l, i == t.list.Pos()))
	}
	return strings.Join(rows, "\n")
}
