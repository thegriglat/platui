package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
)

func init() {
	registerAction(action{ID: "item.copy", Title: "Копировать (ID, URL, название, ветка, Markdown)", Group: "задача",
		Keys: []string{"y"}, Hint: "копировать", Scopes: cardScopes,
		Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
			return pushScreen(newCopyMenu(m.s, it))
		})})
}

// newCopyMenu — меню «что копировать»: i ID, u URL, t название, b ветка, m markdown.
func newCopyMenu(s *session, it plane.WorkItem) screen {
	c := s.deps.App.CopyItems(*s.project, it)
	vals := []struct{ hot, label, val string }{
		{"i", "ID", c.ID}, {"u", "URL", c.URL}, {"t", "название", c.Title}, {"b", "имя ветки", c.Branch}, {"m", "Markdown-ссылка", c.Markdown},
	}
	rows := make([]pickRow, len(vals))
	for i, v := range vals {
		rows[i] = pickRow{Label: v.label + s.st.muted.Render("  "+v.val), Hot: v.hot}
	}
	return newListScreen("Копировать", rows, 0, func(i int) tea.Cmd {
		v := vals[i]
		return func() tea.Msg {
			if v.val == "" {
				return statusMsg{v.label + ": недоступно", true}
			}
			if err := s.deps.Clipboard(v.val); err != nil {
				return statusMsg{"буфер обмена: " + err.Error(), true}
			}
			return statusMsg{"скопировано: " + v.val, false}
		}
	})
}
