package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// stubTable is a table stub until PT-5b: "ID title", cursor and navigation.
type stubTable struct{ list cursorList }

func newStubTable(*session) pane { return &stubTable{} }

func (t *stubTable) sync(s *session, h int) []string {
	vis := s.visible()
	t.list.SetLen(len(vis))
	t.list.SetHeight(h)
	ids := make([]string, len(vis))
	for i, it := range vis {
		ids[i] = it.ID
		if it.ID == s.cursor {
			t.list.Set(i)
		}
	}
	if len(vis) > 0 {
		s.cursor = ids[t.list.Pos()]
	}
	return ids
}

func (t *stubTable) Update(s *session, msg tea.Msg) (pane, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		ids := t.sync(s, max(t.list.h, 1))
		if t.list.Key(keyName(k)) && len(ids) > 0 {
			s.cursor = ids[t.list.Pos()]
		}
	}
	return t, nil
}

func (t *stubTable) View(s *session, width, height int) string {
	t.sync(s, height)
	vis := s.visible()
	from, to := t.list.Range()
	if len(vis) == 0 {
		return s.st.muted.Render("no data")
	}
	var b []string
	for i := from; i < to; i++ {
		line := itoaKey(s, vis[i].Sequence) + "  " + vis[i].Name
		if i == t.list.Pos() {
			line = s.st.sel.Render(clip(line, width))
		}
		b = append(b, line)
	}
	return strings.Join(b, "\n")
}

func itoaKey(s *session, seq int) string {
	if s.project == nil {
		return itoa(seq)
	}
	return s.project.Identifier + "-" + itoa(seq)
}
