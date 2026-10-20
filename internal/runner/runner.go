// Package runner запускает shell-команды пользователя (контракт C11 v1).
// Листовой пакет: не зависит от других пакетов platui.
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

// Result — итог запуска.
type Result struct {
	Stdout, Stderr string
	ExitCode       int
}

// Shell запускает `sh -c command` в каталоге dir с дополнительным env.
// Ошибка — только если команду не удалось запустить (или ctx отменён);
// ненулевой код выхода — в ExitCode.
type Shell func(ctx context.Context, dir, command string, env []string) (Result, error)

// Default — реализация на os/exec: cwd=dir, env = os.Environ()+env.
func Default() Shell {
	return func(ctx context.Context, dir, command string, env []string) (Result, error) {
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Dir = dir
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) } // вся группа, включая детей sh
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
