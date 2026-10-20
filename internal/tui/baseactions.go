package tui

import tea "github.com/charmbracelet/bubbletea"

func init() {
	main := []string{"main"}
	for _, a := range []action{
		{ID: "app.quit", Title: "Quit", Group: "navigation", Keys: []string{"ctrl+c"},
			Run: func(*model) tea.Cmd { return tea.Quit }},
		{ID: "nav.move", Title: "Cursor: line, page, start/end", Group: "navigation",
			Keys: []string{"up", "down", "pgup", "pgdown", "home", "end"}, Hint: "↑↓|move", Scopes: main,
			Run: func(m *model) tea.Cmd { return m.forward() }},
		{ID: "nav.tab.prev", Title: "Previous tab", Group: "navigation", Keys: []string{"ctrl+left"},
			Hint: "Ctrl+←→|tab", Scopes: main, Run: func(m *model) tea.Cmd { m.s.setTab(m.s.tab - 1); return nil }},
		{ID: "nav.tab.next", Title: "Next tab", Group: "navigation", Keys: []string{"ctrl+right"},
			Scopes: main, Run: func(m *model) tea.Cmd { m.s.setTab(m.s.tab + 1); return nil }},
		{ID: "nav.tab.n", Title: "Tab by number (\"All\" is 1)", Group: "navigation",
			Keys:   []string{"alt+1", "alt+2", "alt+3", "alt+4", "alt+5", "alt+6", "alt+7", "alt+8", "alt+9"},
			Scopes: main, Run: func(m *model) tea.Cmd {
				if n := int(m.lastKey.String()[4] - '1'); n < len(m.s.tabList()) {
					m.s.setTab(n)
				}
				return nil
			}},
		{ID: "nav.focus", Title: "Focus: table/preview", Group: "navigation", Keys: []string{"tab", "shift+tab"},
			Scopes: main, Run: func(m *model) tea.Cmd {
				if _, r := m.s.paneWidths(); r > 0 {
					m.s.focus = 1 - m.s.focus
				}
				return nil
			}},
		{ID: "nav.back", Title: "Back / close", Group: "navigation", Keys: []string{"esc"},
			Scopes: []string{"projects", "kanban", "item", "help", "todo"}, Run: func(m *model) tea.Cmd { m.pop(); return nil }},
		{ID: "nav.quit", Title: "Quit / back", Group: "navigation", Keys: []string{"q"}, Hint: "quit",
			Run: func(m *model) tea.Cmd {
				if len(m.stack) > 1 {
					m.pop()
					return nil
				}
				return tea.Quit
			}},
		{ID: "nav.projects", Title: "Projects", Group: "navigation", Keys: []string{"P"}, Hint: "project",
			Scopes: []string{"main", "kanban", "item"}, Run: func(m *model) tea.Cmd {
				if newProjects == nil {
					m.s.setStatus("projects screen unavailable", true)
					return nil
				}
				return pushScreen(newProjects(m.s))
			}},
		{ID: "view.mine", Title: "My tasks only", Group: "views", Keys: []string{"m"}, Hint: "mine",
			Scopes: []string{"main", "kanban"}, Run: func(m *model) tea.Cmd { m.s.mine = !m.s.mine; return nil }},
		{ID: "view.kanban", Title: "Kanban", Group: "views", Keys: []string{"v"}, Hint: "kanban", Scopes: main,
			Run: func(m *model) tea.Cmd {
				if newKanban == nil {
					m.s.setStatus("kanban view unavailable", true)
					return nil
				}
				return pushScreen(newKanban(m.s))
			}},
		{ID: "view.reload", Title: "Refresh from API", Group: "views", Keys: []string{"R"}, Hint: "refresh",
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
		{ID: "view.help", Title: "Help", Group: "views", Keys: []string{"?", "h"}, Hint: "help",
			Scopes: []string{"main", "kanban", "item", "projects", "todo"}, Run: func(m *model) tea.Cmd {
				return pushScreen(newHelp(m.top().Scope()))
			}},
	} {
		registerAction(a)
	}
}
