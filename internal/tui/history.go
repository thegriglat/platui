package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

const historyMax = 50

// history — история открытых карточек (в памяти, до historyMax): предыдущие задачи и текущая.
type history struct {
	stack []string // ID предыдущих открытых задач, последняя — ближайшая
	last  string   // ID последней открытой карточки
}

// visit запоминает открытие карточки id: предыдущая уходит в историю.
func (h *history) visit(id string) {
	if h.last != "" && h.last != id {
		h.stack = append(h.stack, h.last)
		if len(h.stack) > historyMax {
			h.stack = h.stack[len(h.stack)-historyMax:]
		}
	}
	h.last = id
}

func init() {
	registerAction(action{ID: "nav.history", Title: "Назад к предыдущей открытой задаче", Group: "навигация",
		Keys: []string{"ctrl+o"}, Scopes: cardScopes, Run: func(m *model) tea.Cmd {
			s := m.s
			h := &s.hist
			for len(h.stack) > 0 {
				id := h.stack[len(h.stack)-1]
				h.stack = h.stack[:len(h.stack)-1]
				if id == h.last {
					continue
				}
				it, ok := s.itemByID(id)
				if !ok {
					continue
				}
				if m.top().Scope() == "item" {
					m.pop()
				}
				h.last = id // возврат не добавляет текущую в историю
				if newItemScreen == nil {
					return nil
				}
				s.cursor, s.openItem = it.ID, it.ID
				return pushScreen(newItemScreen(s, it))
			}
			s.setStatus("история пуста", false)
			return nil
		}})
}
