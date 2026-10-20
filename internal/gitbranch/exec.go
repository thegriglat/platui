package gitbranch

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// git runs a git command in r.Dir; code is the exit code (-1 if git did
// not run). stderr of git is part of the error text.
func (r *Repo) git(args ...string) (string, int, error) {
	cmd := exec.Command("git", append([]string{"-C", r.Dir}, args...)...)
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return out.String(), ee.ExitCode(), fmt.Errorf("git %s: %w: %s",
				args[0], err, strings.TrimSpace(stderr.String()))
		}
		return out.String(), -1, fmt.Errorf("git %s: %w", args[0], err)
	}
	return out.String(), 0, nil
}

// gitOut runs git and returns trimmed stdout.
func (r *Repo) gitOut(args ...string) (string, error) {
	out, _, err := r.git(args...)
	return strings.TrimSpace(out), err
}
