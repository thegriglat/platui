package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// inputScreen is a modal input line (PT-5d): new task, branch name.
type inputScreen struct {
	title, hint string
	text        []rune
	busy        bool
	onSubmit    func(c *inputScreen) tea.Cmd              // Enter
	onKey       func(c *inputScreen, key string) tea.Cmd  // extra keys (nil: none)
	onMsg       func(c *inputScreen, msg tea.Msg) tea.Cmd // other messages (nil: none)
	extra       func() string                             // extra line under the input

	// Suggestions under the input (similar tasks): sim is recomputed when the text changes;
	// ↓ moves into the list, Enter on a row calls onPick instead of onSubmit.
	similar func(text string) []string
	onPick  func(c *inputScreen, i int) tea.Cmd
	sim     []string
	simCur  int // -1: focus is in the input line
}

func (c *inputScreen) refreshSim() {
	c.simCur = -1
	if c.similar != nil {
		c.sim = c.similar(string(c.text))
	}
}

func (c *inputScreen) Title() string       { return c.title }
func (c *inputScreen) Scope() string       { return "dialog" }
func (c *inputScreen) CapturesInput() bool { return true }

func (c *inputScreen) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		if c.onMsg != nil {
			return c, c.onMsg(c, msg)
		}
		return c, nil
	}
	name := keyName(k)
	if c.busy {
		return c, nil
	}
	if c.onKey != nil && strings.Contains(name, "+") {
		if cmd := c.onKey(c, name); cmd != nil {
			return c, cmd
		}
	}
	switch k.Type {
	case tea.KeyEsc:
		return c, popScreen
	case tea.KeyDown:
		if c.simCur+1 < len(c.sim) {
			c.simCur++
		}
	case tea.KeyUp:
		c.simCur = max(c.simCur-1, -1)
	case tea.KeyEnter:
		if c.simCur >= 0 && c.simCur < len(c.sim) && c.onPick != nil {
			return c, c.onPick(c, c.simCur)
		}
		return c, c.onSubmit(c)
	case tea.KeyBackspace:
		if len(c.text) > 0 {
			c.text = c.text[:len(c.text)-1]
		}
		c.refreshSim()
	case tea.KeySpace:
		c.text = append(c.text, ' ')
		c.refreshSim()
	case tea.KeyRunes:
		if !k.Alt {
			c.text = append(c.text, k.Runes...)
			c.refreshSim()
		}
	}
	return c, nil
}

func (c *inputScreen) View(s *session) string {
	rows := []string{s.st.accent.Bold(true).Render(" " + c.title), " > " + string(c.text) + "▏"}
	if c.extra != nil {
		if e := c.extra(); e != "" {
			rows = append(rows, s.st.muted.Render(" "+e))
		}
	}
	if len(c.sim) > 0 {
		rows = append(rows, s.st.muted.Render(" Similar tasks (↓ select, Enter open):"))
		for i, l := range c.sim {
			if i == c.simCur {
				rows = append(rows, s.st.sel.Render(clip("  > "+l, s.width)))
			} else {
				rows = append(rows, s.st.muted.Render(clip("    "+l, s.width)))
			}
		}
	}
	for len(rows) < s.height-2 {
		rows = append(rows, "")
	}
	rows = append(rows, s.st.muted.Render(" "+c.hint), s.statusView(s.width))
	return clip2(rows, s.height)
}
