package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/thegriglat/platui/internal/runner"
)

// Реальный sh: подставленные значения с кавычками и $() не интерпретируются.
func TestRunCommandRealShellQuoting(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("нет sh")
	}
	tmpl := "printf '%s|%s' {title} {branch}"
	v := CommandVars{Title: `it's $(echo x) "q" ; rm $(touch PWNED)`, Branch: "b`touch PWNED`"}
	dir := t.TempDir()
	out, err := runner.Default()(context.Background(), dir, RenderCommand(tmpl, v), nil)
	if err != nil || out.ExitCode != 0 || out.Stdout != v.Title+"|"+v.Branch {
		t.Fatalf("%q %v %+v", out.Stdout, err, out)
	}
	if _, err := os.Stat(filepath.Join(dir, "PWNED")); err == nil {
		t.Fatal("подстановка выполнилась: файл PWNED создан")
	}
}
