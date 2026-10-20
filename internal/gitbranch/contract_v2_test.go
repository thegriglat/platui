package gitbranch

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const defTpl = "{identifier}-{sequence}-{slug}"

func commitFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "add", name)
	run(t, dir, "commit", "-q", "-m", "c "+name)
}

func openRepo(t *testing.T, dir string) *Repo {
	t.Helper()
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// repoWithRemote: repo on main with one commit, bare origin, main pushed.
func repoWithRemote(t *testing.T) (*Repo, string, string) {
	t.Helper()
	dir := newRepo(t)
	run(t, dir, "branch", "-M", "main")
	bare := t.TempDir()
	run(t, bare, "init", "-q", "--bare")
	run(t, dir, "remote", "add", "origin", bare)
	r := openRepo(t, dir)
	if err := r.Push("origin", "main"); err != nil {
		t.Fatal(err)
	}
	return r, dir, bare
}

func TestContractGitbranchV2ParseBranch(t *testing.T) {
	cases := []struct {
		tpl, name string
		want      Item
		ok        bool
	}{
		{defTpl, "PROJ-12-fix-login", Item{"PROJ", 12, ""}, true},
		{defTpl, "proj-12-fix", Item{"PROJ", 12, ""}, true},
		{defTpl, "feature/PROJ-12-x", Item{"PROJ", 12, ""}, true},
		{"feature/{identifier}-{sequence}", "feature/AB2-7", Item{"AB2", 7, ""}, true},
		{"{slug}", "PROJ-5_x", Item{"PROJ", 5, ""}, true},
		{defTpl, "main", Item{}, false},
		{defTpl, "fix-login", Item{}, false},
		{defTpl, "PROJ-x", Item{}, false},
	}
	for _, c := range cases {
		it, ok := ParseBranch(c.tpl, c.name)
		if ok != c.ok || it != c.want {
			t.Errorf("ParseBranch(%q,%q)=%+v,%v want %+v,%v", c.tpl, c.name, it, ok, c.want, c.ok)
		}
	}
}

func TestContractGitbranchV2CommitMessage(t *testing.T) {
	it := Item{"PROJ", 12, "Fix\nlogin  bug"}
	got, err := CommitMessage("{identifier}-{sequence}: {title} ({slug})", it)
	if err != nil || got != "PROJ-12: Fix login bug (fix-login-bug)" {
		t.Fatalf("got %q %v", got, err)
	}
	if _, err := CommitMessage("{nope}", it); err == nil {
		t.Fatal("expected error")
	}
}

func TestContractGitbranchV2WorktreePath(t *testing.T) {
	if got := WorktreePath("../{repo}-{branch}", "/home/u/proj", "feat/PROJ-1"); got != "/home/u/proj-feat-PROJ-1" {
		t.Errorf("got %q", got)
	}
	if got := WorktreePath(".wt/{branch}", "/home/u/proj", "a"); got != "/home/u/proj/.wt/a" {
		t.Errorf("got %q", got)
	}
	if got := WorktreePath("/tmp/x/../{repo}", "/home/u/proj", "a"); got != "/tmp/proj" {
		t.Errorf("got %q", got)
	}
}

func TestContractGitbranchV2Current(t *testing.T) {
	r := openRepo(t, newRepo(t))
	cur, err := r.Current()
	if err != nil || cur == "" {
		t.Fatalf("%q %v", cur, err)
	}
	if top, err := r.Toplevel(); err != nil || top == "" {
		t.Fatalf("top %q %v", top, err)
	}
	run(t, r.Dir, "checkout", "-q", "--detach")
	if _, err := r.Current(); !errors.Is(err, ErrDetached) {
		t.Fatalf("got %v", err)
	}
	if names, err := r.Branches(); err != nil || len(names) != 1 {
		t.Fatalf("branches %v %v", names, err)
	}
}

func TestContractGitbranchV2Statuses(t *testing.T) {
	r, dir, bare := repoWithRemote(t)
	if _, err := r.Create("feat", Options{Checkout: true}); err != nil {
		t.Fatal(err)
	}
	st, err := r.Statuses([]string{"feat", "main", "ghost"}, "origin", "")
	if err != nil {
		t.Fatal(err)
	}
	if s := st["feat"]; !s.Local || s.Remote || !s.Current {
		t.Errorf("feat: %+v", s)
	}
	if s := st["ghost"]; s.Local || s.Remote || s.Ahead != 0 {
		t.Errorf("ghost: %+v", s)
	}
	if err := r.Push("origin", "feat"); err != nil {
		t.Fatal(err)
	}
	// feat: 2 ahead; main: 1 ahead of feat's fork point
	commitFile(t, dir, "a", "a")
	commitFile(t, dir, "b", "b")
	run(t, dir, "checkout", "-q", "main")
	commitFile(t, dir, "m", "m")
	st, _ = r.Statuses([]string{"feat", "main"}, "origin", "main")
	if s := st["feat"]; !s.Remote || s.Ahead != 2 || s.Behind != 1 || s.Current {
		t.Errorf("feat: %+v", s)
	}
	if s := st["main"]; !s.Current || !s.Remote {
		t.Errorf("main: %+v", s)
	}
	// base auto-detected via local main
	st, _ = r.Statuses(nil, "origin", "")
	if s := st["feat"]; s.Ahead != 2 || s.Behind != 1 {
		t.Errorf("auto base: %+v", s)
	}
	// base via origin/HEAD
	run(t, dir, "remote", "set-head", "origin", "main")
	st, _ = r.Statuses([]string{"feat"}, "origin", "")
	if s := st["feat"]; s.Ahead != 2 {
		t.Errorf("origin head base: %+v", s)
	}
	// worktree
	wt := filepath.Join(t.TempDir(), "wt")
	if _, err := r.AddWorktree("feat", wt, Options{}); err != nil {
		t.Fatal(err)
	}
	st, _ = r.Statuses([]string{"feat"}, "origin", "")
	if !samePath(st["feat"].Worktree, wt) {
		t.Errorf("worktree: %+v", st["feat"])
	}
	_ = bare
}

func TestContractGitbranchV2Worktree(t *testing.T) {
	r := openRepo(t, newRepo(t))
	base := t.TempDir()
	p1 := filepath.Join(base, "new")
	res, err := r.AddWorktree("wt-new", p1, Options{})
	if err != nil || !res.Created || res.Switched || res.Path != p1 {
		t.Fatalf("new: %+v %v", res, err)
	}
	if _, err := os.Stat(filepath.Join(p1, "f")); err != nil {
		t.Fatal(err)
	}
	run(t, r.Dir, "branch", "wt-old")
	p2 := filepath.Join(base, "old")
	res, err = r.AddWorktree("wt-old", p2, Options{})
	if err != nil || res.Created || res.Path != p2 {
		t.Fatalf("old: %+v %v", res, err)
	}
	res, err = r.AddWorktree("wt-old", filepath.Join(base, "again"), Options{})
	if !errors.Is(err, ErrWorktreeBusy) || !samePath(res.Path, p2) {
		t.Fatalf("busy: %+v %v", res, err)
	}
	cur, _ := r.Current()
	if _, err := r.AddWorktree(cur, filepath.Join(base, "cur"), Options{}); !errors.Is(err, ErrWorktreeBusy) {
		t.Fatalf("current busy: %v", err)
	}
	if err := r.Delete("wt-old", true); !errors.Is(err, ErrWorktreeBusy) {
		t.Fatalf("delete busy: %v", err)
	}
}

func TestContractGitbranchV2Delete(t *testing.T) {
	r, dir, _ := repoWithRemote(t)
	if err := r.Delete("main", false); !errors.Is(err, ErrCurrentBranch) {
		t.Fatalf("current: %v", err)
	}
	run(t, dir, "branch", "merged")
	if err := r.Delete("merged", false); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "checkout", "-q", "-b", "wip")
	commitFile(t, dir, "w", "w")
	run(t, dir, "checkout", "-q", "main")
	if err := r.Delete("wip", false); !errors.Is(err, ErrNotMerged) {
		t.Fatalf("not merged: %v", err)
	}
	if err := r.Delete("wip", true); err != nil {
		t.Fatal(err)
	}
	if ok, _ := r.Exists("wip"); ok {
		t.Fatal("still exists")
	}
}

