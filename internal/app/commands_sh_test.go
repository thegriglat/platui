package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/thegriglat/platui/internal/runner"
)

// Real sh: substituted values with quotes and $() are not interpreted.
func TestRunCommandRealShellQuoting(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh")
	}
	tmpl := "printf '%s|%s' {title} {branch}"
	v := CommandVars{Title: `it's $(echo x) "q" ; rm $(touch PWNED)`, Branch: "b`touch PWNED`"}
	dir := t.TempDir()
	out, err := runner.Default()(context.Background(), dir, RenderCommand(tmpl, v), nil)
	if err != nil || out.ExitCode != 0 || out.Stdout != v.Title+"|"+v.Branch {
		t.Fatalf("%q %v %+v", out.Stdout, err, out)
	}
	if _, err := os.Stat(filepath.Join(dir, "PWNED")); err == nil {
		t.Fatal("substitution was executed: file PWNED created")
	}
}
