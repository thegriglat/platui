package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"testing"
	"time"

	"github.com/thegriglat/platui/internal/config"
)

// confirmFlow: действие запускает диалог; Enter и любая клавиша = нет (вызовов нет), y = да.
func confirmFlow(t *testing.T, name string, setup func(m *model, e *testEnv), act []string, calls string, want int) {
	t.Helper()
	for _, no := range []string{"enter", "n", "esc", "x"} {
		m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, nil)
		setup(m, e)
		before := countCalls(e, calls)
		pressD(t, m, act...)
		if m.top().Scope() != "dialog" || !strings.Contains(view(m), "y — да") {
			t.Fatalf("%s: нет диалога:\n%s", name, view(m))
		}
		pressD(t, m, no)
		if m.top().Scope() == "dialog" || countCalls(e, calls) != before || !strings.Contains(m.s.status.text, "отменено") {
			t.Fatalf("%s/%s: вызов без y: %q %v", name, no, m.s.status.text, e.api.Calls)
		}
	}
	for _, yes := range []string{"y", "Y"} {
		m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, nil)
		setup(m, e)
		before := countCalls(e, calls)
		pressD(t, m, act...)
		pressD(t, m, yes)
		if n := countCalls(e, calls) - before; n != want {
			t.Fatalf("%s/%s: вызовов %d, нужно %d: %v", name, yes, n, want, e.api.Calls)
		}
	}
}

func TestContractTUIConfirmState(t *testing.T) {
	// i1 в Backlog; Done — 4-я строка пикера (Backlog, To Do, In Progress, Done)
	confirmFlow(t, "s", func(m *model, e *testEnv) { m.s.cursor = "i1" },
		[]string{"s", "down", "down", "down", "enter"}, "UpdateWorkItem", 1)
	// cancelled — тоже
	confirmFlow(t, "s-cancelled", func(m *model, e *testEnv) { m.s.cursor = "i1" },
		[]string{"s", "end", "enter"}, "UpdateWorkItem", 1)
	// started — без диалога
	m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, nil)
	m.s.cursor = "i1"
	pressD(t, m, "s", "down", "down", "enter")
	if m.top().Scope() == "dialog" || countCalls(e, "UpdateWorkItem") != 1 {
		t.Fatalf("started с диалогом: %v", e.api.Calls)
	}
}

func TestContractTUIConfirmKanban(t *testing.T) {
	// i9 — в Done(s4); i7 в In Progress → вправо в Done
	confirmFlow(t, "kanban", func(m *model, e *testEnv) {
		pressD(t, m, "v")
		k := m.top().(*kanban)
		k.sel = "i6"
		k.sync(m.s)
	}, []string{"shift+right"}, "UpdateWorkItem", 1)
	m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, nil)
	pressD(t, m, "v")
	pressD(t, m, "shift+right")
	if m.top().Scope() == "dialog" || countCalls(e, "UpdateWorkItem") != 1 {
		t.Fatalf("перенос в To Do с диалогом: %v", e.api.Calls)
	}
}

func TestContractTUIConfirmBulk(t *testing.T) {
	two := func(m *model, e *testEnv) { m.s.marks["i1"], m.s.marks["i2"] = true, true }
	confirmFlow(t, "bulk.s", two, []string{"s", "down", "down", "enter"}, "UpdateWorkItem", 2)
	confirmFlow(t, "bulk.s-done", two, []string{"s", "down", "down", "down", "enter"}, "UpdateWorkItem", 2)
	confirmFlow(t, "bulk.a", two, []string{"a"}, "UpdateWorkItem", 1)
	confirmFlow(t, "bulk.l", two, []string{"L", "enter"}, "UpdateWorkItem", 2)
	// отмена сохраняет отметки
	m, _ := newTestModelCfg(t, StartAt{Project: "PROJ"}, nil)
	two(m, nil)
	pressD(t, m, "a", "n")
	if len(m.s.marks) != 2 {
		t.Fatalf("отметки потеряны: %v", m.s.marks)
	}
}

func TestContractTUIConfirmDescription(t *testing.T) {
	syncExec(t)
	edit := func(c *config.Config) { c.Editor = `printf 'new text\n' >` }
	setup := func(m *model, e *testEnv) {
		m.s.cursor = "i1"
		// в Plane описание изменилось после загрузки
		e.api.ItemsV["p1"][0].UpdatedAt = t0.Add(time.Hour)
	}
	// без конфликта — диалога нет
	m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, edit)
	m.s.cursor = "i1"
	pressD(t, m, "e")
	if m.top().Scope() == "dialog" || countCalls(e, "UpdateWorkItem") != 1 {
		t.Fatalf("без конфликта: %v", e.api.Calls)
	}
	for _, no := range []string{"enter", "n"} {
		m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, edit)
		setup(m, e)
		pressD(t, m, "e")
		if m.top().Scope() != "dialog" || !strings.Contains(view(m), "Описание изменилось") {
			t.Fatalf("нет диалога:\n%s", view(m))
		}
		pressD(t, m, no)
		if countCalls(e, "UpdateWorkItem") != 0 || m.top().Scope() == "dialog" {
			t.Fatalf("запись без y: %v", e.api.Calls)
		}
	}
	m, e = newTestModelCfg(t, StartAt{Project: "PROJ"}, edit)
	setup(m, e)
	pressD(t, m, "e", "y")
	if countCalls(e, "UpdateWorkItem") != 1 {
		t.Fatalf("после y: %v", e.api.Calls)
	}
}

func TestContractTUIConfirm(t *testing.T) {
	m, _ := newTestModelCfg(t, StartAt{Project: "PROJ"}, nil)
	called := 0
	feed(t, m, pushScreenMsg{&confirmScreen{question: "Q?", yes: func() tea.Cmd { called++; return nil }}})
	if m.top().Scope() != "dialog" || !m.top().CapturesInput() || !strings.Contains(view(m), "Q?") {
		t.Fatal("диалог")
	}
	pressD(t, m, "enter")
	if called != 0 || m.top().Scope() != "main" {
		t.Fatal("enter = нет")
	}
}
