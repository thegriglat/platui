package todo

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func write(t *testing.T, s string) string {
	p := filepath.Join(t.TempDir(), "sub", "todo.md")
	if s != "" {
		os.MkdirAll(filepath.Dir(p), 0o700)
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func content(t *testing.T, p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestContractTodoParse(t *testing.T) {
	p := write(t, "# Title\n- [ ] a\n* [x] b\n+ [X] c PROJ-1\n  - [ ] indented\ntext\n- [] no\n")
	l, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	want := []Item{{1, "a", false}, {2, "b", true}, {3, "c PROJ-1", true}, {4, "indented", false}}
	if !reflect.DeepEqual(l.Items(), want) {
		t.Fatalf("%+v", l.Items())
	}
}

func TestContractTodoEditPreserves(t *testing.T) {
	src := "# T\n\n- [ ] a\nnote\n* [x] b\n\ntail\n"
	p := write(t, src)
	l, _ := Load(p)
	if err := l.Save(); err != nil || content(t, p) != src {
		t.Fatalf("%v %q", err, content(t, p))
	}
	l.Add("new\nline")
	l.SetDone(1, true)
	l.Edit(2, "bb")
	if err := l.Save(); err != nil {
		t.Fatal(err)
	}
	want := "# T\n\n- [x] a\nnote\n* [x] bb\n- [ ] new line\n\ntail\n"
	if got := content(t, p); got != want {
		t.Fatalf("%q", got)
	}
	l.SetDone(1, false)
	l.Remove(2)
	if err := l.Save(); err != nil {
		t.Fatal(err)
	}
	if got := content(t, p); got != "# T\n\n- [ ] a\nnote\n- [ ] new line\n\ntail\n" {
		t.Fatalf("%q", got)
	}
}

func TestContractTodoMove(t *testing.T) {
	p := write(t, "- [ ] a\nx\n- [ ] b\ny\n- [ ] c\n")
	l, _ := Load(p)
	if err := l.Move(3, 1); err != nil {
		t.Fatal(err)
	}
	l.Save()
	if got := content(t, p); got != "- [ ] c\nx\n- [ ] a\ny\n- [ ] b\n" {
		t.Fatalf("%q", got)
	}
	l.Move(1, 3)
	l.Save()
	if got := content(t, p); got != "- [ ] a\nx\n- [ ] b\ny\n- [ ] c\n" {
		t.Fatalf("%q", got)
	}
	for _, e := range []error{l.Move(0, 1), l.Move(1, 9), l.Remove(9), l.SetDone(0, true), l.Edit(4, "x")} {
		if !errors.Is(e, ErrNoItem) {
			t.Fatal(e)
		}
	}
}

func TestContractTodoMissingAndNoNewline(t *testing.T) {
	p := write(t, "")
	l, err := Load(p)
	if err != nil || len(l.Items()) != 0 {
		t.Fatal(err)
	}
	l.Add("one")
	if err := l.Save(); err != nil {
		t.Fatal(err)
	}
	if got := content(t, p); got != "- [ ] one\n" {
		t.Fatalf("%q", got)
	}
	st, _ := os.Stat(p)
	if st.Mode().Perm() != 0o644 {
		t.Fatal(st.Mode())
	}
	d, _ := os.Stat(filepath.Dir(p))
	if d.Mode().Perm() != 0o700 {
		t.Fatal(d.Mode())
	}
	p2 := write(t, "text\n- [ ] a")
	l, _ = Load(p2)
	l.Add("b")
	l.Save()
	if got := content(t, p2); got != "text\n- [ ] a\n- [ ] b" {
		t.Fatalf("%q", got)
	}
}

func TestContractTodoErrChanged(t *testing.T) {
	p := write(t, "- [ ] a\n")
	l, _ := Load(p)
	os.WriteFile(p, []byte("- [ ] a\n- [ ] hand\n"), 0o644)
	l.Add("mine")
	if err := l.Save(); !errors.Is(err, ErrChanged) {
		t.Fatal(err)
	}
	if got := content(t, p); got != "- [ ] a\n- [ ] hand\n" {
		t.Fatalf("%q", got)
	}
	// second save after success works with the new fingerprint
	l, _ = Load(p)
	l.Add("x")
	if err := l.Save(); err != nil {
		t.Fatal(err)
	}
	l.Add("y")
	if err := l.Save(); err != nil {
		t.Fatal(err)
	}
	// file created by someone after Load of a missing file
	q := write(t, "")
	l, _ = Load(q)
	os.MkdirAll(filepath.Dir(q), 0o700)
	os.WriteFile(q, []byte("z\n"), 0o644)
	if err := l.Save(); !errors.Is(err, ErrChanged) {
		t.Fatal(err)
	}
}

func TestContractTodoKeysAndPath(t *testing.T) {
	got := Keys("fix PROJ-42, (AB2-7) and PROJ-42 xPROJ-1 foo-3")
	if !reflect.DeepEqual(got, []string{"PROJ-42", "AB2-7"}) {
		t.Fatalf("%v", got)
	}
	if Keys("none") != nil {
		t.Fatal("expected nil")
	}
	t.Setenv("XDG_DATA_HOME", "/d")
	if p, _ := DefaultPath(); p != "/d/platui/todo.md" {
		t.Fatal(p)
	}
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/h")
	if p, _ := DefaultPath(); p != "/h/.local/share/platui/todo.md" {
		t.Fatal(p)
	}
}

func TestLoadReadError(t *testing.T) {
	if _, err := Load(t.TempDir()); err == nil {
		t.Fatal("expected error")
	}
	p := write(t, "- [ ] a\n")
	l, _ := Load(p)
	l.path = t.TempDir()
	if err := l.Save(); err == nil {
		t.Fatal("expected error")
	}
	l.path = filepath.Join(p, "x", "y")
	if err := l.Save(); err == nil {
		t.Fatal("expected error")
	}
}
