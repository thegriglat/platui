package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	for _, a := range []action{
		{ID: "view.filters", Title: "Saved filters", Group: "views", Keys: []string{"F"}, Hint: "filters",
			Scopes: []string{"main"}, Run: func(m *model) tea.Cmd {
				if len(m.s.deps.Config.Filters) == 0 {
					m.s.setStatus("no saved filters (config: filters)", false)
					return nil
				}
				return pushScreen(&filterMenu{list: cursorList{n: len(m.s.deps.Config.Filters) + 1, h: 12, pos: m.s.filter}})
			}},
		{ID: "view.filter.reset", Title: "Reset filter", Group: "views", Keys: []string{"alt+0"},
			Scopes: []string{"main"}, Run: func(m *model) tea.Cmd { m.s.filter = 0; return nil }},
	} {
		registerAction(a)
	}
}

// filterMenu picks a saved filter (0 means no filter).
type filterMenu struct{ list cursorList }

func (f *filterMenu) Title() string       { return "filters" }
func (f *filterMenu) Scope() string       { return "dialog" }
func (f *filterMenu) CapturesInput() bool { return false }

func (f *filterMenu) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return f, nil
	}
	switch name := keyName(k); name {
	case "esc", "q":
		return f, popScreen
	case "enter":
		s.filter = f.list.Pos()
		return f, popScreen
	default:
		f.list.Key(name)
	}
	return f, nil
}

func (f *filterMenu) View(s *session) string {
	lines := []string{s.st.accent.Bold(true).Render(" Filters"), ""}
	names := []string{"— no filter —"}
	for _, fl := range s.deps.Config.Filters {
		names = append(names, fl.Name)
	}
	f.list.SetHeight(max(s.height-5, 1))
	from, to := f.list.Range()
	for i := from; i < to; i++ {
		line := "  " + names[i]
		if i == s.filter {
			line += "  ✓"
		}
		if i == f.list.Pos() {
			line = s.st.sel.Render(clip(line, s.width))
		}
		lines = append(lines, line)
	}
	for len(lines) < s.height-2 {
		lines = append(lines, "")
	}
	lines = append(lines, s.st.muted.Render(" ↑↓ choose  Enter apply  Esc cancel"), s.statusView(s.width))
	return strings.Join(lines[:min(len(lines), s.height)], "\n")
}
