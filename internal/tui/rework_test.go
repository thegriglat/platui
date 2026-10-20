package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
)

func TestFocusResetWhenPreviewHidden(t *testing.T) {
	old := newPreview
	defer func() { newPreview = old }()
	fp := &fakePane{}
	newPreview = func(*session) pane { return fp }
	m, _, _, _ := newTestModel(t)
	press(t, m, "tab")
	feed(t, m, tea.WindowSizeMsg{Width: 80, Height: 20})
	if m.s.focus != 0 {
		t.Fatal("focus stuck on hidden preview")
	}
	press(t, m, "down")
	if len(fp.keys) != 0 {
		t.Fatal("key sent to hidden preview")
	}
}

func TestProjectsAction(t *testing.T) {
	old := newProjects
	newProjects = nil
	defer func() { newProjects = old }()
	m, _, _, _ := newTestModel(t)
	press(t, m, "P")
	if !m.s.status.isErr || len(m.stack) != 1 {
		t.Fatalf("nil newProjects: %+v", m.s.status)
	}
	newProjects = func(*session) screen { return newHelp("projects") }
	press(t, m, "P")
	if len(m.stack) != 2 {
		t.Fatal("projects not pushed")
	}
}

func TestStubEmptyAndNilTable(t *testing.T) {
	old := newTable
	defer func() { newTable = old }()
	newTable = nil
	m, _, _, _ := newTestModel(t)
	m.s.setQuery("zzzzqq")
	if !strings.Contains(view(m), "no data") {
		t.Fatalf("no placeholder:\n%s", view(m))
	}
}

func TestResetProjectClearsPolling(t *testing.T) {
	m, _, _, _ := newTestModel(t)
	m.s.polling = true
	m.s.resetProject(plane.Project{ID: "p2", Identifier: "OTH"})
	if m.s.polling {
		t.Fatal("polling kept")
	}
}

func TestReloadOpenItem(t *testing.T) {
	old := newItemScreen
	defer func() { newItemScreen = old }()
	newItemScreen = func(*session, plane.WorkItem) screen { return &itemStub{} }
	m, e := newTestModelCfg(t, StartAt{Item: "PROJ-5"}, nil)
	if m.s.openItem != "i5" || m.top().Scope() != "item" {
		t.Fatalf("openItem=%q", m.s.openItem)
	}
	n1, n2 := e.count("WorkItem:"), e.count("Comments:")
	press(t, m, "R")
	if e.count("WorkItem:") != n1+1 || e.count("Comments:") != n2+1 {
		t.Fatalf("calls: %v", e.api.Calls)
	}
	press(t, m, "esc")
	if m.s.openItem != "" {
		t.Fatal("openItem kept after pop")
	}
}

type itemStub struct{}

func (i *itemStub) Update(*session, tea.Msg) (screen, tea.Cmd) { return i, nil }
func (*itemStub) View(*session) string                         { return "item" }
func (*itemStub) Title() string                                { return "item" }
func (*itemStub) Scope() string                                { return "item" }
func (*itemStub) CapturesInput() bool                          { return false }

func TestPatchItem(t *testing.T) {
	m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, nil)
	st := "s4"
	feed(t, m, mustMsg(t, patchItem(m.s, "i1", plane.WorkItemPatch{StateID: &st})))
	if it, _ := m.s.itemByID("i1"); it.StateID != "s4" || e.count("UpdateWorkItem") != 1 {
		t.Fatalf("patch: %+v calls=%v", it, e.api.Calls)
	}
	e.api.Mu.Lock()
	e.api.Err["UpdateWorkItem"] = errors.New("denied")
	e.api.Mu.Unlock()
	st2 := "s4"
	feed(t, m, mustMsg(t, patchItem(m.s, "i2", plane.WorkItemPatch{StateID: &st2})))
	if it, _ := m.s.itemByID("i2"); it.StateID == "s4" || !m.s.status.isErr || !strings.Contains(m.s.status.text, "denied") {
		t.Fatalf("error path: %+v %+v", it, m.s.status)
	}
}

func mustMsg(t *testing.T, c tea.Cmd) tea.Msg {
	t.Helper()
	ms := run(t, c)
	if len(ms) != 1 {
		t.Fatalf("msgs: %v", ms)
	}
	return ms[0]
}
