package app

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/runner"
)

func TestContractAppV6DiffStat(t *testing.T) {
	e := newEnv(t)
	e.cfg.Branch.Base = "main"
	e.git.BranchesV["PROJ-1-fix-login"] = gitbranch.BranchStatus{Name: "PROJ-1-fix-login", Local: true}
	e.git.Stats = map[string]gitbranch.Stat{"PROJ-1-fix-login": {Files: 2, Insertions: 10, Deletions: 3}}
	e.rebuild()
	s, ok, err := e.app.DiffStat(e.project(), e.item("i1"))
	if err != nil || !ok || s != (gitbranch.Stat{Files: 2, Insertions: 10, Deletions: 3}) {
		t.Fatalf("%v %v %v", s, ok, err)
	}
	if _, ok, err := e.app.DiffStat(e.project(), e.item("i42")); ok || err != nil {
		t.Fatalf("no branch: %v %v", ok, err)
	}
	app := New(Deps{API: e.api, Config: e.cfg})
	if _, ok, err := app.DiffStat(e.project(), e.item("i1")); ok || err != nil {
		t.Fatalf("nil git: %v %v", ok, err)
	}
}

func TestContractAppV6RenderCommand(t *testing.T) {
	v := CommandVars{ID: "i1", Key: "PROJ-1", Branch: "b", URL: "http://u", Title: "it's $(rm -rf x) `y`", MRURL: "http://mr"}
	got := RenderCommand("{id} {key} {branch} {url} {title} {mr_url}", v)
	want := `'i1' 'PROJ-1' 'b' 'http://u' 'it'\''s $(rm -rf x) ` + "`y`" + `' 'http://mr'`
	if got != want {
		t.Fatalf("%s\nwant %s", got, want)
	}
	// реально через sh: ничего не исполняется
	r, err := runner.Default()(context.Background(), "", "printf %s "+RenderCommand("{title}", v), nil)
	if err != nil || r.Stdout != v.Title {
		t.Fatalf("%q %v", r.Stdout, err)
	}
}

func TestContractAppV6RunCommand(t *testing.T) {
	e := newEnv(t)
	var gotDir, gotCmd string
	sh := func(_ context.Context, dir, cmd string, _ []string) (runner.Result, error) {
		gotDir, gotCmd = dir, cmd
		return runner.Result{Stdout: "ok"}, nil
	}
	app := New(Deps{API: e.api, Git: e.git, Forge: e.forge, Config: e.cfg, Shell: sh})
	r, err := app.RunCommand(ctx(), config.Command{Command: "echo {key}"}, CommandVars{Key: "PROJ-1"})
	if err != nil || r.Stdout != "ok" || gotDir != "/src/repo" || gotCmd != "echo 'PROJ-1'" {
		t.Fatalf("%v %v %q %q", r, err, gotDir, gotCmd)
	}
	app = New(Deps{API: e.api, Config: e.cfg, Shell: sh})
	app.RunCommand(ctx(), config.Command{Command: "x"}, CommandVars{})
	if gotDir != "." {
		t.Fatalf("dir %q", gotDir)
	}
	// Shell по умолчанию
	app = New(Deps{API: e.api, Config: e.cfg})
	r, err = app.RunCommand(ctx(), config.Command{Command: "echo hi"}, CommandVars{})
	if err != nil || r.Stdout != "hi\n" {
		t.Fatalf("%v %v", r, err)
	}
}

func TestContractAppV6CommandVars(t *testing.T) {
	e := newEnv(t)
	e.cfg.GitLab.MRCache = nil
	v := e.app.CommandVars(e.project(), e.item("i1"))
	if v.ID != "i1" || v.Key != "PROJ-1" || v.Branch != "PROJ-1-fix-login" || v.Title != "Fix login" || v.MRURL != "" || v.URL == "" {
		t.Fatalf("%+v", v)
	}
	if _, err := e.app.MRs(ctx(), e.project(), e.item("i1"), true); err != nil {
		t.Fatal(err)
	}
	if v := e.app.CommandVars(e.project(), e.item("i1")); v.MRURL != "" && !strings.HasPrefix(v.MRURL, "http") {
		t.Fatalf("%+v", v)
	}
}

func TestContractAppV6Bookmarks(t *testing.T) {
	e := newEnv(t)
	if len(e.app.Bookmarks("p1")) != 0 {
		t.Fatal("not empty")
	}
	for _, id := range []string{"i42", "i1"} {
		if on, err := e.app.ToggleBookmark("p1", id); !on || err != nil {
			t.Fatal(on, err)
		}
	}
	if got := e.app.Bookmarks("p1"); !slices.Equal(got, []string{"i42", "i1"}) {
		t.Fatal(got)
	}
	e.rebuild() // новый App, тот же Store
	if got := e.app.Bookmarks("p1"); !slices.Equal(got, []string{"i42", "i1"}) {
		t.Fatalf("persist: %v", got)
	}
	if on, _ := e.app.ToggleBookmark("p1", "i42"); on {
		t.Fatal("should be off")
	}
	if got := e.app.Bookmarks("p1"); !slices.Equal(got, []string{"i1"}) {
		t.Fatal(got)
	}
	if len(e.app.Bookmarks("p2")) != 0 {
		t.Fatal("projects mixed")
	}
	// без Store — память
	app := New(Deps{API: e.api, Config: e.cfg})
	app.ToggleBookmark("p1", "x")
	if !slices.Equal(app.Bookmarks("p1"), []string{"x"}) {
		t.Fatal("mem")
	}
}

type failingStore struct{}

func (failingStore) Load(string, any) (time.Time, bool, error) { return time.Time{}, false, nil }
func (failingStore) Save(string, any) error                    { return errors.New("disk full") }
func TestContractAppV6BookmarksSaveError(t *testing.T) {
	e := newEnv(t)
	app := New(Deps{API: e.api, Config: e.cfg, Store: failingStore{}})
	on, err := app.ToggleBookmark("p1", "x")
	if !on || err == nil || !slices.Equal(app.Bookmarks("p1"), []string{"x"}) {
		t.Fatal(on, err)
	}
}

func ids(ns []*Node) []string {
	var out []string
	for _, n := range ns {
		out = append(out, strings.Repeat("-", n.Depth)+n.Item.ID)
		out = append(out, ids(n.Children)...)
	}
	return out
}

func TestContractAppV6Tree(t *testing.T) {
	w := func(id, parent string) plane.WorkItem { return plane.WorkItem{ID: id, ParentID: parent} }
	t.Run("three levels", func(t *testing.T) {
		got := ids(Tree([]plane.WorkItem{w("a", ""), w("c", "b"), w("b", "a"), w("d", "a"), w("e", "")}))
		want := []string{"a", "-b", "--c", "-d", "e"}
		if !slices.Equal(got, want) {
			t.Fatal(got)
		}
	})
	t.Run("parent outside", func(t *testing.T) {
		got := ids(Tree([]plane.WorkItem{w("x", "gone"), w("y", "x")}))
		if !slices.Equal(got, []string{"x", "-y"}) {
			t.Fatal(got)
		}
	})
	t.Run("cycle", func(t *testing.T) {
		got := ids(Tree([]plane.WorkItem{w("a", "b"), w("b", "a"), w("c", "a"), w("s", "s")}))
		if !slices.Equal(got, []string{"a", "-c", "b", "s"}) {
			t.Fatal(got)
		}
	})
	if Tree(nil) != nil {
		t.Fatal("nil")
	}
}
