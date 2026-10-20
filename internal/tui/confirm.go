package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
)

// confirmScreen is a y/N dialog: only y/Y confirms, anything else cancels.
type confirmScreen struct {
	question string
	yes      func() tea.Cmd // called synchronously in Update on confirmation
	no       func() tea.Cmd // if non-nil, called instead of "cancelled" on refusal
}

// confirm opens a confirmation dialog for a destructive action; onYes runs only after y/Y.
func confirm(s *session, question string, onYes tea.Cmd) tea.Cmd {
	return confirmWith(question, func() tea.Cmd { return onYes })
}

// confirmWith is like confirm, but the action is evaluated on confirmation (with side effects in the session).
func confirmWith(question string, yes func() tea.Cmd) tea.Cmd {
	return pushScreen(&confirmScreen{question: question, yes: yes})
}

func (c *confirmScreen) Title() string       { return "confirmation" }
func (c *confirmScreen) Scope() string       { return "dialog" }
func (c *confirmScreen) CapturesInput() bool { return true }

func (c *confirmScreen) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return c, nil
	}
	if n := keyName(k); n == "y" || n == "Y" {
		return c, tea.Sequence(popScreen, c.yes())
	}
	if c.no != nil {
		return c, tea.Sequence(popScreen, c.no())
	}
	return c, tea.Sequence(popScreen, func() tea.Msg { return statusMsg{"cancelled", false} })
}

func (c *confirmScreen) View(s *session) string {
	head, rest, _ := strings.Cut(c.question, "\n") // the first line is the question, the rest are details
	rows := []string{s.st.warn.Bold(true).Render(" " + head)}
	for _, l := range strings.Split(rest, "\n") {
		if l != "" {
			rows = append(rows, "  "+l)
		}
	}
	rows = append(rows, "", s.st.muted.Render(" y — yes   any other key — no (default)"))
	for len(rows) < s.height-1 {
		rows = append(rows, "")
	}
	return strings.Join(append(rows, s.statusView(s.width)), "\n")
}

// destructiveState reports that moving to completed/cancelled requires confirmation.
func destructiveState(st plane.State) bool { return st.Group == "completed" || st.Group == "cancelled" }
