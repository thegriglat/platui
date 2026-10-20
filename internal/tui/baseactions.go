package tui

import tea "github.com/charmbracelet/bubbletea"

func init() {
	main := []string{"main"}
	for _, a := range []action{
		{ID: "app.quit", Title: "Выход", Group: "навигация", Keys: []string{"ctrl+c"},
			Run: func(*model) tea.Cmd { return tea.Quit }},
		{ID: "nav.move", Title: "Курсор: строка, страница, начало/конец", Group: "навигация",
			Keys: []string{"up", "down", "pgup", "pgdown", "home", "end"}, Hint: "↑↓|ход", Scopes: main,
			Run: func(m *model) tea.Cmd { return m.forward() }},
		{ID: "nav.tab.prev", Title: "Предыдущий таб", Group: "навигация", Keys: []string{"ctrl+left"},
			Hint: "Ctrl+←→|таб", Scopes: main, Run: func(m *model) tea.Cmd { m.s.setTab(m.s.tab - 1); return nil }},
		{ID: "nav.tab.next", Title: "Следующий таб", Group: "навигация", Keys: []string{"ctrl+right"},
			Scopes: main, Run: func(m *model) tea.Cmd { m.s.setTab(m.s.tab + 1); return nil }},
		{ID: "nav.tab.n", Title: "Таб по номеру («Все» — 1)", Group: "навигация",
			Keys:   []string{"alt+1", "alt+2", "alt+3", "alt+4", "alt+5", "alt+6", "alt+7", "alt+8", "alt+9"},
			Scopes: main, Run: func(m *model) tea.Cmd {
				if n := int(m.lastKey.String()[4] - '1'); n < len(m.s.tabList()) {
					m.s.setTab(n)
				}
				return nil
			}},
		{ID: "nav.focus", Title: "Фокус: таблица/превью", Group: "навигация", Keys: []string{"tab", "shift+tab"},
			Scopes: main, Run: func(m *model) tea.Cmd {
				if _, r := m.s.paneWidths(); r > 0 {
					m.s.focus = 1 - m.s.focus
				}
				return nil
			}},
		{ID: "nav.back", Title: "Назад / закрыть", Group: "навигация", Keys: []string{"esc"},
			Scopes: []string{"projects", "kanban", "item", "help", "todo"}, Run: func(m *model) tea.Cmd { m.pop(); return nil }},
		{ID: "nav.quit", Title: "Выход / назад", Group: "навигация", Keys: []string{"q"}, Hint: "выход",
			Run: func(m *model) tea.Cmd {
				if len(m.stack) > 1 {
					m.pop()
					return nil
				}
				return tea.Quit
			}},
		{ID: "nav.projects", Title: "Проекты", Group: "навигация", Keys: []string{"P"}, Hint: "проект",
			Scopes: []string{"main", "kanban", "item"}, Run: func(m *model) tea.Cmd {
				if newProjects == nil {
					m.s.setStatus("экран проектов недоступен", true)
					return nil
				}
				return pushScreen(newProjects(m.s))
			}},
		{ID: "view.mine", Title: "Только мои задачи", Group: "виды", Keys: []string{"m"}, Hint: "мои",
			Scopes: []string{"main", "kanban"}, Run: func(m *model) tea.Cmd { m.s.mine = !m.s.mine; return nil }},
		{ID: "view.kanban", Title: "Канбан", Group: "виды", Keys: []string{"v"}, Hint: "канбан", Scopes: main,
			Run: func(m *model) tea.Cmd {
				if newKanban == nil {
					m.s.setStatus("вид «канбан» недоступен", true)
					return nil
				}
				return pushScreen(newKanban(m.s))
			}},
		{ID: "view.reload", Title: "Обновить из API", Group: "виды", Keys: []string{"R"}, Hint: "обновить",
			Scopes: []string{"main", "kanban", "item"}, Run: func(m *model) tea.Cmd {
				if m.s.project == nil {
					return nil
				}
				m.s.loading = true
				if id := m.s.openItem; id != "" {
					return tea.Batch(reload(m.s), loadItem(m.s, m.s.project.ID, id), loadComments(m.s, m.s.project.ID, id))
				}
				return reload(m.s)
			}},
		{ID: "view.help", Title: "Справка", Group: "виды", Keys: []string{"?", "h"}, Hint: "справка",
			Scopes: []string{"main", "kanban", "item", "projects", "todo"}, Run: func(m *model) tea.Cmd {
				return pushScreen(newHelp(m.top().Scope()))
			}},
	} {
		registerAction(a)
	}
}
