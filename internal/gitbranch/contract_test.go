package gitbranch

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestContractGitbranchSlug(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"Fix login bug", 0, "fix-login-bug"},
		{"  Hello,   World!! ", 0, "hello-world"},
		{"Исправить ошибку входа", 0, "ispravit-oshibku-vkhoda"},
		{"Щука, жёлтый ёж: ЦЧШ ЮЯ", 0, "shchuka-zhyoltyy-yozh-tschsh-yuya"},
		{"объект подъезд ыэ", 0, "obekt-podezd-ye"},
		{"a---b___c", 0, "a-b-c"},
		{"!!!", 0, "task"},
		{"", 0, "task"},
		{"abcdef-ghij", 7, "abcdef"},
		{"abcdef-ghij", 8, "abcdef-g"},
		{strings.Repeat("a", 100), 0, strings.Repeat("a", 40)},
		{"Ёж", -1, "yozh"},
	}
	for _, c := range cases {
		if got := Slug(c.in, c.max); got != c.want {
			t.Errorf("Slug(%q,%d)=%q want %q", c.in, c.max, got, c.want)
		}
	}
}

func TestContractGitbranchRender(t *testing.T) {
	got, err := Render("{identifier}-{sequence}-{slug}", Item{"PROJ", 12, "Fix login"})
	if err != nil || got != "PROJ-12-fix-login" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := Render("bad name {slug}", Item{"P", 1, "x"}); err == nil {
		t.Error("expected error for name with space")
	}
	if _, err := Render("{identifier}-{foo}", Item{"P", 1, "x"}); err == nil {
		t.Error("expected error for unknown placeholder")
	}
}

func TestContractGitbranchValidName(t *testing.T) {
	for _, n := range []string{"feat/x", "PROJ-1-a"} {
		if err := ValidName(n); err != nil {
			t.Errorf("%q: %v", n, err)
		}
	}
	for _, n := range []string{"", "a b", "a..b", "-x", "x.lock", "a~b", "HEAD", "a:b", "a/"} {
		if err := ValidName(n); err == nil {
			t.Errorf("%q: expected error", n)
		}
	}
}

func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func newRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	dir := t.TempDir()
	run(t, dir, "init", "-q")
	run(t, dir, "config", "user.name", "T")
	run(t, dir, "config", "user.email", "t@example.com")
	run(t, dir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "add", "f")
	run(t, dir, "commit", "-q", "-m", "init")
	return dir
}

func TestContractGitbranchNotRepo(t *testing.T) {
	if _, err := Open(t.TempDir()); !errors.Is(err, ErrNotRepo) {
		t.Fatalf("got %v", err)
	}
}

func TestContractGitbranchRepo(t *testing.T) {
	dir := newRepo(t)
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := r.Exists("feat-a"); err != nil || ok {
		t.Fatalf("exists: %v %v", ok, err)
	}
	res, err := r.Create("feat-a", Options{})
	if err != nil || !res.Created || res.Switched || res.Dirty {
		t.Fatalf("create: %+v %v", res, err)
	}
	if ok, _ := r.Exists("feat-a"); !ok {
		t.Fatal("should exist")
	}
	res, err = r.Create("feat-a", Options{})
	if !errors.Is(err, ErrBranchExists) || res.Created || res.Name != "feat-a" || res.Dirty {
		t.Fatalf("dup: %+v %v", res, err)
	}

	// dirty tree does not block create+checkout
	if err := os.WriteFile(filepath.Join(dir, "g"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if d, err := r.Dirty(); err != nil || !d {
		t.Fatalf("dirty: %v %v", d, err)
	}
	res, err = r.Create("feat-b", Options{Checkout: true})
	if err != nil || !res.Created || !res.Switched || !res.Dirty {
		t.Fatalf("create b: %+v %v", res, err)
	}
	if cur := run(t, dir, "rev-parse", "--abbrev-ref", "HEAD"); cur != "feat-b" {
		t.Fatalf("HEAD=%s", cur)
	}
	res, err = r.Switch("feat-a")
	if err != nil || res.Created || !res.Switched {
		t.Fatalf("switch: %+v %v", res, err)
	}
	if _, err := r.Switch("nope"); err == nil {
		t.Fatal("switch to missing branch should fail")
	}
	base := run(t, dir, "rev-parse", "--abbrev-ref", "HEAD")
	if _, err := r.Create("feat-c", Options{Base: base}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Create("feat-d", Options{Base: "no-such-base"}); err == nil {
		t.Fatal("bad base should fail")
	}
	var _ Brancher = r
}
