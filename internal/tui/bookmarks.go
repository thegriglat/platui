package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
)

func init() {
	registerAction(action{ID: "item.bookmark", Title: "Toggle bookmark", Group: "task", Keys: []string{"B"},
		Hint: "bookmark", Scopes: cardScopes,
		Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
			s := m.s
			on, err := s.deps.App.ToggleBookmark(s.project.ID, it.ID)
			switch {
			case err != nil:
				s.setStatus("bookmarks: "+err.Error(), true)
			case on:
				s.setStatus("bookmarked: "+itoaKey(s, it.Sequence), false)
			default:
				s.setStatus("bookmark removed: "+itoaKey(s, it.Sequence), false)
			}
			return nil
		})})
}
