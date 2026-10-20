package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
)

// screen is a screen in the model stack.
type screen interface {
	Update(s *session, msg tea.Msg) (screen, tea.Cmd)
	View(s *session) string
	Title() string
	Scope() string       // "projects"|"main"|"kanban"|"item"|"dialog"|"palette"|"help"
	CapturesInput() bool // text input: global keys other than Ctrl+C are not intercepted
}

// pane is a main screen pane.
type pane interface {
	Update(s *session, msg tea.Msg) (pane, tea.Cmd)
	View(s *session, width, height int) string
}

// Hook points of subtasks PT-5b…5f (assigned in init() of their files).
var (
	newTable   func(s *session) pane   = newStubTable // PT-5b
	newPreview func(s *session) pane                  // PT-5c; nil — no preview is shown
	newKanban  func(s *session) screen                // PT-5f; nil — `v` reports "no view"
	// PT-5a additions (do not change the contract). All nil-safe.
	newProjects   func(s *session) screen                           // PT-5b: projects screen
	newItemScreen func(s *session, it plane.WorkItem) screen        // PT-5c: full card
	pickState     func(s *session, onPick func(plane.State)) screen // PT-5c
	pickPriority  func(s *session, onPick func(string)) screen      // PT-5c
	pickLabel     func(s *session, onPick func(plane.Label)) screen // PT-5c
)

type pushScreenMsg struct{ screen screen }
type popScreenMsg struct{}

func pushScreen(sc screen) tea.Cmd { return func() tea.Msg { return pushScreenMsg{sc} } }
func popScreen() tea.Msg           { return popScreenMsg{} }

// openItem opens the full card; without PT-5c the status is "card unavailable".
func openItem(s *session, it plane.WorkItem) tea.Cmd {
	if newItemScreen == nil {
		s.setStatus("card unavailable", true)
		return nil
	}
	s.hist.visit(it.ID)
	s.cursor, s.openItem = it.ID, it.ID
	return pushScreen(newItemScreen(s, it))
}

// openItemMsg is the message that opens a card (for Sequence); nil if the card is unavailable.
func openItemMsg(s *session, it plane.WorkItem) tea.Msg {
	if cmd := openItem(s, it); cmd != nil {
		return cmd()
	}
	return nil
}
