package tui

import tea "github.com/charmbracelet/bubbletea"

var sortNames = []string{"API order", "priority ▲", "state ▲", "updated ▼", "ID ▲"}

func init() {
	registerAction(action{ID: "view.sort", Title: "Sort: next order", Group: "views", Keys: []string{"o"},
		Hint: "sort", Scopes: []string{"main"}, Run: func(m *model) tea.Cmd {
			m.s.sortBy = (m.s.sortBy + 1) % len(sortNames)
			m.s.setStatus("sort: "+sortNames[m.s.sortBy], false)
			return nil
		}})
	registerAction(action{ID: "item.mark", Title: "Mark task", Group: "task", Keys: []string{"space"},
		Hint: "mark", Scopes: []string{"main"}, Run: func(m *model) tea.Cmd {
			s := m.s
			if _, r := s.paneWidths(); s.cursor == "" || (s.focus == 1 && r > 0) {
				return nil
			}
			if s.marks[s.cursor] {
				delete(s.marks, s.cursor)
			} else {
				s.marks[s.cursor] = true
			}
			var cmd tea.Cmd
			m.stack[len(m.stack)-1], cmd = m.top().Update(s, tea.KeyMsg{Type: tea.KeyDown})
			return cmd
		}})
}
