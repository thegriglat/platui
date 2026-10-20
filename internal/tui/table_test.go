package tui

import (
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/forge"
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
)

// pump runs a command; tea.Sequence (unexported sequenceMsg) is expanded in order.
func pump(t *testing.T, m *model, cmd tea.Cmd) {
	t.Helper()
	for _, r := range run(t, cmd) {
		if v := reflect.ValueOf(r); v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeOf(cmd) {
			for i := 0; i < v.Len(); i++ {
				pump(t, m, v.Index(i).Interface().(tea.Cmd))
			}
			continue
		}
		_, c := m.Update(r)
		pump(t, m, c)
	}
}

func send(t *testing.T, m *model, msg tea.Msg) {
	t.Helper()
	_, c := m.Update(msg)
	pump(t, m, c)
}

func keys(t *testing.T, m *model, ks ...string) {
	t.Helper()
	for _, k := range ks {
		send(t, m, keyMsg(k))
	}
}

func typeText(t *testing.T, m *model, s string) {
	t.Helper()
	for _, r := range s {
		if r == ' ' {
			send(t, m, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
			continue
		}
		keys(t, m, string(r))
	}
}

func TestContractTUITable(t *testing.T) {
	m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, func(c *config.Config) {
		c.Theme.NoColor = true
		c.Filters = []config.Filter{{Name: "Urgent", Priorities: []string{"urgent"}}}
	})
	s := m.s
	e.api.ItemsV["p1"][0].Name = "Alpha"
	// table: columns and branch marker
	s.branches = map[string]gitbranch.BranchStatus{"i5": {Local: true, Ahead: 2, Remote: true}}
	feed(t, m, tea.WindowSizeMsg{Width: 200, Height: 30}) // with the preview (PT-5c) the table is narrower, the "Asg" column is hidden
	v := view(m)
	for _, w := range []string{"ID", "Title", "Pri", "Branch", "PROJ-1", "● ↑2 ☁", "ME"} {
		if !strings.Contains(v, w) {
			t.Errorf("table lacks %q:\n%s", w, v)
		}
	}
	// navigation and Enter
	keys(t, m, "down", "down")
	if s.cursor != s.visible()[2].ID {
		t.Errorf("cursor %s", s.cursor)
	}
	// space — mark
	id := s.cursor
	keys(t, m, "space")
	if !s.marks[id] || s.cursor == id {
		t.Errorf("mark: %v cursor=%s", s.marks, s.cursor)
	}
	keys(t, m, "up", "space")
	if s.marks[id] {
		t.Error("unmark")
	}
	// sorting
	keys(t, m, "o")
	if s.sortBy != sortPriority || !strings.Contains(view(m), "Pri▲") {
		t.Errorf("sort %d\n%s", s.sortBy, view(m))
	}
	keys(t, m, "o", "o", "o", "o")
	if s.sortBy != sortAPI {
		t.Errorf("sort cycle %d", s.sortBy)
	}
	// filters
	keys(t, m, "F", "down", "enter")
	if s.filter != 1 || s.tabCount(0) != 2 {
		t.Errorf("filter=%d count=%d", s.filter, s.tabCount(0))
	}
	keys(t, m, "alt+0")
	if s.filter != 0 {
		t.Error("filter reset")
	}
	// search as you type: title only, no API calls
	calls := len(e.api.Calls)
	keys(t, m, "/")
	typeText(t, m, "login")
	if got := s.visible(); len(got) != 1 || got[0].Name != "Fix login page" {
		t.Fatalf("visible %v", got)
	}
	if s.tabCount(0) != 1 || s.tabCount(2) != 1 || s.tabCount(1) != 0 {
		t.Errorf("counts %d %d %d", s.tabCount(0), s.tabCount(1), s.tabCount(2))
	}
	if v := view(m); !strings.Contains(v, "Fix [login] page") || !strings.Contains(v, "found 1") {
		t.Errorf("search view:\n%s", v)
	}
	if len(e.api.Calls) != calls {
		t.Errorf("API calls during search: %v", e.api.Calls[calls:])
	}
	keys(t, m, "esc")
	if s.query != "" || len(s.visible()) == 0 || s.searching {
		t.Errorf("esc: %q", s.query)
	}
	// title only: a word from the description does not match
	keys(t, m, "/")
	typeText(t, m, "desc")
	if len(s.visible()) != 0 {
		t.Errorf("description matched")
	}
	keys(t, m, "esc")
	// the 33rd rune is not accepted
	keys(t, m, "/")
	typeText(t, m, strings.Repeat("a", 40))
	if n := len([]rune(s.query)); n != 32 {
		t.Errorf("query len %d", n)
	}
	keys(t, m, "esc")
	// Enter on a non-key: the filter stays
	keys(t, m, "/")
	typeText(t, m, "number 3")
	keys(t, m, "enter")
	if s.searching || s.query != "number 3" || len(s.visible()) != 1 {
		t.Errorf("enter: %q %d", s.query, len(s.visible()))
	}
	keys(t, m, "esc") // not in search: Esc breaks nothing
	s.setQuery("")
	// key query: the filter does not change
	keys(t, m, "/")
	typeText(t, m, "PROJ-5")
	if len(s.visible()) != 11 || !strings.Contains(view(m), "Enter — open PROJ-5") {
		t.Errorf("key query:\n%s", view(m))
	}
	keys(t, m, "enter")
	if s.query != "" {
		t.Errorf("after key: %q", s.query)
	}
	if s.cursor != "i5" {
		t.Errorf("cursor %s", s.cursor)
	}
	// Ctrl+G to another project → WorkItemByKey
	send(t, m, tea.KeyMsg{Type: tea.KeyCtrlG})
	typeText(t, m, "OTH-7")
	keys(t, m, "enter")
	if e.count("WorkItemByKey") == 0 || s.project.Identifier != "OTH" {
		t.Errorf("goto other: calls=%v project=%v", e.api.Calls, s.project)
	}
	// not found
	send(t, m, tea.KeyMsg{Type: tea.KeyCtrlG})
	typeText(t, m, "OTH-99")
	keys(t, m, "enter")
	if !strings.Contains(s.status.text, "not found") {
		t.Errorf("status %q", s.status.text)
	}
	// projects screen
	keys(t, m, "P")
	if m.top().Scope() != "projects" || !strings.Contains(view(m), "One") {
		t.Fatalf("projects:\n%s", view(m))
	}
	keys(t, m, "up", "enter")
	if s.project.Identifier != "PROJ" || m.top().Scope() != "main" {
		t.Errorf("project %v scope %s", s.project, m.top().Scope())
	}
}

