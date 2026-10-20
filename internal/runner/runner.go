// Package runner runs the user's shell commands (contract C11 v1).
// A leaf package: it does not depend on other platui packages.
package runner

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Result is the outcome of a run.
type Result struct {
	Stdout, Stderr string
	ExitCode       int
}

// Shell runs `sh -c command` in directory dir with extra env.
// An error is returned only if the command could not be started (or ctx was
// canceled); a non-zero exit code goes into ExitCode.
type Shell func(ctx context.Context, dir, command string, env []string) (Result, error)

// Default is the os/exec implementation: cwd=dir, env = os.Environ()+env.
func Default() Shell {
	return func(ctx context.Context, dir, command string, env []string) (Result, error) {
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Dir = dir
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) } // the whole group, including sh's children
		cmd.WaitDelay = time.Second
		cmd.Env = append(os.Environ(), env...)
		var out, errb bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errb
		err := cmd.Run()
		res := Result{Stdout: out.String(), Stderr: errb.String()}
		if err == nil {
			return res, nil
		}
		if ctx.Err() != nil {
			res.ExitCode = -1
			return res, ctx.Err()
		}
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			res.ExitCode = ee.ExitCode()
			return res, nil
		}
		return res, err
	}
}
