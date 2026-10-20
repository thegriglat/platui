package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func special(t *testing.T, m *model, k tea.KeyType) {
	t.Helper()
	feed(t, m, tea.KeyMsg{Type: k})
}

func stateOf(m *model, id string) string {
	it, _ := m.s.itemByID(id)
	return it.StateID
}

func TestContractTUIKanban(t *testing.T) {
	m, api, _, _ := newTestModel(t)
	press(t, m, "v")
	if len(m.stack) != 2 || m.top().Scope() != "kanban" {
		t.Fatalf("kanban not opened")
	}
	out := view(m)
	for _, want := range []string{"Backlog 2", "To Do 3", "In Progress 3", "Done 3", "Task number 1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Cancelled") || strings.Contains(out, "All") {
		t.Fatalf("unexpected column:\n%s", out)
	}
	k := m.top().(*kanban)
	if k.sel != "i1" {
		t.Fatalf("sel %q", k.sel)
	}
	special(t, m, tea.KeyRight)
	special(t, m, tea.KeyDown)
	if k.col != 1 || k.sel != "i4" {
		t.Fatalf("nav col=%d sel=%q", k.col, k.sel)
	}
	special(t, m, tea.KeyShiftRight)
	if stateOf(m, "i4") != "s3" || k.col != 2 || k.sel != "i4" {
		t.Fatalf("move: state=%s col=%d", stateOf(m, "i4"), k.col)
	}
	if !strings.Contains(strings.Join(api.Calls, "\n"), "UpdateWorkItem") {
		t.Fatalf("no patch call: %v", api.Calls)
	}
	special(t, m, tea.KeyShiftLeft)
	if stateOf(m, "i4") != "s2" {
		t.Fatal("move back")
	}
	special(t, m, tea.KeyShiftLeft)
	special(t, m, tea.KeyShiftLeft)
	special(t, m, tea.KeyShiftLeft) // from the first column — nowhere to go
	if stateOf(m, "i4") != "s1" {
		t.Fatalf("edge: %s", stateOf(m, "i4"))
	}
	press(t, m, "v")
	if len(m.stack) != 1 {
		t.Fatal("v did not return")
	}
	press(t, m, "v", "esc")
	if len(m.stack) != 1 {
		t.Fatal("esc did not return")
	}
}

func TestKanbanMoveRollback(t *testing.T) {
	m, api, _, _ := newTestModel(t)
	press(t, m, "v")
	api.Err["UpdateWorkItem"] = errors.New("boom")
	special(t, m, tea.KeyShiftRight)
	if stateOf(m, "i1") != "s1" {
		t.Fatalf("not rolled back: %s", stateOf(m, "i1"))
	}
	if !m.s.status.isErr || !strings.Contains(m.s.status.text, "boom") {
		t.Fatalf("status %+v", m.s.status)
	}
	if k := m.top().(*kanban); k.col != 0 || k.sel != "i1" {
		t.Fatalf("selection col=%d sel=%q", k.col, k.sel)
	}
}

func TestKanbanNarrowScrollAndOpen(t *testing.T) {
	m, _, _, _ := newTestModel(t)
	feed(t, m, tea.WindowSizeMsg{Width: 50, Height: 12})
	press(t, m, "v")
	if out := view(m); strings.Contains(out, "Done") || !strings.Contains(out, "Backlog") {
		t.Fatalf("narrow:\n%s", out)
	}
	for range 3 {
		special(t, m, tea.KeyRight)
	}
	if out := view(m); !strings.Contains(out, "Done") || strings.Contains(out, "Backlog") {
		t.Fatalf("scrolled:\n%s", out)
	}
	special(t, m, tea.KeyEnd)
	special(t, m, tea.KeyHome)
	special(t, m, tea.KeyPgDown)
	special(t, m, tea.KeyPgUp)
	special(t, m, tea.KeyEnter)
	if !m.s.status.isErr { // no PT-5c yet — "card unavailable"
		if len(m.stack) < 3 {
			t.Fatal("enter did nothing")
		}
	}
}

func TestKanbanEmptyAndMine(t *testing.T) {
	m, _, _, _ := newTestModel(t)
	press(t, m, "v", "m")
	if !m.s.mine || !strings.Contains(view(m), "[Mine]") {
		t.Fatal("mine")
	}
	m.s.setQuery("zzzzqq")
	_ = view(m)
	special(t, m, tea.KeyShiftRight)
	m.s.tabs = nil
	if !strings.Contains(view(m), "no data") {
		t.Fatal("empty")
	}
}
