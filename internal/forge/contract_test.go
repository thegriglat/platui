package forge_test

import (
	"context"
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/thegriglat/platui/internal/forge"
	"github.com/thegriglat/platui/internal/forge/forgetest"
)

type call struct {
	dir  string
	env  []string
	name string
	args []string
}

type step struct {
	out string
	err error
}

func seq(calls *[]call, steps ...step) forge.Runner {
	return func(_ context.Context, dir string, env []string, name string, args ...string) (string, error) {
		*calls = append(*calls, call{dir, env, name, args})
		if len(*calls) > len(steps) {
			return "", nil
		}
		s := steps[len(*calls)-1]
		return s.out, s.err
	}
}

func TestContractForgeCreateArgs(t *testing.T) {
	var calls []call
	tl := forge.New(forge.Options{Glab: "/bin/glab", Host: "gl.x", Repo: "a/b", Run: seq(&calls,
		step{out: "!1 T\nhttps://gitlab.com/a/b/-/merge_requests/1\n"}, step{out: "https://g/mr/2"})})
	u, err := tl.CreateMR(context.Background(), forge.MRRequest{Dir: "/r", Head: "h", Target: "dev", Title: "T", Body: "B", Draft: true})
	if err != nil || u != "https://gitlab.com/a/b/-/merge_requests/1" {
		t.Fatal(u, err)
	}
	want := []string{"mr", "create", "--source-branch", "h", "--title", "T", "--description", "B", "--target-branch", "dev", "--draft", "--yes", "-R", "a/b"}
	if c := calls[0]; c.dir != "/r" || c.name != "/bin/glab" || !reflect.DeepEqual(c.args, want) || !reflect.DeepEqual(c.env, []string{"GITLAB_HOST=gl.x"}) {
		t.Errorf("%+v", c)
	}
	tl = forge.New(forge.Options{Run: seq(&calls, step{}, step{}, step{out: "https://g/mr/2"})})
	calls = nil
	if _, err = tl.CreateMR(context.Background(), forge.MRRequest{Head: "h", Title: "T", Body: "B"}); err == nil {
		t.Error("no url must fail")
	}
	calls = nil
	tl = forge.New(forge.Options{Run: seq(&calls, step{out: "https://g/mr/2"})})
	if _, err = tl.CreateMR(context.Background(), forge.MRRequest{Head: "h", Title: "T", Body: "B"}); err != nil {
		t.Fatal(err)
	}
	want = []string{"mr", "create", "--source-branch", "h", "--title", "T", "--description", "B", "--yes"}
	if calls[0].name != "glab" || calls[0].env != nil || !reflect.DeepEqual(calls[0].args, want) {
		t.Errorf("%+v", calls[0])
	}
}

func TestContractForgeAlreadyExists(t *testing.T) {
	var calls []call
	tl := forge.New(forge.Options{Repo: "a/b", Run: seq(&calls,
		step{err: errors.New("already exists")}, step{out: `[{"iid":3,"web_url":"https://gl/mr/3","state":"opened"}]`})})
	u, err := tl.CreateMR(context.Background(), forge.MRRequest{Head: "h"})
	if err != nil || u != "https://gl/mr/3" {
		t.Fatal(u, err)
	}
	want := []string{"mr", "list", "-s", "h", "-A", "-F", "json", "-R", "a/b"}
	if !reflect.DeepEqual(calls[1].args, want) {
		t.Errorf("%v", calls[1].args)
	}
	// «already exists» в stdout
	calls = nil
	tl = forge.New(forge.Options{Run: seq(&calls, step{out: "already exists", err: errors.New("x")}, step{out: `[{"web_url":"u"}]`})})
	if u, err = tl.CreateMR(context.Background(), forge.MRRequest{Head: "h"}); err != nil || u != "u" {
		t.Fatal(u, err)
	}
}

func TestContractForgeErrors(t *testing.T) {
	ctx := context.Background()
	for name, steps := range map[string][]step{
		"notfound":  {{err: exec.ErrNotFound}},
		"notfound2": {{err: forge.ErrNotFound}},
	} {
		var calls []call
		tl := forge.New(forge.Options{Run: seq(&calls, steps...)})
		if _, err := tl.CreateMR(ctx, forge.MRRequest{}); !errors.Is(err, forge.ErrNotFound) {
			t.Errorf("%s: %v", name, err)
		}
		calls = nil
		if _, err := tl.FindMRs(ctx, "", []string{"b"}, ""); !errors.Is(err, forge.ErrNotFound) {
			t.Errorf("%s find: %v", name, err)
		}
	}
	var calls []call
	tl := forge.New(forge.Options{Run: seq(&calls, step{out: "out", err: errors.New("boom")})})
	if _, err := tl.CreateMR(ctx, forge.MRRequest{}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("err: %v", err)
	}
	calls = nil
	tl = forge.New(forge.Options{Run: seq(&calls, step{err: errors.New("already exists")}, step{err: errors.New("listfail")})})
	if _, err := tl.CreateMR(ctx, forge.MRRequest{}); err == nil || !strings.Contains(err.Error(), "listfail") {
		t.Errorf("list: %v", err)
	}
	calls = nil
	tl = forge.New(forge.Options{Run: seq(&calls, step{err: errors.New("already exists")}, step{out: "[]"})})
	if _, err := tl.CreateMR(ctx, forge.MRRequest{}); err == nil {
		t.Error("empty list")
	}
	calls = nil
	tl = forge.New(forge.Options{Run: seq(&calls, step{out: "garbage"})})
	if _, err := tl.FindMRs(ctx, "", []string{"b"}, ""); err == nil {
		t.Error("garbage")
	}
	calls = nil
	tl = forge.New(forge.Options{Run: seq(&calls, step{out: "[]"}, step{err: errors.New("e2")})})
	if _, err := tl.FindMRs(ctx, "", []string{"b"}, "K"); err == nil {
		t.Error("search err")
	}
	calls = nil
	tl = forge.New(forge.Options{Run: seq(&calls, step{out: "[]"}, step{out: "[]"}, step{err: errors.New("e3")})})
	if _, err := tl.FindMRs(ctx, "", []string{"b"}, "K"); err == nil {
		t.Error("recent err")
	}
	calls = nil
	tl = forge.New(forge.Options{Run: seq(&calls, step{err: errors.New("e1")})})
	if _, err := tl.FindMRs(ctx, "", []string{"b"}, "K"); err == nil {
		t.Error("branch err")
	}
}

