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
		t.Fatal("окно не закрылось")
	}
	m, _ = customModel(t, config.Command{Name: "Quiet", Key: "ctrl+e", Command: "echo hidden", Output: "none"})
	drive(t, m, ctrlE())
	if m.s.status.text != "готово: Quiet" {
		t.Fatalf("%+v", m.s.status)
	}
}

func TestContractTUIV6CommandsConfirmAndPalette(t *testing.T) {
	m, e := customModel(t, config.Command{Name: "Zork", Key: "ctrl+e", Command: "touch ran; echo ok", Confirm: true})
	drive(t, m, ctrlE())
	if m.top().Scope() != "dialog" || !strings.Contains(view(m), "Zork") {
		t.Fatalf("нет диалога: %s", view(m))
	}
	pressD(t, m, "n")
	if m.s.status.text != "отменено" {
		t.Fatalf("%+v", m.s.status)
	}
	if _, err := os.Stat(filepath.Join(e.git.TopV, "ran")); err == nil {
		t.Fatal("после n команда запущена")
	}
	// палитра: команда под названием; y — запуск
	drive(t, m, tea.KeyMsg{Type: tea.KeyCtrlP})
	typeD(t, m, "Zork")
	pressD(t, m, "enter")
	if m.top().Scope() != "dialog" {
		t.Fatalf("палитра: %s", view(m))
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
			t.Errorf("BuiltinKeys без %q", k)
		}
	}
	cfg := &config.Config{Commands: []config.Command{{Name: "Bad", Key: "ctrl+p", Command: "true"}}}
	if err := cfg.CheckCommandKeys(keys); err == nil {
		t.Fatal("конфликт с ctrl+p не найден")
	}
	cfg.Commands[0].Key = "ctrl+e"
	if err := cfg.CheckCommandKeys(keys); err != nil {
		t.Fatalf("ctrl+e свободна: %v", err)
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
	if !strings.Contains(out, "Свои") || !strings.Contains(out, "Zork") || !strings.Contains(out, "Ctrl+e") {
		t.Fatalf("нет своих команд в справке:\n%s", out)
	}
	m, _ = customModel(t)
	if strings.Contains(newHelp("main").content(m.s), "Свои") {
		t.Fatal("раздел «Свои» без команд")
	}
}

// Esc в окне выполняющейся команды отменяет её (статус «отменено»), окно закрывается.
func TestCustomWindowEscCancels(t *testing.T) {
	m, e := customModel(t, config.Command{Name: "Slow", Key: "ctrl+e", Command: "sleep 5; touch done", Output: "window"})
	_, cmd := m.Update(ctrlE())
	batch, ok := cmd().(tea.BatchMsg)
	if !ok || len(batch) != 2 {
		t.Fatalf("ожидался batch из окна и запуска: %T", cmd())
	}
	m.Update(batch[0]()) // окно
	done := make(chan tea.Msg, 1)
	go func() { done <- batch[1]() }()
	if m.top().Scope() != "dialog" || !strings.Contains(view(m), "выполняется") {
		t.Fatalf("нет окна: %s", view(m))
	}
	pressD(t, m, "esc")
	if m.top().Scope() != "main" || m.s.status.text != "отменено" {
		t.Fatalf("scope=%s %+v", m.top().Scope(), m.s.status)
	}
	select {
	case msg := <-done:
		m.Update(msg) // результат отменённой команды окно не оживляет
	case <-time.After(3 * time.Second):
		t.Fatal("команда не отменена")
	}
	if m.top().Scope() != "main" || m.s.status.text != "отменено" {
		t.Fatalf("scope=%s %+v", m.top().Scope(), m.s.status)
	}
	if _, err := os.Stat(filepath.Join(e.git.TopV, "done")); err == nil {
		t.Fatal("команда дошла до конца")
	}
}

func TestCustomCancelOnExit(t *testing.T) {
	m, _ := customModel(t)
	m.s.cancel()
	if m.s.ctx.Err() == nil {
		t.Fatal("контекст не отменён")
	}
}
