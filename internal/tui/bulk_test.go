package tui

import (
	"strings"
	"testing"
)

func countCalls(e *testEnv, sub string) int {
	e.api.Mu.Lock()
	defer e.api.Mu.Unlock()
	n := 0
	for _, c := range e.api.Calls {
		if strings.Contains(c, sub) {
			n++
		}
	}
	return n
}

func TestBulkActions(t *testing.T) {
	m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, nil)
	m.s.marks["i1"], m.s.marks["i2"] = true, true
	before := countCalls(e, "UpdateWorkItem")
	pressD(t, m, "s")
	if m.top().Scope() != "dialog" {
		t.Fatalf("no state choice")
	}
	pressD(t, m, "down", "down", "enter", "y")
	if len(m.s.marks) != 0 || !strings.HasPrefix(m.s.status.text, "2 ok, 0 failed") {
		t.Fatalf("summary: %q marks=%v calls=%v", m.s.status.text, m.s.marks, e.api.Calls)
	}
	if n := countCalls(e, "UpdateWorkItem") - before; n != 2 {
		t.Fatalf("patches %d: %v", n, e.api.Calls)
	}
	// assign to me: i2 is not mine yet
	m.s.marks["i2"], m.s.marks["i3"] = true, true
	pressD(t, m, "a", "y")
	if !strings.HasPrefix(m.s.status.text, "") || !strings.Contains(m.s.status.text, "ok") {
		t.Fatalf("a: %q", m.s.status.text)
	}
	// label
	m.s.marks["i4"] = true
	pressD(t, m, "L")
	pressD(t, m, "enter", "y")
	if !strings.Contains(m.s.status.text, "1 ok") {
		t.Fatalf("l: %q", m.s.status.text)
	}
	// clearing the selection
	m.s.marks["i4"] = true
	pressD(t, m, "L", "esc")
	if m.top().Scope() != "main" || len(m.s.marks) != 1 {
		t.Fatalf("esc")
	}
	// palette with a summary of marked items
	pressD(t, m, "ctrl+p")
	if !strings.Contains(view(m), "(marked)") {
		t.Fatalf("palette without bulk actions:\n%s", view(m))
	}
}
