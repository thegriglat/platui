package tui

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/cache"
	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/forge/forgetest"
	"github.com/thegriglat/platui/internal/gitbranch/gitbranchtest"
	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/plane/planetest"
)

var t0 = time.Date(2026, 10, 20, 18, 40, 0, 0, time.UTC)

// run выполняет команду, разворачивая Batch; Tick и зависшие команды (>100 мс) пропускаются.
func run(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		if b, ok := msg.(tea.BatchMsg); ok {
			var out []tea.Msg
			for _, c := range b {
				out = append(out, run(t, c)...)
			}
			return out
		}
		if msg == nil {
			return nil
		}
		return []tea.Msg{msg}
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

// feed подаёт сообщение и рекурсивно обрабатывает порождённые команды.
func feed(t *testing.T, m *model, msg tea.Msg) {
	t.Helper()
	_, cmd := m.Update(msg)
	for _, r := range run(t, cmd) {
		feed(t, m, r)
	}
}

func press(t *testing.T, m *model, keys ...string) {
	t.Helper()
	for _, k := range keys {
		feed(t, m, keyMsg(k))
	}
}

func keyMsg(k string) tea.KeyMsg {
	named := map[string]tea.KeyType{"up": tea.KeyUp, "down": tea.KeyDown, "esc": tea.KeyEsc, "tab": tea.KeyTab,
		"pgdown": tea.KeyPgDown, "pgup": tea.KeyPgUp, "home": tea.KeyHome, "end": tea.KeyEnd, "ctrl+c": tea.KeyCtrlC,
		"ctrl+left": tea.KeyCtrlLeft, "ctrl+right": tea.KeyCtrlRight, "enter": tea.KeyEnter, "shift+tab": tea.KeyShiftTab}
	if t, ok := named[k]; ok {
		return tea.KeyMsg{Type: t}
	}
	if strings.HasPrefix(k, "alt+") {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k[4:]), Alt: true}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

func view(m *model) string { return m.View() }

type testEnv struct {
	api   *planetest.Fake
	git   *gitbranchtest.Fake
	forge *forgetest.Fake
	cfg   *config.Config
	mu    sync.Mutex
	bells int
}

// newTestModel — модель с образцовым проектом PROJ (5 состояний, 2 участника, 2 метки, 12 задач).
func newTestModel(t *testing.T) (*model, *planetest.Fake, *gitbranchtest.Fake, *forgetest.Fake) {
	m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, nil)
	return m, e.api, e.git, e.forge
}

func newTestModelCfg(t *testing.T, start StartAt, edit func(*config.Config)) (*model, *testEnv) {
	t.Helper()
	zero := time.Duration(0)
	cfg := &config.Config{Workspace: "ws", Refresh: config.Refresh{Interval: &zero}, Tabs: config.Tabs{HideGroups: []string{"cancelled"}}}
	if edit != nil {
		edit(cfg)
	}
	api, git, fg := planetest.NewFake(), gitbranchtest.NewFake(), &forgetest.Fake{URL: "https://x/pr/1"}
	api.MeV = plane.Member{ID: "u-me", DisplayName: "Me"}
	api.ProjectsV = []plane.Project{{ID: "p1", Identifier: "PROJ", Name: "One"}, {ID: "p2", Identifier: "OTH", Name: "Two"}}
	api.StatesV["p1"] = []plane.State{
		{ID: "s1", Name: "Backlog", Group: "backlog", Sequence: 1}, {ID: "s2", Name: "To Do", Group: "unstarted", Sequence: 2},
		{ID: "s3", Name: "In Progress", Group: "started", Sequence: 3}, {ID: "s4", Name: "Done", Group: "completed", Sequence: 4},
		{ID: "s5", Name: "Cancelled", Group: "cancelled", Sequence: 5}}
	api.MembersV["p1"] = []plane.Member{{ID: "u-me", DisplayName: "Me"}, {ID: "u-an", DisplayName: "Anna"}}
	api.LabelsV["p1"] = []plane.Label{{ID: "l1", Name: "Bug"}, {ID: "l2", Name: "UI"}}
	states := []string{"s1", "s1", "s2", "s2", "s2", "s3", "s3", "s3", "s4", "s4", "s4", "s5"}
	prios := []string{"low", "high", "urgent", "none", "medium"}
	for i, st := range states {
		it := plane.WorkItem{ID: fmt.Sprintf("i%d", i+1), ProjectID: "p1", Sequence: i + 1, Name: fmt.Sprintf("Task number %d", i+1),
			StateID: st, Priority: prios[i%5], UpdatedAt: t0.Add(time.Duration(i) * time.Minute)}
		if i%2 == 0 {
			it.AssigneeIDs = []string{"u-me"}
		}
		if i < 3 {
			it.DescHTML = "<p>desc</p>"
		}
		api.ItemsV["p1"] = append(api.ItemsV["p1"], it)
	}
	api.ItemsV["p1"][4].Name = "Fix login page"
	api.ItemsV["p2"] = []plane.WorkItem{{ID: "o7", ProjectID: "p2", Sequence: 7, Name: "Other thing", UpdatedAt: t0}}
	git.TopV, git.CurrentV = "/src/repo", "main"
	a := app.New(app.Deps{API: api, Git: git, Forge: &forgetest.Fake{},
		Store: cache.NewMem(func() time.Time { return t0 }), Config: cfg, Now: func() time.Time { return t0 }})
	e := &testEnv{api: api, git: git, forge: fg, cfg: cfg}
	m := newModel(Deps{App: a, Config: cfg, Now: func() time.Time { return t0 }, Start: start})
	m.s.bell = func() { e.mu.Lock(); e.bells++; e.mu.Unlock() }
	feed(t, m, tea.WindowSizeMsg{Width: 120, Height: 30})
	for _, r := range run(t, m.Init()) {
		feed(t, m, r)
	}
	return m, e
}

func (e *testEnv) count(prefix string) int {
	e.api.Mu.Lock()
	defer e.api.Mu.Unlock()
	n := 0
	for _, c := range e.api.Calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}
