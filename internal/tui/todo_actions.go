package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/todo"
)

func init() {
	t := []string{"todo"}
	registerAction(action{ID: "view.todo", Title: "Personal todo", Group: "views", Keys: []string{"T"},
		Hint: "todo", Scopes: []string{"main"}, Run: func(m *model) tea.Cmd {
			path, err := m.s.deps.Config.TodoPath()
			if err != nil {
				m.s.setStatus("todo: "+err.Error(), true)
				return nil
			}
			l, err := todo.Load(path)
			if err != nil {
				m.s.setStatus("todo: "+err.Error(), true)
				return nil
			}
			return pushScreen(newTodoScreen(path, l))
		}})
	// Reference entries: the keys are handled by the screen itself (Run == nil).
	for _, a := range []action{
		{ID: "todo.nav", Title: "Cursor", Keys: []string{"up", "down", "pgup", "pgdown", "home", "end"}, Hint: "↑↓|move"},
		{ID: "todo.new", Title: "New item", Keys: []string{"n"}, Hint: "new"},
		{ID: "todo.done", Title: "Done / not done", Keys: []string{"space"}, Hint: "toggle"},
		{ID: "todo.edit", Title: "Edit item", Keys: []string{"e"}, Hint: "edit"},
		{ID: "todo.delete", Title: "Delete item (with confirmation)", Keys: []string{"d"}, Hint: "delete"},
		{ID: "todo.move", Title: "Reorder item", Keys: []string{"shift+up", "shift+down"}, Hint: "Shift+↑↓|order"},
		{ID: "todo.task", Title: "Make a Plane task", Keys: []string{"C"}, Hint: "to task"},
		{ID: "todo.open", Title: "Go to the task from the item", Keys: []string{"enter"}, Hint: "to task"},
	} {
		a.Group, a.Scopes = "todo", t
		registerAction(a)
	}
}