func TestContractGitbranchV2Hook(t *testing.T) {
	dir := newRepo(t)
	r := openRepo(t, dir)
	path, inst, err := r.InstallHook()
	if err != nil || !inst {
		t.Fatalf("%v %v", inst, err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o755 {
		t.Fatalf("mode %v %v", fi, err)
	}
	if _, inst, err := r.InstallHook(); err != nil || inst {
		t.Fatalf("repeat: %v %v", inst, err)
	}
	// outdated platui hook is rewritten
	old := "#!/bin/sh\n# platui prepare-commit-msg v1\n# old\n"
	os.WriteFile(path, []byte(old), 0o755)
	if _, inst, err := r.InstallHook(); err != nil || !inst {
		t.Fatalf("rewrite: %v %v", inst, err)
	}
	// foreign hook
	os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755)
	if _, _, err := r.InstallHook(); !errors.Is(err, ErrHookExists) {
		t.Fatalf("foreign: %v", err)
	}
	if b, _ := os.ReadFile(path); string(b) != "#!/bin/sh\nexit 0\n" {
		t.Fatal("foreign hook changed")
	}
	os.Remove(path)

	// core.hooksPath
	hp := filepath.Join(t.TempDir(), "hooks")
	run(t, dir, "config", "core.hooksPath", hp)
	p2, inst, err := r.InstallHook()
	if err != nil || !inst || p2 != filepath.Join(hp, "prepare-commit-msg") {
		t.Fatalf("hooksPath: %q %v %v", p2, inst, err)
	}
}

func commitEditor(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "commit", "-q", "--allow-empty"}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_EDITOR=true")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v: %s", err, out)
	}
	return run(t, dir, "log", "-1", "--format=%B")
}

