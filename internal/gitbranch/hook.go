package gitbranch

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const hookMarker = "# platui prepare-commit-msg v2"

// hookMarkerPrefix identifies any platui hook version as our own.
const hookMarkerPrefix = "# platui prepare-commit-msg v"

const hookScript = `#!/bin/sh
` + hookMarker + `
case "$2" in
  ""|template) ;;
  *) exit 0 ;;
esac
branch=$(git symbolic-ref --short -q HEAD) || exit 0
common=$(git rev-parse --path-format=absolute --git-common-dir) || exit 0
f="$common/platui/commit-msg/$branch"
[ -f "$f" ] || exit 0
msg=$(cat "$f")
[ -n "$msg" ] || exit 0
first=$(head -n 1 "$1")
case "$first" in
  "$msg"*) exit 0 ;;
esac
tmp="$1.platui.$$"
{ printf '%s\n' "$msg"; cat "$1"; } > "$tmp" && mv "$tmp" "$1"
exit 0
`

// gitPath resolves `git rev-parse --git-path p` to an absolute path.
func (r *Repo) gitPath(p string) (string, error) {
	out, err := r.gitOut("rev-parse", "--git-path", p)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(out) {
		out = filepath.Join(r.Dir, out)
	}
	return out, nil
}

// commitMsgFile returns the path of the stored commit message for branch.
func (r *Repo) commitMsgFile(branch string) (string, error) {
	if err := ValidName(branch); err != nil {
		return "", err
	}
	common, err := r.gitOut("rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}
	return filepath.Join(common, "platui", "commit-msg", filepath.FromSlash(branch)), nil
}

// ReadCommitMsg reads the stored commit message; a missing file gives ok=false.
func (r *Repo) ReadCommitMsg(branch string) (msg string, ok bool, err error) {
	file, err := r.commitMsgFile(branch)
	if err != nil {
		return "", false, err
	}
	b, err := os.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("commit-msg: %w", err)
	}
	return string(b), true, nil
}

// WriteCommitMsg stores the commit message for branch (atomic write).
func (r *Repo) WriteCommitMsg(branch, msg string) error {
	file, err := r.commitMsgFile(branch)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return fmt.Errorf("commit-msg: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(file), ".tmp-")
	if err != nil {
		return fmt.Errorf("commit-msg: %w", err)
	}
	_, werr := tmp.WriteString(msg)
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("commit-msg: %w", err)
	}
	if err := os.Rename(tmp.Name(), file); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("commit-msg: %w", err)
	}
	return nil
}

// InstallHook installs the prepare-commit-msg hook. installed is false when
// the same hook was already in place; a foreign hook gives ErrHookExists.
func (r *Repo) InstallHook() (path string, installed bool, err error) {
	path, err = r.gitPath("hooks/prepare-commit-msg")
	if err != nil {
		return "", false, err
	}
	old, rerr := os.ReadFile(path)
	switch {
	case rerr == nil:
		if !bytes.Contains(old, []byte(hookMarkerPrefix)) {
			return path, false, fmt.Errorf("%w: %s", ErrHookExists, path)
		}
		if string(old) == hookScript {
			return path, false, nil
		}
	case !errors.Is(rerr, os.ErrNotExist):
		return path, false, rerr
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path, false, err
	}
	if err := os.WriteFile(path, []byte(hookScript), 0o755); err != nil {
		return path, false, err
	}
	if err := os.Chmod(path, 0o755); err != nil {
		return path, false, err
	}
	return path, true, nil
}
