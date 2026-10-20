package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/plane"
)

func TestActionsUnique(t *testing.T) {
	if err := checkActions(registry); err != nil {
		t.Fatal(err)
	}
	mustPanic := func(name string, a action) {
		defer func() {
			if recover() == nil {
				t.Errorf("%s: no panic", name)
			}
		}()
		registerAction(a)
	}
	mustPanic("dup id", action{ID: "view.mine", Keys: []string{"F12"}})
	mustPanic("key conflict", action{ID: "x.conflict", Keys: []string{"m"}, Scopes: []string{"main"}})
	if err := checkActions([]action{{ID: "a", Keys: []string{"s"}}, {ID: "b", Keys: []string{"s"}, Bulk: true}}); err != nil {
		t.Errorf("bulk pair: %v", err)
	}
	if err := checkActions([]action{{ID: "a", Keys: []string{"s"}, Scopes: []string{"main"}}, {ID: "b", Keys: []string{"s"}, Scopes: []string{"kanban"}}}); err != nil {
		t.Errorf("different scopes: %v", err)
	}
}

func TestContractTUIBase(t *testing.T) {
	m, api, _, _ := newTestModel(t)
	s := m.s
	if s.project == nil || s.project.Identifier != "PROJ" || len(s.items) != 12 || s.me.ID != "u-me" {
		t.Fatalf("start: %+v items=%d", s.project, len(s.items))
	}
	if got := len(s.tabList()); got != 5 { // All + 4 (cancelled hidden)
		t.Fatalf("tabs=%d", got)
	}
	counts := []int{}
	for i := range s.tabList() {
		counts = append(counts, s.tabCount(i))
	}
	if want := []int{11, 2, 3, 3, 3}; !equal(counts, want) {
		t.Fatalf("counts %v want %v", counts, want)
	}
	v := view(m)
	for _, w := range []string{"PROJ", "All 11", "Backlog 2", "In Progress 3", "Task number 1", "help", "quit"} {
		if !strings.Contains(v, w) {
			t.Errorf("view lacks %q:\n%s", w, v)
		}
	}
	// tabs: Ctrl+→, Alt+n
	press(t, m, "ctrl+right")
	if s.tab != 1 || len(s.visible()) != 2 {
		t.Fatalf("tab=%d", s.tab)
	}
	press(t, m, "alt+4")
	if s.tab != 3 {
		t.Fatalf("alt+4 → %d", s.tab)
	}
	press(t, m, "alt+9", "ctrl+left")
	if s.tab != 2 {
		t.Fatalf("tab=%d", s.tab)
	}
	press(t, m, "alt+1")
	// "Mine"
	press(t, m, "m")
	if !s.mine || s.tabCount(0) != 6 || !strings.Contains(view(m), "[Mine]") {
		t.Fatalf("mine: count=%d", s.tabCount(0))
	}
	press(t, m, "m")
	// help
	press(t, m, "?")
	if m.top().Scope() != "help" || !strings.Contains(view(m), "Alt+1…9") {
		t.Fatalf("help:\n%s", view(m))
	}
	press(t, m, "end") // help scrolls: "views" is at the end
	if !strings.Contains(view(m), "Refresh from API") {
		t.Fatalf("help end:\n%s", view(m))
	}
	press(t, m, "h")
	if m.top().Scope() != "main" {
		t.Fatal("help not closed by h")
	}
	press(t, m, "?", "esc")
	if m.top().Scope() != "main" {
		t.Fatal("help not closed by esc")
	}
	// R: reload bypassing the cache
	api.Mu.Lock()
	api.ItemsV["p1"] = api.ItemsV["p1"][:10]
	api.Mu.Unlock()
	press(t, m, "R")
	if len(s.items) != 10 || !strings.Contains(s.status.text, "updated: 10 tasks, 18:40") || s.loading {
		t.Fatalf("reload: %d %q loading=%v", len(s.items), s.status.text, s.loading)
	}
	// q quits
	if cmd := m.key(keyMsg("q")); cmd == nil || cmd() != tea.Quit() {
		t.Error("q must quit")
	}
	if cmd := m.key(keyMsg("ctrl+c")); cmd == nil {
		t.Error("ctrl+c must quit")
	}
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestReloadError(t *testing.T) {
	m, api, _, _ := newTestModel(t)
	api.Mu.Lock()
	api.Err["WorkItems"] = errors.New("boom")
	api.Mu.Unlock()
	press(t, m, "R")
	if !m.s.status.isErr || !strings.Contains(m.s.status.text, "boom") || len(m.s.items) != 12 || m.s.loading {
		t.Fatalf("status %+v items=%d", m.s.status, len(m.s.items))
	}
}

func TestNilHooksSafe(t *testing.T) {
	oldTable, oldPreview, oldKanban, oldProjects := newTable, newPreview, newKanban, newProjects
	oldItem, oldState, oldPrio, oldLabel := newItemScreen, pickState, pickPriority, pickLabel
	t.Cleanup(func() {
		newTable, newPreview, newKanban, newProjects = oldTable, oldPreview, oldKanban, oldProjects
		newItemScreen, pickState, pickPriority, pickLabel = oldItem, oldState, oldPrio, oldLabel
	})
	newTable, newPreview, newKanban, newProjects = nil, nil, nil, nil
	newItemScreen, pickState, pickPriority, pickLabel = nil, nil, nil, nil
	m, _, _, _ := newTestModel(t)
	press(t, m, "v")
	if !m.s.status.isErr || m.top().Scope() != "main" {
		t.Fatalf("kanban nil: %+v", m.s.status)
	}
	if cmd := openItem(m.s, m.s.items[0]); cmd != nil || m.s.status.text != "card unavailable" {
		t.Fatalf("openItem: %+v", m.s.status)
	}
	press(t, m, "tab", "up", "down", "pgdown", "end", "home", "x")
}

func TestVisibleFilters(t *testing.T) {
	m, _, _, _ := newTestModel(t)
	s := m.s
	s.setQuery("login")
	if len(s.visible()) != 1 || s.tabCount(0) != 1 || s.visible()[0].Sequence != 5 {
		t.Fatalf("query: %v", s.visible())
	}
	if !strings.Contains(view(m), "found 1") {
		t.Error("search row missing")
	}
	s.setQuery("PROJ-3") // a key leaves the filter unchanged
	if s.hits != nil || s.tabCount(0) != 11 {
		t.Fatal("key query changed filter")
	}
	s.setQuery(strings.Repeat("a", 40))
	if len([]rune(s.query)) != 32 {
		t.Fatalf("query len %d", len([]rune(s.query)))
	}
	s.setQuery("")
	s.deps.Config.Filters = []config.Filter{{Name: "hi", Priorities: []string{"high"}}}
	s.filter = 1
	if n := s.tabCount(0); n != 2 || !strings.Contains(view(m), "[filter: hi]") {
		t.Fatalf("filter count %d", n)
	}
	s.filter = 0
	for sb, first := range map[int]int{sortPriority: 3, sortID: 1, sortUpdated: 11, sortState: 1, sortAPI: 1} {
		s.sortBy = sb
		if got := s.visible()[0].Sequence; got != first {
			t.Errorf("sort %d first=%d want %d", sb, got, first)
		}
	}
}

func TestNarrowAndSmall(t *testing.T) {
	m, _, _, _ := newTestModel(t)
	for _, w := range []int{40, 80, 99, 100, 200} {
		feed(t, m, tea.WindowSizeMsg{Width: w, Height: 12})
		for i, l := range strings.Split(view(m), "\n") {
			if lipglossWidth(l) > w {
				t.Fatalf("w=%d line %d wider: %d", w, i, lipglossWidth(l))
			}
		}
	}
	press(t, m, "alt+5")
	feed(t, m, tea.WindowSizeMsg{Width: 30, Height: 8})
	_ = view(m)
}

type fakePane struct{ keys []string }

func (p *fakePane) Update(s *session, msg tea.Msg) (pane, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		p.keys = append(p.keys, keyName(k))
	}
	return p, nil
}
func (p *fakePane) View(s *session, w, h int) string { return "PREVIEW" }

