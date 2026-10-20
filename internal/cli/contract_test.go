package cli

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/plane"
)

func decode(t *testing.T, s string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(s), v); err != nil {
		t.Fatalf("json: %v\n%s", err, s)
	}
}

func TestContractCLILs(t *testing.T) {
	f := newFx(t)
	if c := f.run("ls"); c != 0 {
		t.Fatalf("code %d: %s", c, f.err.String())
	}
	want := "PROJ-1  Todo  high  Fix login\nPROJ-2  Todo  none  Fix logout\nPROJ-42  Done  none  Add search\n"
	got := strings.Join(strings.Fields(f.out.String()), " ")
	if got != strings.Join(strings.Fields(want), " ") {
		t.Fatalf("got:\n%s", f.out.String())
	}
	if !strings.Contains(f.out.String(), "PROJ-1   Todo  high  Fix login\n") {
		t.Fatalf("columns not aligned:\n%s", f.out.String())
	}
	f.run("ls", "--all")
	if !strings.Contains(f.out.String(), "Old idea") {
		t.Fatalf("--all: %s", f.out.String())
	}
	f.run("ls", "--mine")
	if strings.Count(f.out.String(), "\n") != 1 || !strings.Contains(f.out.String(), "PROJ-1") {
		t.Fatalf("--mine: %s", f.out.String())
	}
	f.run("ls", "--state", "done")
	if strings.Count(f.out.String(), "\n") != 1 || !strings.Contains(f.out.String(), "PROJ-42") {
		t.Fatalf("--state: %s", f.out.String())
	}
	f.run("ls", "--json", "--project", "oth")
	var rows []map[string]any
	decode(t, f.out.String(), &rows)
	if len(rows) != 1 || rows[0]["key"] != "OTH-7" {
		t.Fatalf("%v", rows)
	}
	f.run("ls", "--json")
	decode(t, f.out.String(), &rows)
	last := rows[2]
	for _, k := range []string{"key", "id", "name", "state", "state_group", "priority", "assignees", "labels", "url", "updated_at"} {
		if _, ok := last[k]; !ok {
			t.Errorf("missing %s", k)
		}
	}
	if last["key"] != "PROJ-42" || last["state"] != "Done" || last["state_group"] != "completed" ||
		last["assignees"].([]any)[0] != "Anna K" || last["labels"].([]any)[0] != "Bug" ||
		last["url"] != "https://app.plane.so/ws/browse/PROJ-42/" || last["updated_at"] != "2026-10-20T12:00:00Z" {
		t.Fatalf("%v", last)
	}
}

func TestContractCLIShow(t *testing.T) {
	f := newFx(t)
	if c := f.run("show", "PROJ-42"); c != 0 {
		t.Fatalf("%d %s", c, f.err.String())
	}
	for _, s := range []string{"PROJ-42", "Add search", "Done", "Anna K", "Bug", "full text"} {
		if !strings.Contains(f.out.String(), s) {
			t.Errorf("no %q in\n%s", s, f.out.String())
		}
	}
	if c := f.run("show", "42", "--json"); c != 0 {
		t.Fatal(c)
	}
	var o map[string]any
	decode(t, f.out.String(), &o)
	if o["key"] != "PROJ-42" || o["description"] != "full text" {
		t.Fatalf("%v", o)
	}
	if c := f.run("show", "oth-7"); c != 0 || !strings.Contains(f.out.String(), "OTH-7") {
		t.Fatalf("%d %s", c, f.out.String())
	}
	if c := f.run("show", "PROJ-99"); c != 1 || !strings.Contains(f.err.String(), "platui: ") {
		t.Fatalf("%d %s", c, f.err.String())
	}
	if c := f.run("show", "bad ref"); c != 2 {
		t.Fatal(c)
	}
	f.api.CommentsV["i42"] = []plane.Comment{{ID: "c1", Author: "anna", HTML: "<p>looks <b>good</b></p>", CreatedAt: t0}}
	if c := f.run("show", "PROJ-42", "--comments"); c != 0 || !strings.Contains(f.out.String(), "looks good") {
		t.Fatalf("%d %s %s", c, f.out.String(), f.err.String())
	}
	f.run("show", "PROJ-42", "--comments", "--json")
	decode(t, f.out.String(), &o)
	if cs, _ := o["comments"].([]any); len(cs) != 1 {
		t.Fatalf("%v", o)
	}
	f.cfgYA += "tabs:\n  states: [Done, Todo]\n"
	f.run("ls")
	if got := strings.Fields(f.out.String()); got[0] != "PROJ-42" || strings.Count(f.out.String(), "\n") != 3 {
		t.Fatalf("tabs.states order: %s", f.out.String())
	}
}

