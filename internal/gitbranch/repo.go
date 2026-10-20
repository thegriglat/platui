package gitbranch

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Options controls branch creation.
type Options struct {
	Base     string // empty: current HEAD
	Checkout bool
}

// Result describes what was done.
type Result struct {
	Name     string
	Created  bool
	Switched bool
	Dirty    bool
	Path     string // worktree path; empty for ordinary branches
}

// Errors returned by Repo methods.
var (
	ErrNotRepo       = errors.New("not a git repository")
	ErrBranchExists  = errors.New("branch already exists")
	ErrDetached      = errors.New("HEAD is detached")
	ErrCurrentBranch = errors.New("cannot delete the current branch")
	ErrNotMerged     = errors.New("branch is not fully merged")
	ErrHookExists    = errors.New("foreign prepare-commit-msg hook exists")
	ErrWorktreeBusy  = errors.New("branch is checked out in another worktree")
)

// Repo is a git repository working directory.
type Repo struct{ Dir string }

// Brancher is the interface consumed by the TUI.
type Brancher interface {
	Create(name string, o Options) (Result, error)
	Switch(name string) (Result, error)
}

var _ Brancher = (*Repo)(nil)

// Open verifies that dir is inside a git repository.
func Open(dir string) (*Repo, error) {
	r := &Repo{Dir: dir}
	if _, code, err := r.git("rev-parse", "--git-dir"); err != nil {
		if code < 0 {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %s", ErrNotRepo, dir)
	}
	return r, nil
}

// Exists reports whether a local branch with that name exists.
func (r *Repo) Exists(name string) (bool, error) {
	if err := ValidName(name); err != nil {
		return false, err
	}
	_, code, err := r.git("show-ref", "--verify", "--quiet", "refs/heads/"+name)
	if err == nil {
		return true, nil
	}
	if code == 1 {
		return false, nil
	}
	return false, err
}

// Dirty reports whether the working tree has uncommitted changes.
func (r *Repo) Dirty() (bool, error) {
	out, _, err := r.git("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// Create makes a new branch; if it exists, returns ErrBranchExists.
func (r *Repo) Create(name string, o Options) (Result, error) {
	res := Result{Name: name}
	if err := ValidName(name); err != nil {
		return res, err
	}
	ok, err := r.Exists(name)
	if err != nil {
		return res, err
	}
	if res.Dirty, err = r.Dirty(); err != nil {
		return res, err
	}
	if ok {
		return res, ErrBranchExists
	}
	args := []string{"branch", name}
	if o.Base != "" {
		args = append(args, o.Base)
	}
	if _, _, err := r.git(args...); err != nil {
		return res, err
	}
	res.Created = true
	if o.Checkout {
		if _, _, err := r.git("checkout", name); err != nil {
			return res, err
		}
		res.Switched = true
	}
	return res, nil
}

// Switch checks out an existing branch.
func (r *Repo) Switch(name string) (Result, error) {
	res := Result{Name: name}
	ok, err := r.Exists(name)
	if err != nil {
		return res, err
	}
	if !ok {
		return res, fmt.Errorf("branch %q does not exist", name)
	}
	if res.Dirty, err = r.Dirty(); err != nil {
		return res, err
	}
	if _, _, err := r.git("checkout", name); err != nil {
		return res, err
	}
	res.Switched = true
	return res, nil
}

// Toplevel returns the root of the working tree.
func (r *Repo) Toplevel() (string, error) {
	return r.gitOut("rev-parse", "--show-toplevel")
}

// Current returns the branch checked out in this working tree.
func (r *Repo) Current() (string, error) {
	out, code, err := r.git("symbolic-ref", "--short", "-q", "HEAD")
	if err != nil {
		if code == 1 {
			return "", ErrDetached
		}
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Branches lists local branches sorted by name.
func (r *Repo) Branches() ([]string, error) {
	out, err := r.gitOut("for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, l := range strings.Split(out, "\n") {
		if l != "" {
			names = append(names, l)
		}
	}
	sort.Strings(names)
	return names, nil
}

// Delete removes a local branch (`git branch -d`, or -D with force).
func (r *Repo) Delete(name string, force bool) error {
	if err := ValidName(name); err != nil {
		return err
	}
	if cur, err := r.Current(); err == nil && cur == name {
		return ErrCurrentBranch
	}
	wts, err := r.worktrees()
	if err != nil {
		return err
	}
	if p, ok := wts[name]; ok {
		return fmt.Errorf("%w: %s", ErrWorktreeBusy, p)
	}
	flag := "-d"
	if force {
		flag = "-D"
	}
	if _, _, err := r.git("branch", flag, name); err != nil {
		if strings.Contains(err.Error(), "not fully merged") {
			return fmt.Errorf("%w: %s", ErrNotMerged, name)
		}
		return err
	}
	return nil
}
