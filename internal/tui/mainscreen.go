package tui

import tea "github.com/charmbracelet/bubbletea"

// mainScreen is the main screen: table and preview.
type mainScreen struct {
	table, preview pane
	last           clickMark // last click on a row (double click)
}

func newMainScreen(s *session) *mainScreen {
	mk := newTable
	if mk == nil {
		mk = newStubTable
	}
	ms := &mainScreen{table: mk(s)}
	if newPreview != nil {
		ms.preview = newPreview(s)
	}
	return ms
}

func (ms *mainScreen) Title() string       { return "platui" }
func (ms *mainScreen) Scope() string       { return "main" }
func (ms *mainScreen) CapturesInput() bool { return false }

func (ms *mainScreen) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	var c1, c2 tea.Cmd
	if m, ok := msg.(tea.MouseMsg); ok {
		return ms, ms.mouse(s, m)
	}
	if k, ok := msg.(tea.KeyMsg); ok {
		if s.panelKey(keyName(k)) {
			return ms, nil
		}
		if _, r := s.paneWidths(); (s.focus == 1 || s.panel.zoom) && ms.preview != nil && r > 0 {
			ms.preview, c2 = ms.preview.Update(s, msg)
		} else {
			ms.table, c1 = ms.table.Update(s, msg)
		}
		return ms, tea.Batch(c1, c2)
	}
	ms.table, c1 = ms.table.Update(s, msg)
	if ms.preview != nil {
		ms.preview, c2 = ms.preview.Update(s, msg)
	}
	return ms, tea.Batch(c1, c2)
}

func (ms *mainScreen) View(s *session) string {
	var prev func(w, h int) string
	if ms.preview != nil {
		prev = func(w, h int) string { return ms.preview.View(s, w, h) }
	}
	return s.frame("main", func(w, h int) string { return ms.table.View(s, w, h) }, prev)
}
