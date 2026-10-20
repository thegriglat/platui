package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/todo"
)

func init() {
	t := []string{"todo"}
	registerAction(action{ID: "view.todo", Title: "Личный todo", Group: "виды", Keys: []string{"T"},
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
	// Справочные записи: клавиши обрабатывает сам экран (Run == nil).
	for _, a := range []action{
		{ID: "todo.nav", Title: "Курсор", Keys: []string{"up", "down", "pgup", "pgdown", "home", "end"}, Hint: "↑↓|ход"},
		{ID: "todo.new", Title: "Новый пункт", Keys: []string{"n"}, Hint: "новый"},
		{ID: "todo.done", Title: "Выполнен / нет", Keys: []string{"space"}, Hint: "отметка"},
		{ID: "todo.edit", Title: "Править пункт", Keys: []string{"e"}, Hint: "правка"},
		{ID: "todo.delete", Title: "Удалить пункт (с подтверждением)", Keys: []string{"d"}, Hint: "удалить"},
		{ID: "todo.move", Title: "Переставить пункт", Keys: []string{"shift+up", "shift+down"}, Hint: "Shift+↑↓|порядок"},
		{ID: "todo.task", Title: "Сделать задачей Plane", Keys: []string{"C"}, Hint: "в задачу"},
		{ID: "todo.open", Title: "Перейти к задаче из пункта", Keys: []string{"enter"}, Hint: "к задаче"},
	} {
		a.Group, a.Scopes = "todo", t
		registerAction(a)
	}
}
