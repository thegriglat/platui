package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/config"
)

func customModel(t *testing.T, cmds ...config.Command) (*model, *testEnv) {
	t.Helper()
	m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, func(c *config.Config) { c.Commands = cmds })
	e.git.TopV = t.TempDir()
	m.s.cursor = "i1"
	return m, e
}

func ctrlE() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyCtrlE} }

func TestContractTUIV6CommandsStatus(t *testing.T) {
	m, _ := customModel(t, config.Command{Name: "Echo", Key: "ctrl+e", Command: "echo hi {key}; echo last {title}", Output: "status"})
	drive(t, m, ctrlE())
	if m.s.status.text != "last Task number 1" || m.s.status.isErr {
		t.Fatalf("%+v", m.s.status)
	}
	m, _ = customModel(t, config.Command{Name: "Fail", Key: "ctrl+e", Command: "echo boom; exit 3", Output: "status"})
	drive(t, m, ctrlE())
	if !m.s.status.isErr || !strings.Contains(m.s.status.text, "boom") {
		t.Fatalf("%+v", m.s.status)
	}
}

func TestContractTUIV6CommandsWindowNone(t *testing.T) {
	m, _ := customModel(t, config.Command{Name: "Win", Key: "ctrl+e", Command: "echo out1; echo out2", Output: "window"})
	drive(t, m, ctrlE())
	if m.top().Scope() != "dialog" || !strings.Contains(view(m), "out2") {
		t.Fatalf("%s", view(m))
	}
	pressD(t, m, "esc")
	if m.top().Scope() != "main" {
		t.Fatal("window did not close")
	}
	m, _ = customModel(t, config.Command{Name: "Quiet", Key: "ctrl+e", Command: "echo hidden", Output: "none"})
	drive(t, m, ctrlE())
	if m.s.status.text != "done: Quiet" {
		t.Fatalf("%+v", m.s.status)
	}
}

func TestContractTUIV6CommandsConfirmAndPalette(t *testing.T) {
	m, e := customModel(t, config.Command{Name: "Zork", Key: "ctrl+e", Command: "touch ran; echo ok", Confirm: true})
	drive(t, m, ctrlE())
	if m.top().Scope() != "dialog" || !strings.Contains(view(m), "Zork") {
		t.Fatalf("no dialog: %s", view(m))
	}
	pressD(t, m, "n")
	if m.s.status.text != "cancelled" {
		t.Fatalf("%+v", m.s.status)
	}
	if _, err := os.Stat(filepath.Join(e.git.TopV, "ran")); err == nil {
		t.Fatal("command ran after n")
	}
	// palette: command under its title; y runs it
	drive(t, m, tea.KeyMsg{Type: tea.KeyCtrlP})
	typeD(t, m, "Zork")
	pressD(t, m, "enter")
	if m.top().Scope() != "dialog" {
		t.Fatalf("palette: %s", view(m))
	}
	pressD(t, m, "y")
	if m.s.status.text != "ok" {
		t.Fatalf("%+v", m.s.status)
	}
}

func TestContractTUIV6CommandsBuiltinKeys(t *testing.T) {
	keys := BuiltinKeys()
	for _, k := range []string{"q", "ctrl+p", "R", "ctrl+c"} {
		found := false
		for _, x := range keys {
			found = found || x == k
		}
		if !found {
			t.Errorf("BuiltinKeys lacks %q", k)
		}
	}
	cfg := &config.Config{Commands: []config.Command{{Name: "Bad", Key: "ctrl+p", Command: "true"}}}
	if err := cfg.CheckCommandKeys(keys); err == nil {
		t.Fatal("conflict with ctrl+p not found")
	}
	cfg.Commands[0].Key = "ctrl+e"
	if err := cfg.CheckCommandKeys(keys); err != nil {
		t.Fatalf("ctrl+e is free: %v", err)
	}
}

func TestContractTUIV6Commands(t *testing.T) {
	m, _ := customModel(t, config.Command{Name: "Echo", Key: "ctrl+e", Command: "echo {id}"})
	drive(t, m, ctrlE())
	if m.s.status.text != "i1" {
		t.Fatalf("%+v", m.s.status)
	}
}

func TestCustomInHelp(t *testing.T) {
	m, _ := customModel(t, config.Command{Name: "Zork", Key: "ctrl+e", Command: "true"})
	h := newHelp("main")
	out := h.content(m.s)
	if !strings.Contains(out, "Custom") || !strings.Contains(out, "Zork") || !strings.Contains(out, "Ctrl+e") {
		t.Fatalf("no custom commands in help:\n%s", out)
	}
	m, _ = customModel(t)
	if strings.Contains(newHelp("main").content(m.s), "Custom") {
		t.Fatal("\"Custom\" section without commands")
	}
}

// Esc in the window of a running command cancels it (status "cancelled") and closes the window.
func TestCustomWindowEscCancels(t *testing.T) {
	m, e := customModel(t, config.Command{Name: "Slow", Key: "ctrl+e", Command: "sleep 5; touch done", Output: "window"})
	_, cmd := m.Update(ctrlE())
	batch, ok := cmd().(tea.BatchMsg)
	if !ok || len(batch) != 2 {
		t.Fatalf("expected a batch of window and run: %T", cmd())
	}
	m.Update(batch[0]()) // window
	done := make(chan tea.Msg, 1)
	go func() { done <- batch[1]() }()
	if m.top().Scope() != "dialog" || !strings.Contains(view(m), "running") {
		t.Fatalf("no window: %s", view(m))
	}
	pressD(t, m, "esc")
	if m.top().Scope() != "main" || m.s.status.text != "cancelled" {
		t.Fatalf("scope=%s %+v", m.top().Scope(), m.s.status)
	}
	select {
	case msg := <-done:
		m.Update(msg) // the result of a cancelled command does not revive the window
	case <-time.After(3 * time.Second):
		t.Fatal("command not cancelled")
	}
	if m.top().Scope() != "main" || m.s.status.text != "cancelled" {
		t.Fatalf("scope=%s %+v", m.top().Scope(), m.s.status)
	}
	if _, err := os.Stat(filepath.Join(e.git.TopV, "done")); err == nil {
		t.Fatal("command ran to completion")
	}
}

func TestCustomCancelOnExit(t *testing.T) {
	m, _ := customModel(t)
	m.s.cancel()
	if m.s.ctx.Err() == nil {
		t.Fatal("context not cancelled")
	}
}
