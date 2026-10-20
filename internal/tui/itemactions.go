package tui

import (
	"context"
	"slices"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
)

var cardScopes = []string{"main", "kanban", "item"}

// target — задача, над которой работает действие: открытая карточка или задача под курсором.
func (m *model) target() (plane.WorkItem, bool) {
	id := m.s.cursor
	if m.top().Scope() == "item" && m.s.openItem != "" {
		id = m.s.openItem
	}
	it, ok := m.s.itemByID(id)
	if !ok || m.s.project == nil {
		m.s.setStatus("нет задачи", true)
		return it, false
	}
	return it, true
}

// patchWith применяет патч (app.Update) → itemMsg и сообщение об успехе в статусе.
func patchWith(s *session, id string, p plane.WorkItemPatch, done string) tea.Cmd {
	a, pid := s.deps.App, s.project.ID
	return func() tea.Msg {
		it, err := a.Update(context.Background(), pid, id, p)
		if err != nil {
			return statusMsg{err.Error(), true}
		}
		return tea.BatchMsg{func() tea.Msg { return itemMsg{item: it} }, func() tea.Msg { return statusMsg{done, false} }}
	}
}

// withTarget оборачивает действие: Run получает задачу или ничего не делает.
func withTarget(f func(m *model, it plane.WorkItem) tea.Cmd) func(m *model) tea.Cmd {
	return func(m *model) tea.Cmd {
		if it, ok := m.target(); ok {
			return f(m, it)
		}
		return nil
	}
}

func init() {
	for _, a := range []action{
		{ID: "item.state", Title: "Сменить состояние", Group: "задача", Keys: []string{"s"}, Hint: "состояние", Scopes: cardScopes,
			Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
				s := m.s
				return pushScreen(pickState(s, func(st plane.State) {
					cmd := patchWith(s, it.ID, plane.WorkItemPatch{StateID: &st.ID}, "состояние: "+st.Name)
					if destructiveState(st) {
						cmd = confirm(s, "Перевести "+itoaKey(s, it.Sequence)+" в "+st.Name+"?", cmd)
					}
					pickLater(cmd)
				}))
			})},
		{ID: "item.priority", Title: "Сменить приоритет", Group: "задача", Keys: []string{"p"}, Hint: "приоритет", Scopes: cardScopes,
			Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
				s := m.s
				return pushScreen(pickPriority(s, func(p string) {
					pickLater(patchWith(s, it.ID, plane.WorkItemPatch{Priority: &p}, "приоритет: "+p))
				}))
			})},
		{ID: "item.assign", Title: "Назначить на себя", Group: "задача", Keys: []string{"a"}, Hint: "на себя", Scopes: cardScopes,
			Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
				s := m.s
				if slices.Contains(it.AssigneeIDs, s.me.ID) {
					s.setStatus("задача уже назначена на вас", false)
					return nil
				}
				ids := append(slices.Clone(it.AssigneeIDs), s.me.ID)
				return patchWith(s, it.ID, plane.WorkItemPatch{AssigneeIDs: &ids}, "назначено на вас")
			})},
		{ID: "item.open", Title: "Открыть в браузере", Group: "задача", Keys: []string{"O"}, Hint: "браузер", Scopes: cardScopes,
			Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
				s := m.s
				url := s.deps.App.CopyItems(*s.project, it).URL
				return func() tea.Msg {
					if err := s.deps.OpenURL(url); err != nil {
						return statusMsg{"браузер: " + err.Error(), true}
					}
					return statusMsg{"открыто: " + url, false}
				}
			})},
	} {
		registerAction(a)
	}
}