func TestContractGitbranchV2HookRuns(t *testing.T) {
	dir := newRepo(t)
	r := openRepo(t, dir)
	if _, _, err := r.InstallHook(); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "checkout", "-q", "-b", "feature/PROJ-1-x")
	if err := r.WriteCommitMsg("feature/PROJ-1-x", "PROJ-1: do x"); err != nil {
		t.Fatal(err)
	}
	if got := commitEditor(t, dir, "-t", "/dev/null"); !strings.HasPrefix(got, "PROJ-1: do x") {
		t.Errorf("template commit: %q", got)
	}
	if got := commitEditor(t, dir); !strings.HasPrefix(got, "PROJ-1: do x") {
		t.Errorf("editor commit: %q", got)
	}
	if got := commitEditor(t, dir, "-m", "plain"); got != "plain" {
		t.Errorf("-m commit: %q", got)
	}
	run(t, dir, "checkout", "-q", "-b", "other")
	if got := commitEditor(t, dir, "-m", "other msg"); got != "other msg" {
		t.Errorf("other branch: %q", got)
	}
}

func TestContractGitbranchV2Remote(t *testing.T) {
	r, dir, bare := repoWithRemote(t)
	if u, err := r.RemoteURL("origin"); err != nil || u != bare {
		t.Fatalf("url %q %v", u, err)
	}
	if _, err := r.RemoteURL("nope"); err == nil {
		t.Fatal("expected error")
	}
	run(t, dir, "checkout", "-q", "-b", "pushed")
	commitFile(t, dir, "p", "p")
	if err := r.Push("origin", "pushed"); err != nil {
		t.Fatal(err)
	}
	if got := run(t, bare, "branch", "--list", "pushed"); !strings.Contains(got, "pushed") {
		t.Fatalf("bare: %q", got)
	}
	if err := r.Push("nope", "pushed"); err == nil {
		t.Fatal("expected error")
	}
}

func TestContractGitbranchV2HookWorktree(t *testing.T) {
	dir := newRepo(t)
	r := openRepo(t, dir)
	if _, _, err := r.InstallHook(); err != nil {
		t.Fatal(err)
	}
	wt := filepath.Join(t.TempDir(), "wt")
	if _, err := r.AddWorktree("feature/PROJ-2-y", wt, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteCommitMsg("feature/PROJ-2-y", "PROJ-2: do y"); err != nil {
		t.Fatal(err)
	}
	if got := commitEditor(t, wt); !strings.HasPrefix(got, "PROJ-2: do y") {
		t.Errorf("worktree commit: %q", got)
	}
	if got := commitEditor(t, wt, "-m", "plain"); got != "plain" {
		t.Errorf("-m in worktree: %q", got)
	}
}

func TestContractGitbranchV2DeleteLocale(t *testing.T) {
	t.Setenv("LANG", "ru_RU.UTF-8")
	t.Setenv("LC_ALL", "ru_RU.UTF-8")
	r, dir, _ := repoWithRemote(t)
	run(t, dir, "checkout", "-q", "-b", "wip")
	commitFile(t, dir, "w", "w")
	run(t, dir, "checkout", "-q", "main")
	if err := r.Delete("wip", false); !errors.Is(err, ErrNotMerged) {
		t.Fatalf("not merged: %v", err)
	}
}

func TestContractGitbranchV4ReadCommitMsg(t *testing.T) {
	r := openRepo(t, newRepo(t))
	if msg, ok, err := r.ReadCommitMsg("feature/PROJ-1-x"); err != nil || ok || msg != "" {
		t.Fatalf("missing: %q %v %v", msg, ok, err)
	}
	if err := r.WriteCommitMsg("feature/PROJ-1-x", "PROJ-1: do x"); err != nil {
		t.Fatal(err)
	}
	if msg, ok, err := r.ReadCommitMsg("feature/PROJ-1-x"); err != nil || !ok || msg != "PROJ-1: do x" {
		t.Fatalf("read: %q %v %v", msg, ok, err)
	}
}
