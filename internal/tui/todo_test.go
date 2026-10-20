package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/todo"
)

func fileOf(t *testing.T, p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestContractTUIV7Todo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "todo.md")
	m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, func(c *config.Config) { c.Todo.Path = path })
	_ = e
	gpress(t, m, "T")
	if m.top().Scope() != "todo" {
		t.Fatalf("scope=%s", m.top().Scope())
	}
	// add
	gpress(t, m, "n")
	typeText(t, m, "first")
	gpress(t, m, "enter", "n")
	typeText(t, m, "second PROJ-3")
	gpress(t, m, "enter", "n")
	typeText(t, m, "third")
	gpress(t, m, "enter")
	if got := fileOf(t, path); got != "- [ ] first\n- [ ] second PROJ-3\n- [ ] third\n" {
		t.Fatalf("file=%q", got)
	}
	if !strings.Contains(view(m), "second PROJ-3") {
		t.Error("item missing on screen")
	}
	// mark and reorder (cursor on the last item)
	gpress(t, m, "space")
	gfeed(t, m, tea.KeyMsg{Type: tea.KeyShiftUp})
	if got := fileOf(t, path); got != "- [ ] first\n- [x] third\n- [ ] second PROJ-3\n" {
		t.Fatalf("file=%q", got)
	}
	gfeed(t, m, tea.KeyMsg{Type: tea.KeyShiftDown})
	// edit
	gpress(t, m, "e")
	typeText(t, m, "!")
	gpress(t, m, "enter")
	if got := fileOf(t, path); !strings.Contains(got, "- [x] third!\n") {
		t.Fatalf("file=%q", got)
	}
	// delete: not y — cancel, y — deletes
	gpress(t, m, "d", "n")
	if !strings.Contains(fileOf(t, path), "third!") || m.top().Scope() != "todo" {
		t.Fatalf("deleted without confirmation: %s %q", m.top().Scope(), fileOf(t, path))
	}
	gpress(t, m, "d", "y")
	if got := fileOf(t, path); got != "- [ ] first\n- [ ] second PROJ-3\n" {
		t.Fatalf("file=%q", got)
	}
	// ErrChanged: the file is edited externally
	if err := os.WriteFile(path, []byte("# my\n- [ ] first\n- [ ] second PROJ-3\n- [ ] ext\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gpress(t, m, "space")
	if m.s.status.text != todoChangedStatus || !m.s.status.isErr {
		t.Fatalf("status=%q", m.s.status.text)
	}
	if got := fileOf(t, path); strings.Contains(got, "[x]") || !strings.Contains(got, "ext") {
		t.Fatalf("change applied: %q", got)
	}
	if n := len(m.top().(*todoScreen).list.Items()); n != 3 {
		t.Fatalf("not re-read: %d", n)
	}
	// into a Plane task
	gpress(t, m, "home", "C")
	if got := fileOf(t, path); !strings.Contains(got, "- [ ] first (PROJ-13)\n") {
		t.Fatalf("file=%q status=%q", got, m.s.status.text)
	}
	// jump by key
	gpress(t, m, "down", "enter")
	if m.top().Scope() != "item" || m.s.openItem != "i3" {
		t.Fatalf("top=%s open=%s", m.top().Scope(), m.s.openItem)
	}
	gpress(t, m, "esc")
	if m.top().Scope() != "todo" {
		t.Fatalf("scope=%s", m.top().Scope())
	}
	gpress(t, m, "esc")
	if m.top().Scope() != "main" {
		t.Fatalf("scope=%s", m.top().Scope())
	}
	l, _ := todo.Load(path)
	if len(l.Items()) != 3 {
		t.Fatal("items")
	}
}

func gfeed(t *testing.T, m *model, msg tea.Msg) {
	t.Helper()
	_, cmd := m.Update(msg)
	gdrain(t, m, cmd)
}

func TestTodoHelpAndBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "todo.md")
	m, _ := newTestModelCfg(t, StartAt{Project: "PROJ"}, func(c *config.Config) { c.Todo.Path = path })
	gpress(t, m, "T")
	gpress(t, m, "?")
	if m.top().Scope() != "help" || !strings.Contains(view(m), "New item") {
		t.Fatalf("todo help not opened: %s", m.top().Scope())
	}
	gpress(t, m, "esc")
	if m.top().Scope() != "todo" {
		t.Fatalf("scope=%s", m.top().Scope())
	}
	gpress(t, m, "h", "esc")
	gpress(t, m, "q")
	if m.top().Scope() == "todo" {
		t.Fatal("q did not close todo")
	}
}
