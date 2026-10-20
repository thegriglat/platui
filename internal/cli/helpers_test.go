package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/cache"
	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/forge/forgetest"
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/gitbranch/gitbranchtest"
	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/plane/planetest"
)

var t0 = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

type fx struct {
	t     *testing.T
	api   *planetest.Fake
	git   *gitbranchtest.Fake
	forge *forgetest.Fake
	store *cache.Mem
	cfgYA string
	out   bytes.Buffer
	err   bytes.Buffer
	tui   *StartAt
	built int
}

func newFx(t *testing.T) *fx {
	t.Helper()
	f := &fx{t: t, api: planetest.NewFake(), git: gitbranchtest.NewFake(), forge: &forgetest.Fake{URL: "https://gh/pr/1"},
		store: cache.NewMem(func() time.Time { return t0 })}
	f.api.MeV = plane.Member{ID: "u-me", DisplayName: "Me", Email: "me@x.io"}
	f.api.ProjectsV = []plane.Project{{ID: "p1", Identifier: "PROJ", Name: "One"}, {ID: "p2", Identifier: "OTH", Name: "Two"}}
	f.api.StatesV["p1"] = []plane.State{
		{ID: "s-todo", Name: "Todo", Group: "unstarted", Sequence: 2},
		{ID: "s-prog", Name: "In Progress", Group: "started", Sequence: 3},
		{ID: "s-rev", Name: "In Review", Group: "started", Sequence: 4},
		{ID: "s-done", Name: "Done", Group: "completed", Sequence: 5},
		{ID: "s-canc", Name: "Cancelled", Group: "cancelled", Sequence: 6},
	}
	f.api.StatesV["p2"] = []plane.State{{ID: "s2", Name: "Todo", Group: "unstarted"}}
	f.api.MembersV["p1"] = []plane.Member{{ID: "u-me", DisplayName: "Me"}, {ID: "u-anna", DisplayName: "Anna K"}}
	f.api.LabelsV["p1"] = []plane.Label{{ID: "l-bug", Name: "Bug"}}
	f.api.ItemsV["p1"] = []plane.WorkItem{
		{ID: "i42", ProjectID: "p1", Sequence: 42, Name: "Add search", DescHTML: "<p>full <b>text</b></p>", StateID: "s-done",
			AssigneeIDs: []string{"u-anna"}, LabelIDs: []string{"l-bug"}, UpdatedAt: t0},
		{ID: "i1", ProjectID: "p1", Sequence: 1, Name: "Fix login", StateID: "s-todo", Priority: "high", AssigneeIDs: []string{"u-me"}, UpdatedAt: t0},
		{ID: "i3", ProjectID: "p1", Sequence: 3, Name: "Old idea", StateID: "s-canc"},
		{ID: "i2", ProjectID: "p1", Sequence: 2, Name: "Fix logout", StateID: "s-todo"},
	}
	f.api.ItemsV["p2"] = []plane.WorkItem{{ID: "o7", ProjectID: "p2", Sequence: 7, Name: "Other", StateID: "s2", UpdatedAt: t0}}
	f.git.TopV, f.git.RemoteURLV, f.git.CurrentV = "/src/repo", "git@github.com:a/b.git", "main"
	f.cfgYA = "api_key: k\nworkspace: ws\ndefault_project: PROJ\n"
	return f
}

// run executes a command; the config is written to a temp file.
func (f *fx) run(args ...string) int {
	f.t.Helper()
	f.out.Reset()
	f.err.Reset()
	path := filepath.Join(f.t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(f.cfgYA), 0o644); err != nil {
		f.t.Fatal(err)
	}
	return Run(context.Background(), append([]string{"--config", path}, args...), f.env())
}

func (f *fx) env() Env {
	return Env{
		Stdout: &f.out, Stderr: &f.err,
		Getenv: func(string) string { return "" },
		Getwd:  func() (string, error) { return "/src/repo", nil },
		Build: func(cfg *config.Config, _ string) (*app.App, error) {
			f.built++
			return app.New(app.Deps{API: f.api, Git: f.git, Store: f.store, Config: cfg,
				Forge: f.forge, Now: func() time.Time { return t0 }}), nil
		},
		RunTUI: func(_ *app.App, _ *config.Config, s StartAt) error { f.tui = &s; return nil },
	}
}

func (f *fx) useGit() {
	old := openGit
	openGit = func(string) (gitbranch.Git, error) { return f.git, nil }
	f.t.Cleanup(func() { openGit = old })
}