func TestContractCLISearch(t *testing.T) {
	f := newFx(t)
	if c := f.run("search", "fix", "log"); c != 0 {
		t.Fatalf("%d %s", c, f.err.String())
	}
	if strings.Count(f.out.String(), "\n") != 2 || !strings.Contains(f.out.String(), "Fix login") {
		t.Fatalf("%s", f.out.String())
	}
	f.run("search", "--json", "zzzzqq")
	if strings.TrimSpace(f.out.String()) != "[]" {
		t.Fatalf("%s", f.out.String())
	}
	if c := f.run("search", "PROJ-42"); c != 0 || !strings.Contains(f.out.String(), "Add search") || !strings.Contains(f.out.String(), "State:") {
		t.Fatalf("key: %s", f.out.String())
	}
	if c := f.run("search", "--server", "--json", "login"); c != 0 {
		t.Fatalf("%d %s", c, f.err.String())
	}
	if c := f.run("search", "--server", "login"); c != 0 {
		t.Fatalf("%d %s", c, f.err.String())
	}
	if c := f.run("search"); c != 2 {
		t.Fatal(c)
	}
}

func TestContractCLIBranchStartReview(t *testing.T) {
	f := newFx(t)
	f.useGit()
	if c := f.run("branch", "PROJ-1", "--print"); c != 0 || strings.TrimSpace(f.out.String()) != "PROJ-1-fix-login" {
		t.Fatalf("%d %q %s", c, f.out.String(), f.err.String())
	}
	if c := f.run("branch", "PROJ-1"); c != 0 || !strings.Contains(f.out.String(), "PROJ-1-fix-login (created)") {
		t.Fatalf("%d %s %s", c, f.out.String(), f.err.String())
	}
	if c := f.run("branch", "PROJ-1"); c != 0 || !strings.Contains(f.out.String(), "(switched)") {
		t.Fatalf("existing: %d %s %s", c, f.out.String(), f.err.String())
	}
	if c := f.run("branch", "PROJ-2", "--worktree"); c != 0 || !strings.Contains(f.out.String(), "worktree") {
		t.Fatalf("wt: %d %s %s", c, f.out.String(), f.err.String())
	}
	if c := f.run("branch"); c != 2 {
		t.Fatal(c)
	}
	f = newFx(t)
	if c := f.run("start", "PROJ-1"); c != 0 {
		t.Fatalf("%d %s", c, f.err.String())
	}
	for _, s := range []string{"PROJ-1-fix-login", "state: In Progress"} {
		if !strings.Contains(f.out.String(), s) {
			t.Errorf("no %q in %s", s, f.out.String())
		}
	}
	if c := f.run("start", "PROJ-2", "--worktree"); c != 0 || !strings.Contains(f.out.String(), "worktree") || !strings.Contains(f.out.String(), "assigned: me") {
		t.Fatalf("%d %s", c, f.out.String())
	}
	f.git.CurrentV = "PROJ-1-fix-login"
	if c := f.run("review"); c != 0 || !strings.Contains(f.out.String(), "MR: https://gh/pr/1") || !strings.Contains(f.out.String(), "state: In Review") {
		t.Fatalf("%d %s %s", c, f.out.String(), f.err.String())
	}
	if c := f.run("review", "PROJ-2"); c != 0 || !strings.Contains(f.out.String(), "MR:") {
		t.Fatalf("%d %s %s", c, f.out.String(), f.err.String())
	}
	f.git.CurrentV = "main"
	if c := f.run("review"); c != 1 {
		t.Fatalf("no item: %d", c)
	}
	f.git.Err["Create"] = errors.New("boom")
	if c := f.run("start", "PROJ-3"); c != 1 || !strings.Contains(f.err.String(), "platui: boom") {
		t.Fatalf("%d %s", c, f.err.String())
	}
}

