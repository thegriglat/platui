package app

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/thegriglat/platui/internal/forge"
)

func TestContractAppV3(t *testing.T) {
	mr := forge.MR{IID: 5, Title: "PROJ-1", URL: "https://gl/mr/5", State: "opened"}
	setup := func() *env {
		e := newEnv(t)
		e.forge.MRs = []forge.MR{mr}
		ttl := time.Hour
		e.cfg.GitLab.MRCache = &ttl
		e.git.CurrentV = "PROJ-1-other-slug"
		e.git.BranchesV["PROJ-1-old"] = gitbranchtestStatus("PROJ-1-old")
		e.git.BranchesV["PROJ-2-x"] = gitbranchtestStatus("PROJ-2-x")
		e.rebuild()
		return e
	}
	t.Run("branches and key", func(t *testing.T) {
		e := setup()
		got, err := e.app.MRs(ctx(), e.project(), e.item("i1"), false)
		if err != nil || len(got) != 1 || got[0].IID != 5 || len(e.forge.Finds) != 1 {
			t.Fatalf("%+v %v", got, err)
		}
		f := e.forge.Finds[0]
		if f[0] != "PROJ-1-fix-login" || f[len(f)-1] != "PROJ-1" || !slices.Contains(f, "PROJ-1-other-slug") ||
			!slices.Contains(f, "PROJ-1-old") || slices.Contains(f, "PROJ-2-x") {
			t.Fatalf("%v", f)
		}
	})
	t.Run("cache and fresh", func(t *testing.T) {
		e := setup()
		it := e.item("i1")
		if _, ok := e.app.CachedMRs(it.ID); ok {
			t.Fatal("empty")
		}
		e.app.MRs(ctx(), e.project(), it, false)
		e.app.MRs(ctx(), e.project(), it, false)
		if len(e.forge.Finds) != 1 {
			t.Fatal("cache not used")
		}
		e.app.MRs(ctx(), e.project(), it, true)
		if len(e.forge.Finds) != 2 {
			t.Fatal("fresh ignored")
		}
		e.rebuild() // new process: Store only
		if c, ok := e.app.CachedMRs(it.ID); !ok || len(c) != 1 || len(e.forge.Finds) != 2 {
			t.Fatalf("%v %v", c, ok)
		}
		e.now = e.now.Add(2 * time.Hour)
		e.rebuild()
		e.app.MRs(ctx(), e.project(), it, false)
		if len(e.forge.Finds) != 3 {
			t.Fatal("ttl")
		}
	})
	t.Run("no cache", func(t *testing.T) {
		e := setup()
		zero := time.Duration(0)
		e.cfg.GitLab.MRCache = &zero
		e.app.MRs(ctx(), e.project(), e.item("i1"), false)
		e.app.MRs(ctx(), e.project(), e.item("i1"), false)
		if len(e.forge.Finds) != 2 {
			t.Fatal("cached with ttl 0")
		}
	})
	t.Run("errors and no git", func(t *testing.T) {
		e := setup()
		e.forge.Err = forge.ErrNotFound
		if _, err := e.app.MRs(ctx(), e.project(), e.item("i1"), true); !errors.Is(err, forge.ErrNotFound) {
			t.Fatalf("%v", err)
		}
		e.forge.Err = nil
		a := New(Deps{API: e.api, Forge: e.forge, Config: e.cfg})
		if _, err := a.MRs(ctx(), e.project(), e.item("i1"), true); err != nil || len(e.forge.Finds[len(e.forge.Finds)-1]) != 2 {
			t.Fatalf("%v %v", err, e.forge.Finds)
		}
	})
}
