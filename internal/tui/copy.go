package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
)

func init() {
	registerAction(action{ID: "item.copy", Title: "Copy (ID, URL, title, branch, Markdown)", Group: "task",
		Keys: []string{"y"}, Hint: "copy", Scopes: cardScopes,
		Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
			return pushScreen(newCopyMenu(m.s, it))
		})})
}

// newCopyMenu is the "what to copy" menu: i ID, u URL, t title, b branch, m markdown.
func newCopyMenu(s *session, it plane.WorkItem) screen {
	c := s.deps.App.CopyItems(*s.project, it)
	vals := []struct{ hot, label, val string }{
		{"i", "ID", c.ID}, {"u", "URL", c.URL}, {"t", "title", c.Title}, {"b", "branch name", c.Branch}, {"m", "Markdown link", c.Markdown},
	}
	rows := make([]pickRow, len(vals))
	for i, v := range vals {
		rows[i] = pickRow{Label: v.label + s.st.muted.Render("  "+v.val), Hot: v.hot}
	}
	return newListScreen("Copy", rows, 0, func(i int) tea.Cmd {
		v := vals[i]
		return func() tea.Msg {
			if v.val == "" {
				return statusMsg{v.label + ": unavailable", true}
			}
			if err := s.deps.Clipboard(v.val); err != nil {
				return statusMsg{"clipboard: " + err.Error(), true}
			}
			return statusMsg{"copied: " + v.val, false}
		}
	})
}
