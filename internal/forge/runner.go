package forge

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Runner runs an external command with extra environment variables.
// The error contains stderr; stdout is always returned.
type Runner func(ctx context.Context, dir string, env []string, name string, args ...string) (stdout string, err error)

// ExecRunner runs commands via os/exec with LC_ALL=C and env.
func ExecRunner() Runner {
	return func(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = dir
		cmd.Env = append(append(os.Environ(), "LC_ALL=C"), env...)
		var out, errb bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errb
		if err := cmd.Run(); err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				return out.String(), ErrNotFound
			}
			return out.String(), fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(errb.String()))
		}
		return out.String(), nil
	}
}
