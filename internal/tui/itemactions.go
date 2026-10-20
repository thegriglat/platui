package tui

import (
	"context"
	"slices"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
)

var cardScopes = []string{"main", "kanban", "item"}

// target returns the task the action works on: the open card or the task under the cursor.
func (m *model) target() (plane.WorkItem, bool) {
	id := m.s.cursor
	if m.top().Scope() == "item" && m.s.openItem != "" {
		id = m.s.openItem
	}
	it, ok := m.s.itemByID(id)
	if !ok || m.s.project == nil {
		m.s.setStatus("no task", true)
		return it, false
	}
	return it, true
}

// patchWith applies a patch (app.Update), yielding an itemMsg and a success message in the status.
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

// withTarget wraps an action: Run gets the task or does nothing.
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
		{ID: "item.state", Title: "Change state", Group: "task", Keys: []string{"s"}, Hint: "state", Scopes: cardScopes,
			Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
				s := m.s
				return pushScreen(pickState(s, func(st plane.State) {
					cmd := patchWith(s, it.ID, plane.WorkItemPatch{StateID: &st.ID}, "state: "+st.Name)
					if destructiveState(st) {
						cmd = confirm(s, "Move "+itoaKey(s, it.Sequence)+" to "+st.Name+"?", cmd)
					}
					pickLater(cmd)
				}))
			})},
		{ID: "item.priority", Title: "Change priority", Group: "task", Keys: []string{"p"}, Hint: "priority", Scopes: cardScopes,
			Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
				s := m.s
				return pushScreen(pickPriority(s, func(p string) {
					pickLater(patchWith(s, it.ID, plane.WorkItemPatch{Priority: &p}, "priority: "+p))
				}))
			})},
		{ID: "item.assign", Title: "Assign to me", Group: "task", Keys: []string{"a"}, Hint: "assign", Scopes: cardScopes,
			Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
				s := m.s
				if slices.Contains(it.AssigneeIDs, s.me.ID) {
					s.setStatus("task is already assigned to you", false)
					return nil
				}
				ids := append(slices.Clone(it.AssigneeIDs), s.me.ID)
				return patchWith(s, it.ID, plane.WorkItemPatch{AssigneeIDs: &ids}, "assigned to you")
			})},
		{ID: "item.open", Title: "Open in browser", Group: "task", Keys: []string{"O"}, Hint: "browser", Scopes: cardScopes,
			Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
				s := m.s
				url := s.deps.App.CopyItems(*s.project, it).URL
				return func() tea.Msg {
					if err := s.deps.OpenURL(url); err != nil {
						return statusMsg{"browser: " + err.Error(), true}
					}
					return statusMsg{"opened: " + url, false}
				}
			})},
	} {
		registerAction(a)
	}
}
