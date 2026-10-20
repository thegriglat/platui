package gitbranch

import (
	"fmt"
	"path/filepath"
	"strings"
)

// worktrees maps branch name to the path of the worktree where it is
// checked out (all worktrees, including this one).
func (r *Repo) worktrees() (map[string]string, error) {
	out, err := r.gitOut("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	res := map[string]string{}
	path := ""
	for _, l := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(l, "worktree "):
			path = strings.TrimPrefix(l, "worktree ")
		case strings.HasPrefix(l, "branch refs/heads/"):
			res[strings.TrimPrefix(l, "branch refs/heads/")] = path
		}
	}
	return res, nil
}

// AddWorktree checks out name in a separate worktree at path. A missing
// branch is created from o.Base (empty: HEAD). If the branch is already
// checked out somewhere, the result carries that path and ErrWorktreeBusy.
func (r *Repo) AddWorktree(name, path string, o Options) (Result, error) {
	res := Result{Name: name}
	if err := ValidName(name); err != nil {
		return res, err
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return res, err
	}
	wts, err := r.worktrees()
	if err != nil {
		return res, err
	}
	if p, ok := wts[name]; ok {
		res.Path = p
		return res, fmt.Errorf("%w: %s", ErrWorktreeBusy, p)
	}
	exists, err := r.Exists(name)
	if err != nil {
		return res, err
	}
	args := []string{"worktree", "add"}
	if exists {
		args = append(args, abs, name)
	} else {
		args = append(args, "-b", name, abs)
		if o.Base != "" {
			args = append(args, o.Base)
		}
	}
	if _, _, err := r.git(args...); err != nil {
		return res, err
	}
	res.Created = !exists
	res.Path = abs
	return res, nil
}
