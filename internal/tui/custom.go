package tui

import (
	"context"
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/runner"
)

// BuiltinKeys returns all keys of the built-in action registry (deduplicated, in order);
// main.go checks custom command keys against them before starting the TUI (C4 v6).
func BuiltinKeys() []string {
	var out []string
	for _, a := range registry {
		for _, k := range a.Keys {
			if !slices.Contains(out, k) {
				out = append(out, k)
			}
		}
	}
	return out
}

// customFor returns the custom command bound to a key in the screen scope (built-in actions take priority).
func (s *session) customFor(scope, name string) (config.Command, bool) {
	if !slices.Contains(cardScopes, scope) {
		return config.Command{}, false
	}
	for _, c := range s.deps.Config.Commands {
		if c.Key != "" && c.Key == name {
			return c, true
		}
	}
	return config.Command{}, false
}

// customTarget returns the task under the cursor or the open card.
func (s *session) customTarget(scope string) (plane.WorkItem, bool) {
	id := s.cursor
	if scope == "item" && s.openItem != "" {
		id = s.openItem
	}
	it, ok := s.itemByID(id)
	if !ok || s.project == nil {
		s.setStatus("no task", true)
		return it, false
	}
	return it, true
}

// runCustom: placeholders, then confirm (when confirm: true), then run.
func runCustom(s *session, scope string, c config.Command) tea.Cmd {
	it, ok := s.customTarget(scope)
	if !ok {
		return nil
	}
	v := s.deps.App.CommandVars(*s.project, it)
	start := func() tea.Cmd { return execCustom(s, c, v) }
	if !c.Confirm {
		return start()
	}
	return confirmWith("Run \""+c.Name+"\": "+app.RenderCommand(c.Command, v)+"?", start)
}

// customDoneMsg is the result of an output=window command for its own output window.
type customDoneMsg struct {
	win *outputScreen
	res runner.Result
	err error
}

func execCustom(s *session, c config.Command, v app.CommandVars) tea.Cmd {
	a := s.deps.App
	s.setStatus("running: "+c.Name+"…", false)
	if c.Output == config.OutputWindow {
		ctx, cancel := context.WithCancel(s.ctx)
		win := newRunningOutput(c.Name, cancel)
		run := func() tea.Msg {
			res, err := a.RunCommand(ctx, c, v)
			return customDoneMsg{win, res, err}
		}
		return tea.Batch(pushScreen(win), run)
	}
	return func() tea.Msg {
		res, err := a.RunCommand(s.ctx, c, v)
		if err != nil {
			return statusMsg{c.Name + ": " + err.Error(), true}
		}
		failed := res.ExitCode != 0
		if c.Output == config.OutputNone {
			return statusMsg{doneText(c.Name, res.ExitCode), failed}
		}
		if l := lastLine(res.Stdout, res.Stderr); l != "" {
			if failed {
				l = fmt.Sprintf("%s (code %d)", l, res.ExitCode)
			}
			return statusMsg{l, failed}
		}
		return statusMsg{doneText(c.Name, res.ExitCode), failed}
	}
}

func doneText(name string, code int) string {
	if code != 0 {
		return fmt.Sprintf("error: %s (code %d)", name, code)
	}
	return "done: " + name
}

// lastLine returns the last non-empty line of stdout, otherwise of stderr.
func lastLine(outs ...string) string {
	for _, o := range outs {
		lines := strings.Split(strings.TrimRight(o, "\n\r "), "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			if l := strings.TrimSpace(lines[i]); l != "" {
				return l
			}
		}
	}
	return ""
}

// customEntries returns custom commands for the palette (under the title).
func customEntries(s *session, under string, closeThen func(tea.Cmd) tea.Cmd) []palEntry {
	if !slices.Contains(cardScopes, under) {
		return nil
	}
	var out []palEntry
	for _, c := range s.deps.Config.Commands {
		out = append(out, palEntry{kind: "command", title: c.Name, keys: keysLabel(nonEmpty(c.Key)),
			run: func() tea.Cmd { return closeThen(runCustom(s, under, c)) }})
	}
	return out
}

func nonEmpty(k string) []string {
	if k == "" {
		return nil
	}
	return []string{k}
}
