package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestContractConfigV6(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/x/data")
	c, err := Parse([]byte(v5Base))
	if err != nil {
		t.Fatal(err)
	}
	p, err := c.TodoPath()
	if err != nil || p != filepath.Join("/x/data", "platui", "todo.md") {
		t.Fatalf("%q %v", p, err)
	}
	c, err = Parse([]byte(v5Base + "todo:\n  path: /tmp/my.md\n"))
	if err != nil {
		t.Fatal(err)
	}
	if p, _ := c.TodoPath(); p != "/tmp/my.md" {
		t.Fatal(p)
	}
}

func TestErrNoAPIKey(t *testing.T) {
	t.Setenv("PT44_EMPTY", "")
	for _, y := range []string{"workspace: ws\n", "workspace: ws\napi_key_env: PT44_EMPTY\n"} {
		if _, err := Parse([]byte(y)); !errors.Is(err, ErrNoAPIKey) {
			t.Errorf("%q: %v", y, err)
		}
	}
	p := filepath.Join(t.TempDir(), "c.yml")
	if err := os.WriteFile(p, []byte("workspace: ws\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); !errors.Is(err, ErrNoAPIKey) {
		t.Errorf("Load: %v", err)
	}
	c, err := LoadNoKey(p)
	if err != nil || c.Workspace != "ws" {
		t.Fatalf("LoadNoKey: %v", err)
	}
	if err := os.WriteFile(p, []byte("workspace: \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadNoKey(p); err == nil {
		t.Error("валидация должна работать без ключа")
	}
}