func TestPreviewFocus(t *testing.T) {
	old := newPreview
	defer func() { newPreview = old }()
	fp := &fakePane{}
	newPreview = func(*session) pane { return fp }
	m, _, _, _ := newTestModel(t)
	if !strings.Contains(view(m), "PREVIEW") {
		t.Fatal("no preview at 120")
	}
	press(t, m, "tab", "down")
	if m.s.focus != 1 || len(fp.keys) != 1 {
		t.Fatalf("focus=%d keys=%v", m.s.focus, fp.keys)
	}
	feed(t, m, tea.WindowSizeMsg{Width: 80, Height: 20})
	if strings.Contains(view(m), "PREVIEW") {
		t.Fatal("preview at 80")
	}
}

func TestPoll(t *testing.T) {
	m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, func(c *config.Config) {
		iv := 2 * time.Minute
		c.Refresh.Interval = &iv
	})
	if !m.s.pollArmed {
		t.Fatal("poll chain not armed")
	}
	e.api.Mu.Lock()
	e.api.ItemsV["p1"] = append(e.api.ItemsV["p1"], plane.WorkItem{ID: "i99", ProjectID: "p1", Sequence: 99, Name: "New one", StateID: "s2"})
	e.api.Mu.Unlock()
	feed(t, m, pollMsg{})
	if _, ok := m.s.fresh["i99"]; !ok || e.bells != 1 || len(m.s.items) != 13 || m.s.polling {
		t.Fatalf("poll: fresh=%v bells=%d items=%d", m.s.fresh, e.bells, len(m.s.items))
	}
	if !strings.Contains(m.s.status.text, "+1 new") {
		t.Errorf("status %q", m.s.status.text)
	}
	e.api.Mu.Lock()
	e.api.Err["WorkItems"] = errors.New("429")
	e.api.Mu.Unlock()
	before := m.s.pollEvery
	feed(t, m, pollMsg{})
	if m.s.pollEvery <= before || !m.s.status.isErr {
		t.Fatalf("backoff %v → %v", before, m.s.pollEvery)
	}
}