func TestTableLayoutNarrow(t *testing.T) {
	m, _, _, _ := newTestModel(t)
	for _, w := range []int{40, 60, 80, 100, 160} {
		feed(t, m, tea.WindowSizeMsg{Width: w, Height: 20})
		for i, l := range strings.Split(view(m), "\n") {
			if lipglossWidth(l) > w {
				t.Errorf("w=%d line %d too wide (%d)", w, i, lipglossWidth(l))
			}
		}
	}
	feed(t, m, tea.WindowSizeMsg{Width: 50, Height: 20})
	if v := view(m); strings.Contains(v, "Asg") || strings.Contains(v, "State") {
		t.Errorf("columns not hidden:\n%s", v)
	}
	feed(t, m, tea.WindowSizeMsg{Width: 99, Height: 20})
	if v := view(m); !strings.Contains(v, "State") {
		t.Errorf("state hidden at 99:\n%s", v)
	}
}

func TestInitialsAndFresh(t *testing.T) {
	for in, want := range map[string]string{"Anna Lee": "AL", "anna": "AN", "": "", "a.b": "AB"} {
		if got := initials(in); got != want {
			t.Errorf("initials(%q)=%q want %q", in, got, want)
		}
	}
	m, _, _, _ := newTestModel(t)
	m.s.fresh = map[string]time.Time{"i1": t0}
	if !strings.Contains(view(m), "+  PROJ-1") {
		t.Errorf("fresh not marked:\n%s", view(m))
	}
	_ = plane.WorkItem{}
}

func TestMRSign(t *testing.T) {
	for want, mrs := range map[string][]forge.MR{
		"":  nil,
		"⇄": {{State: "merged"}, {State: "opened"}},
		"✓": {{State: "closed"}, {State: "merged"}},
	} {
		if got := mrSign(mrs); got != want {
			t.Errorf("mrSign(%v)=%q want %q", mrs, got, want)
		}
	}
	m, _, _, _ := newTestModel(t)
	if v := view(m); !strings.Contains(v, "MR") {
		t.Errorf("no MR column:\n%s", v)
	}
}

func TestSpaceInPreviewFocus(t *testing.T) {
	old := newPreview
	defer func() { newPreview = old }()
	newPreview = func(*session) pane { return &fakePane{} }
	m, _, _, _ := newTestModel(t)
	before := m.s.cursor
	keys(t, m, "tab", "space")
	if m.s.focus != 1 || len(m.s.marks) != 0 || m.s.cursor != before {
		t.Errorf("focus=%d marks=%v cursor=%s", m.s.focus, m.s.marks, m.s.cursor)
	}
}
