package app

import (
	"context"
	"testing"
	"time"

	"github.com/thegriglat/platui/internal/cache"
	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/forge/forgetest"
	"github.com/thegriglat/platui/internal/gitbranch/gitbranchtest"
	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/plane/planetest"
)

var t0 = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

type env struct {
	api   *planetest.Fake
	git   *gitbranchtest.Fake
	forge *forgetest.Fake
	store *cache.Mem
	cfg   *config.Config
	now   time.Time
	app   *App
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{api: planetest.NewFake(), git: gitbranchtest.NewFake(), forge: &forgetest.Fake{URL: "https://gh/pr/1"},
		store: cache.NewMem(func() time.Time { return t0 }), cfg: &config.Config{Workspace: "ws"}, now: t0}
	e.api.MeV = plane.Member{ID: "u-me", DisplayName: "Me", Email: "me@x.io"}
	e.api.ProjectsV = []plane.Project{{ID: "p1", Identifier: "PROJ", Name: "One"}, {ID: "p2", Identifier: "OTH", Name: "Two"}}
	e.api.StatesV["p1"] = []plane.State{
		{ID: "s-back", Name: "Backlog", Group: "backlog", Sequence: 1},
		{ID: "s-todo", Name: "Todo", Group: "unstarted", Sequence: 2},
		{ID: "s-prog", Name: "In Progress", Group: "started", Sequence: 3},
		{ID: "s-rev", Name: "In Review", Group: "started", Sequence: 4},
		{ID: "s-done", Name: "Done", Group: "completed", Sequence: 5},
		{ID: "s-canc", Name: "Cancelled", Group: "cancelled", Sequence: 6},
	}
	e.api.MembersV["p1"] = []plane.Member{
		{ID: "u-me", DisplayName: "Me", Email: "me@x.io"},
		{ID: "u-anna", DisplayName: "Anna K", Email: "anna@x.io"},
		{ID: "u-bob1", DisplayName: "Bob", Email: "bob1@x.io"},
		{ID: "u-bob2", DisplayName: "bob", Email: "bob2@x.io"},
	}
	e.api.LabelsV["p1"] = []plane.Label{{ID: "l-bug", Name: "Bug"}, {ID: "l-ui", Name: "UI"}}
	e.api.ItemsV["p1"] = []plane.WorkItem{
		{ID: "i1", ProjectID: "p1", Sequence: 1, Name: "Fix login", StateID: "s-todo", Priority: "high", UpdatedAt: t0},
		{ID: "i42", ProjectID: "p1", Sequence: 42, Name: "Add search", DescHTML: "<p>full text</p>", StateID: "s-done",
			AssigneeIDs: []string{"u-anna"}, LabelIDs: []string{"l-bug"}, UpdatedAt: t0},
	}
	e.api.ItemsV["p2"] = []plane.WorkItem{{ID: "o7", ProjectID: "p2", Sequence: 7, Name: "Other", UpdatedAt: t0}}
	e.git.TopV, e.git.RemoteURLV, e.git.CurrentV = "/src/repo", "git@github.com:a/b.git", "main"
	e.git.BranchesV["main"] = gitbranchtestStatus("main")
	e.rebuild()
	return e
}

func (e *env) rebuild() {
	e.app = New(Deps{API: e.api, Git: e.git, Forge: e.forge,
		Store: e.store, Config: e.cfg, Now: func() time.Time { return e.now }})
}

func (e *env) calls(prefix string) int {
	e.api.Mu.Lock()
	defer e.api.Mu.Unlock()
	n := 0
	for _, c := range e.api.Calls {
		if len(c) >= len(prefix) && c[:len(prefix)] == prefix {
			n++
		}
	}
	return n
}

func ctx() context.Context { return context.Background() }

func str(s string) *string { return &s }

func (e *env) project() plane.Project { return e.api.ProjectsV[0] }
func (e *env) item(id string) plane.WorkItem {
	for _, it := range e.api.ItemsV["p1"] {
		if it.ID == id {
			return it
		}
	}
	panic(id)
}

func contextCancelled() (context.Context, context.CancelFunc) {
	c, cancel := context.WithCancel(context.Background())
	cancel()
	return c, cancel
}
