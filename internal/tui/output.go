package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// outputScreen is a window with command output (output: window); Esc/q close it,
// while the command runs Esc cancels it.
type outputScreen struct {
	title, body string
	vp          viewport.Model
	cancel      context.CancelFunc // non-nil while the command is running
}

func newOutputVP() viewport.Model {
	vp := viewport.New(0, 0)
	vp.KeyMap = viewport.KeyMap{
		PageDown: key.NewBinding(key.WithKeys("pgdown")), PageUp: key.NewBinding(key.WithKeys("pgup")),
		Down: key.NewBinding(key.WithKeys("down")), Up: key.NewBinding(key.WithKeys("up")),
	}
	return vp
}

// newRunningOutput is a window for a command that is still running.
func newRunningOutput(name string, cancel context.CancelFunc) *outputScreen {
	return &outputScreen{title: name, body: "running… (Esc to cancel)", vp: newOutputVP(), cancel: cancel}
}

func newOutputScreen(name, stdout, stderr string, code int) *outputScreen {
	o := &outputScreen{vp: newOutputVP()}
	o.finish(name, stdout, stderr, code)
	return o
}

// finish fills the window with the command result.
func (o *outputScreen) finish(name, stdout, stderr string, code int) {
	body := stdout
	if stderr != "" {
		body += "\n" + stderr
	}
	if body = strings.TrimRight(body, "\n"); body == "" {
		body = "(no output)"
	}
	o.title, o.body, o.cancel = name, body, nil
	if code != 0 {
		o.title += fmt.Sprintf(" — exit %d", code)
	}
}

func (o *outputScreen) Title() string       { return o.title }
func (o *outputScreen) Scope() string       { return "dialog" }
func (o *outputScreen) CapturesInput() bool { return true }

func (o *outputScreen) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case customDoneMsg:
		if msg.win != o || o.cancel == nil {
			return o, nil
		}
		if msg.err != nil {
			o.finish(o.title, "", msg.err.Error(), 1)
			return o, nil
		}
		o.finish(o.title, msg.res.Stdout, msg.res.Stderr, msg.res.ExitCode)
	case tea.KeyMsg:
		switch keyName(msg) {
		case "esc", "q":
			if o.cancel != nil {
				o.cancel()
				o.cancel = nil
				s.setStatus("cancelled", false)
			}
			return o, popScreen
		case "home":
			o.vp.GotoTop()
			return o, nil
		case "end":
			o.vp.GotoBottom()
			return o, nil
		}
	}
	var cmd tea.Cmd
	o.vp, cmd = o.vp.Update(msg)
	return o, cmd
}

func (o *outputScreen) View(s *session) string {
	o.vp.Width, o.vp.Height = s.width, max(s.height-2, 1)
	o.vp.SetContent(o.body)
	return s.st.accent.Bold(true).Render(" "+o.title) + "\n" + o.vp.View() + "\n" +
		s.st.muted.Render(" ↑↓ scroll   Esc, q close")
}
