package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

const historyMax = 50

// history is the history of opened cards (in memory, up to historyMax): previous tasks and the current one.
type history struct {
	stack []string // IDs of previously opened tasks, the last is the nearest
	last  string   // ID of the last opened card
}

// visit records opening card id: the previous one goes to history.
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
	registerAction(action{ID: "nav.history", Title: "Back to previously opened task", Group: "navigation",
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
				h.last = id // going back does not add the current one to history
				if newItemScreen == nil {
					return nil
				}
				s.cursor, s.openItem = it.ID, it.ID
				return pushScreen(newItemScreen(s, it))
			}
			s.setStatus("history is empty", false)
			return nil
		}})
}
