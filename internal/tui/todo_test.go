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
	// добавление
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
		t.Error("нет пункта в экране")
	}
	// отметка и перестановка (курсор на последнем пункте)
	gpress(t, m, "space")
	gfeed(t, m, tea.KeyMsg{Type: tea.KeyShiftUp})
	if got := fileOf(t, path); got != "- [ ] first\n- [x] third\n- [ ] second PROJ-3\n" {
		t.Fatalf("file=%q", got)
	}
	gfeed(t, m, tea.KeyMsg{Type: tea.KeyShiftDown})
	// правка
	gpress(t, m, "e")
	typeText(t, m, "!")
	gpress(t, m, "enter")
	if got := fileOf(t, path); !strings.Contains(got, "- [x] third!\n") {
		t.Fatalf("file=%q", got)
	}
	// удаление: не y — отмена, y — удаляет
	gpress(t, m, "d", "n")
	if !strings.Contains(fileOf(t, path), "third!") || m.top().Scope() != "todo" {
		t.Fatalf("удалено без подтверждения: %s %q", m.top().Scope(), fileOf(t, path))
	}
	gpress(t, m, "d", "y")
	if got := fileOf(t, path); got != "- [ ] first\n- [ ] second PROJ-3\n" {
		t.Fatalf("file=%q", got)
	}
	// ErrChanged: файл правят снаружи
	if err := os.WriteFile(path, []byte("# my\n- [ ] first\n- [ ] second PROJ-3\n- [ ] ext\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gpress(t, m, "space")
	if m.s.status.text != todoChangedStatus || !m.s.status.isErr {
		t.Fatalf("status=%q", m.s.status.text)
	}
	if got := fileOf(t, path); strings.Contains(got, "[x]") || !strings.Contains(got, "ext") {
		t.Fatalf("изменение применено: %q", got)
	}
	if n := len(m.top().(*todoScreen).list.Items()); n != 3 {
		t.Fatalf("не перечитан: %d", n)
	}
	// в задачу Plane
	gpress(t, m, "home", "C")
	if got := fileOf(t, path); !strings.Contains(got, "- [ ] first (PROJ-13)\n") {
		t.Fatalf("file=%q status=%q", got, m.s.status.text)
	}
	// переход по ключу
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
	if m.top().Scope() != "help" || !strings.Contains(view(m), "Новый пункт") {
		t.Fatalf("справка todo не открыта: %s", m.top().Scope())
	}
	gpress(t, m, "esc")
	if m.top().Scope() != "todo" {
		t.Fatalf("scope=%s", m.top().Scope())
	}
	gpress(t, m, "h", "esc")
	gpress(t, m, "q")
	if m.top().Scope() == "todo" {
		t.Fatal("q не закрыл todo")
	}
}
