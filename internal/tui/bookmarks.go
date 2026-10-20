package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
)

func init() {
	registerAction(action{ID: "item.bookmark", Title: "Закладка вкл/выкл", Group: "задача", Keys: []string{"B"},
		Hint: "закладка", Scopes: cardScopes,
		Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
			s := m.s
			on, err := s.deps.App.ToggleBookmark(s.project.ID, it.ID)
			switch {
			case err != nil:
				s.setStatus("закладки: "+err.Error(), true)
			case on:
				s.setStatus("закладка: "+itoaKey(s, it.Sequence), false)
			default:
				s.setStatus("закладка снята: "+itoaKey(s, it.Sequence), false)
			}
			return nil
		})})
}