func TestContractForgeFindMRs(t *testing.T) {
	var calls []call
	tl := forge.New(forge.Options{Host: "h", Repo: "a/b", Run: seq(&calls,
		step{out: `[{"iid":1,"title":"a","state":"merged","source_branch":"PROJ-1-x","updated_at":"2026-01-02T00:00:00Z"},
		            {"iid":2,"state":"closed","source_branch":"PROJ-1-y"}]`},
		step{out: `[{"iid":1,"state":"merged"},{"iid":3,"state":"opened","updated_at":"2026-01-01T00:00:00Z","web_url":"u3","target_branch":"main"}]`},
		step{out: `[{"iid":4,"state":"merged","source_branch":"feature/proj-1-z","updated_at":"2026-03-01T00:00:00Z"},
		            {"iid":5,"state":"opened","source_branch":"other","updated_at":"2026-04-01T00:00:00Z"},
		            {"iid":6,"state":"opened","source_branch":"proj-1-new","updated_at":"2026-05-01T00:00:00Z"}]`},
	)})
	mrs, err := tl.FindMRs(context.Background(), "/r", []string{"PROJ-1-x"}, "PROJ-1")
	if err != nil {
		t.Fatal(err)
	}
	var iids []int
	for _, m := range mrs {
		iids = append(iids, m.IID)
	}
	if !reflect.DeepEqual(iids, []int{6, 3, 4, 1}) {
		t.Fatalf("%v", iids)
	}
	if mrs[1].URL != "u3" || mrs[1].TargetBranch != "main" {
		t.Errorf("%+v", mrs[1])
	}
	wants := [][]string{
		{"mr", "list", "-s", "PROJ-1-x", "-A", "-F", "json", "-R", "a/b"},
		{"mr", "list", "-A", "--search", "PROJ-1", "-F", "json", "-R", "a/b"},
		{"mr", "list", "-A", "-o", "updated_at", "-P", "100", "-F", "json", "-R", "a/b"},
	}
	if len(calls) != 3 {
		t.Fatalf("%d calls", len(calls))
	}
	for i, w := range wants {
		if !reflect.DeepEqual(calls[i].args, w) || calls[i].dir != "/r" || !reflect.DeepEqual(calls[i].env, []string{"GITLAB_HOST=h"}) {
			t.Errorf("%d: %+v", i, calls[i])
		}
	}
	// пустой вывод, без key
	calls = nil
	tl = forge.New(forge.Options{Run: seq(&calls, step{out: ""}, step{out: "[]"})})
	mrs, err = tl.FindMRs(context.Background(), "", []string{"a", "b"}, "")
	if err != nil || len(mrs) != 0 || len(calls) != 2 {
		t.Errorf("%v %v %d", mrs, err, len(calls))
	}
}

func TestContractForgeExecRunner(t *testing.T) {
	run := forge.ExecRunner()
	ctx := context.Background()
	out, err := run(ctx, t.TempDir(), nil, "sh", "-c", "echo hi; echo bad >&2; exit 3")
	if err == nil || !strings.Contains(err.Error(), "bad") || strings.TrimSpace(out) != "hi" {
		t.Errorf("%q %v", out, err)
	}
	if out, err = run(ctx, "", []string{"FOO=bar"}, "sh", "-c", "echo $FOO $LC_ALL"); err != nil || strings.TrimSpace(out) != "bar C" {
		t.Errorf("%q %v", out, err)
	}
	if _, err = run(ctx, "", nil, "no-such-binary-xyz"); !errors.Is(err, forge.ErrNotFound) {
		t.Errorf("%v", err)
	}
}

func TestContractForgeFake(t *testing.T) {
	f := &forgetest.Fake{URL: "u", MRs: []forge.MR{{IID: 1}}}
	var tl forge.Tool = f
	u, err := tl.CreateMR(context.Background(), forge.MRRequest{Head: "h"})
	if u != "u" || err != nil || len(f.Reqs) != 1 {
		t.Error(u, err, f.Reqs)
	}
	mrs, _ := tl.FindMRs(context.Background(), "", []string{"a"}, "K")
	if len(mrs) != 1 || !reflect.DeepEqual(f.Finds, [][]string{{"a", "K"}}) {
		t.Error(mrs, f.Finds)
	}
}