func TestStartByItemAndBranch(t *testing.T) {
	var pushed plane.WorkItem
	old := newItemScreen
	defer func() { newItemScreen = old }()
	newItemScreen = func(s *session, it plane.WorkItem) screen { pushed = it; return newHelp("item") }
	m, _ := newTestModelCfg(t, StartAt{Item: "PROJ-5"}, nil)
	if m.s.cursor != "i5" || pushed.ID != "i5" || len(m.stack) != 2 {
		t.Fatalf("cursor=%q pushed=%q stack=%d", m.s.cursor, pushed.ID, len(m.stack))
	}
	m, _ = newTestModelCfg(t, StartAt{}, func(c *config.Config) { c.DefaultProject = "OTH" })
	if m.s.project.Identifier != "OTH" || len(m.s.items) != 1 {
		t.Fatalf("default project: %+v", m.s.project)
	}
}

func TestNoProject(t *testing.T) {
	old := newProjects
	newProjects = nil
	defer func() { newProjects = old }()
	m, _ := newTestModelCfg(t, StartAt{}, nil)
	if m.s.project != nil || m.s.status.text != "no project selected" || len(m.s.projects) != 2 {
		t.Fatalf("status %+v", m.s.status)
	}
	newProjects = func(s *session) screen { return newHelp("projects") }
	m, _ = newTestModelCfg(t, StartAt{}, nil)
	if len(m.stack) != 2 {
		t.Fatal("projects screen not pushed")
	}
	press(t, m, "R") // must not panic without a project in the projects scope
}

func TestBootErrors(t *testing.T) {
	m, e := newTestModelCfg(t, StartAt{Project: "NOPE"}, nil)
	if m.s.project != nil || !m.s.status.isErr {
		t.Fatalf("unknown project: %+v", m.s.status)
	}
	_ = e
}

func TestBranchesLoaded(t *testing.T) {
	m, _ := newTestModelCfg(t, StartAt{Project: "PROJ"}, nil)
	if m.s.branches == nil {
		t.Fatal("branches nil")
	}
}

func TestNavList(t *testing.T) {
	var c cursorList
	c.SetLen(20)
	c.SetHeight(5)
	for _, k := range []string{"pgdown", "down", "end", "up", "pgup", "home"} {
		if !c.Key(k) {
			t.Fatal(k)
		}
	}
	if c.Pos() != 0 || c.Key("x") {
		t.Fatal("home/x")
	}
	c.Key("end")
	if f, to := c.Range(); f != 15 || to != 20 {
		t.Fatalf("range %d-%d", f, to)
	}
	c.SetLen(0)
	if c.Pos() != 0 {
		t.Fatal("empty")
	}
}

func TestKeyLabels(t *testing.T) {
	for in, want := range map[string]string{"ctrl+left": "Ctrl+←", "shift+tab": "Shift+Tab", "R": "R", "esc": "Esc", "alt+1": "Alt+1"} {
		if got := keyLabel(in); got != want {
			t.Errorf("%s → %s", in, got)
		}
	}
	if keysLabel([]string{"alt+1", "alt+2"}) != "Alt+1 Alt+2" {
		t.Error("partial digits")
	}
}

func TestDefaultsAndStyles(t *testing.T) {
	d := Deps{}.withDefaults()
	if d.Now == nil || d.OpenURL == nil || d.Clipboard == nil || d.Config == nil {
		t.Fatal("defaults")
	}
	cfg := &config.Config{Theme: config.Theme{NoColor: true}}
	if st := newStyles(cfg); !st.plain || st.priority("high").Render("x") != "x" {
		t.Fatal("plain styles")
	}
	if st := newStyles(&config.Config{}); st.plain || st.prio["high"].Render("x") == "" {
		t.Fatal("color styles")
	}
}