func TestContractCLIPrompt(t *testing.T) {
	f := newFx(t)
	f.git.CurrentV = "PROJ-1-fix-login"
	f.run("ls") // греет кэш
	f.api.Mu.Lock()
	n := len(f.api.Calls)
	f.api.Mu.Unlock()
	if c := f.run("prompt"); c != 0 || strings.TrimSpace(f.out.String()) != "PROJ-1 Todo" || f.err.Len() != 0 {
		t.Fatalf("%d %q %q", c, f.out.String(), f.err.String())
	}
	f.api.Mu.Lock()
	if len(f.api.Calls) != n {
		t.Fatalf("network used: %v", f.api.Calls[n:])
	}
	f.api.Mu.Unlock()
	f.run("prompt", "--format", "[{key}|{state}]")
	if strings.TrimSpace(f.out.String()) != "[PROJ-1|Todo]" {
		t.Fatalf("%q", f.out.String())
	}
	f.git.CurrentV = "main"
	if c := f.run("prompt"); c != 0 || f.out.Len() != 0 || f.err.Len() != 0 {
		t.Fatalf("no data: %d %q %q", c, f.out.String(), f.err.String())
	}
	f.cfgYA = "workspace: ws\n" // нет ключа
	if c := f.run("prompt"); c != 0 || f.out.Len() != 0 || f.err.Len() != 0 {
		t.Fatalf("bad config: %d %q %q", c, f.out.String(), f.err.String())
	}
}

func TestContractCLIHookVersion(t *testing.T) {
	f := newFx(t)
	f.useGit()
	if c := f.run("hook", "install"); c != 0 || !f.git.HookInstalled {
		t.Fatalf("%d %s", c, f.err.String())
	}
	f.git.Err["InstallHook"] = errors.New("hook exists")
	if c := f.run("hook", "install"); c != 1 || !strings.Contains(f.err.String(), "platui: ") {
		t.Fatalf("%d %s", c, f.err.String())
	}
	if c := f.run("version"); c != 0 || f.out.String() != "platui dev\n" {
		t.Fatalf("%d %q", c, f.out.String())
	}
}

func TestContractCLIExitCodesAndTUI(t *testing.T) {
	f := newFx(t)
	for _, a := range [][]string{{"nope"}, {"--bogus"}, {"ls", "--bogus"}, {"show"}, {"ls", "extra"}} {
		if c := f.run(a...); c != 2 || !strings.Contains(f.err.String(), "platui: ") {
			t.Errorf("%v: code %d %s", a, c, f.err.String())
		}
	}
	for _, a := range [][]string{{"-h"}, {"ls", "--help"}, {"show", "-h"}} {
		if c := f.run(a...); c != 0 || f.out.Len() == 0 {
			t.Errorf("%v: code %d", a, c)
		}
	}
	if c := f.run("--project", "OTH", "--item", "OTH-7"); c != 0 || f.tui == nil || *f.tui != (StartAt{Project: "OTH", Item: "OTH-7"}) {
		t.Fatalf("%d %v", c, f.tui)
	}
	f.cfgYA = "workspace: ws\n"
	if c := f.run("ls"); c != 1 || !strings.Contains(f.err.String(), "platui: ") {
		t.Fatalf("%d %s", c, f.err.String())
	}
	f.cfgYA = "api_key: k\nworkspace: ws\n"
	if c := f.run("ls"); c != 1 || !strings.Contains(f.err.String(), "--project") {
		t.Fatalf("no project: %d %s", c, f.err.String())
	}
	e := f.env()
	e.RunTUI = nil
	if c := Run(context.Background(), []string{"--config", "/nonexistent.yml"}, e); c != 1 {
		t.Fatal(c)
	}
	e.Build = func(*config.Config, string) (*app.App, error) { return nil, errors.New("x") }
	if c := f.run("ls"); c != 1 {
		t.Fatal(c)
	}
	if c := Run(nil, []string{"version"}, Env{}); c != 0 { //nolint
		t.Fatal(c)
	}
}
