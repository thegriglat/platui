package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContractCLITodo(t *testing.T) {
	f := newFx(t)
	file := filepath.Join(t.TempDir(), "todo.md")
	if err := os.WriteFile(file, []byte("# my\n- [ ] one\ntext\n- [x] two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// без ключа API и без сети
	f.cfgYA = "workspace: ws\ntodo:\n  path: " + file + "\n"
	if c := f.run("todo", "add", "buy", "milk"); c != 0 {
		t.Fatalf("add: %d %s", c, f.err.String())
	}
	if c := f.run("todo", "done", "1"); c != 0 {
		t.Fatalf("done: %d %s", c, f.err.String())
	}
	if c := f.run("todo", "undone", "2"); c != 0 {
		t.Fatalf("undone: %d %s", c, f.err.String())
	}
	if c := f.run("todo"); c != 0 {
		t.Fatalf("ls: %d %s", c, f.err.String())
	}
	want := "1. [x] one\n2. [ ] two\n3. [ ] buy milk\n"
	if f.out.String() != want {
		t.Fatalf("got %q want %q", f.out.String(), want)
	}
	b, _ := os.ReadFile(file)
	if !strings.HasPrefix(string(b), "# my\n- [x] one\ntext\n- [ ] two\n- [ ] buy milk") {
		t.Fatalf("file: %q", b)
	}
	if c := f.run("todo", "done", "9"); c != 1 {
		t.Fatalf("bad N: %d", c)
	}
	if c := f.run("todo", "done", "x"); c != 2 {
		t.Fatalf("bad arg: %d", c)
	}
	if f.built != 0 {
		t.Fatal("app must not be built")
	}
}

func TestCLITodoBadConfig(t *testing.T) {
	f := newFx(t)
	f.cfgYA = "workspace: \"\"\n"
	if c := f.run("todo"); c == 0 {
		t.Fatal("ошибка конфига не из-за ключа должна провалить todo")
	}
}
