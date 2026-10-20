package tui

import tea "github.com/charmbracelet/bubbletea"

var sortNames = []string{"порядок API", "приоритет ▲", "состояние ▲", "обновлено ▼", "ID ▲"}

func init() {
	registerAction(action{ID: "view.sort", Title: "Сортировка: следующий порядок", Group: "виды", Keys: []string{"o"},
		Hint: "сортировка", Scopes: []string{"main"}, Run: func(m *model) tea.Cmd {
			m.s.sortBy = (m.s.sortBy + 1) % len(sortNames)
			m.s.setStatus("сортировка: "+sortNames[m.s.sortBy], false)
			return nil
		}})
	registerAction(action{ID: "item.mark", Title: "Отметить задачу", Group: "задача", Keys: []string{"space"},
		Hint: "отметить", Scopes: []string{"main"}, Run: func(m *model) tea.Cmd {
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
