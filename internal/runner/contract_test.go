package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestContractRunnerOutputAndExit(t *testing.T) {
	r, err := Default()(context.Background(), "", "echo out; echo err >&2; exit 3", nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Stdout != "out\n" || r.Stderr != "err\n" || r.ExitCode != 3 {
		t.Fatalf("%+v", r)
	}
}

func TestContractRunnerEnvAndDir(t *testing.T) {
	dir := t.TempDir()
	r, err := Default()(context.Background(), dir, "echo $PT35_X; pwd", []string{"PT35_X=hello"})
	if err != nil || r.ExitCode != 0 {
		t.Fatalf("%+v %v", r, err)
	}
	if !strings.HasPrefix(r.Stdout, "hello\n") || !strings.Contains(r.Stdout, dir) {
		t.Fatalf("%q", r.Stdout)
	}
	r, _ = Default()(context.Background(), "", "echo $PATH", nil)
	if strings.TrimSpace(r.Stdout) == "" {
		t.Fatal("environ not inherited")
	}
}

func TestContractRunnerCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := Default()(ctx, "", "sleep 5", nil)
	if err == nil || time.Since(start) > 3*time.Second {
		t.Fatalf("err=%v after %v", err, time.Since(start))
	}
}

func TestRunnerBadDir(t *testing.T) {
	if _, err := Default()(context.Background(), "/nonexistent-dir-x", "true", nil); err == nil {
		t.Fatal("want error")
	}
}
