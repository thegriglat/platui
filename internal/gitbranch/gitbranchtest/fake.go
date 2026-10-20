// Package gitbranchtest provides an in-memory gitbranch.Git for tests.
package gitbranchtest

import (
	"fmt"
	"sort"
	"sync"

	"github.com/thegriglat/platui/internal/gitbranch"
)

// Fake is an in-memory gitbranch.Git. Fields are exported so tests can set
// state and inspect calls; guard concurrent access with Mu.
type Fake struct {
	Mu            sync.Mutex
	BranchesV     map[string]gitbranch.BranchStatus
	CurrentV      string
	TopV          string
	RemoteURLV    string
	DirtyV        bool
	Err           map[string]error // by method name; returned instead of acting
	Calls         []string
	Pushed        []string // "remote/name"
	CommitMsgs    map[string]string
	HookInstalled bool
	Stats         map[string]gitbranch.Stat // by branch
}

var _ gitbranch.Git = (*Fake)(nil)

// NewFake returns an empty Fake with initialised maps.
func NewFake() *Fake {
	return &Fake{
		BranchesV:  map[string]gitbranch.BranchStatus{},
		Err:        map[string]error{},
		CommitMsgs: map[string]string{},
	}
}

// begin records the call and returns the injected error. Caller holds Mu.
func (f *Fake) begin(method string, args ...any) error {
	f.Calls = append(f.Calls, fmt.Sprint(append([]any{method}, args...)...))
	return f.Err[method]
}

func (f *Fake) lock(method string, args ...any) error {
	f.Mu.Lock()
	return f.begin(method, args...)
}

// Create adds a local branch; an existing one gives ErrBranchExists.
func (f *Fake) Create(name string, o gitbranch.Options) (gitbranch.Result, error) {
	defer f.Mu.Unlock()
	res := gitbranch.Result{Name: name, Dirty: f.DirtyV}
	if err := f.lock("Create", name); err != nil {
		return res, err
	}
	if _, ok := f.BranchesV[name]; ok {
		return res, gitbranch.ErrBranchExists
	}
	f.BranchesV[name] = gitbranch.BranchStatus{Name: name, Local: true}
	res.Created = true
	if o.Checkout {
		f.CurrentV = name
		res.Switched = true
	}
	return res, nil
}

// Switch makes an existing local branch current.
func (f *Fake) Switch(name string) (gitbranch.Result, error) {
	defer f.Mu.Unlock()
	res := gitbranch.Result{Name: name, Dirty: f.DirtyV}
	if err := f.lock("Switch", name); err != nil {
		return res, err
	}
	if st, ok := f.BranchesV[name]; !ok || !st.Local {
		return res, fmt.Errorf("branch %q does not exist", name)
	}
	f.CurrentV = name
	res.Switched = true
	return res, nil
}

// Exists reports whether a local branch exists.
func (f *Fake) Exists(name string) (bool, error) {
	defer f.Mu.Unlock()
	if err := f.lock("Exists", name); err != nil {
		return false, err
	}
	st, ok := f.BranchesV[name]
	return ok && st.Local, nil
}

// Dirty returns DirtyV.
func (f *Fake) Dirty() (bool, error) {
	defer f.Mu.Unlock()
	err := f.lock("Dirty")
	return f.DirtyV, err
}

// Toplevel returns TopV.
func (f *Fake) Toplevel() (string, error) {
	defer f.Mu.Unlock()
	err := f.lock("Toplevel")
	return f.TopV, err
}

// Current returns CurrentV, or ErrDetached when empty.
func (f *Fake) Current() (string, error) {
	defer f.Mu.Unlock()
	if err := f.lock("Current"); err != nil {
		return "", err
	}
	if f.CurrentV == "" {
		return "", gitbranch.ErrDetached
	}
	return f.CurrentV, nil
}

// Branches lists local branches sorted by name.
func (f *Fake) Branches() ([]string, error) {
	defer f.Mu.Unlock()
	if err := f.lock("Branches"); err != nil {
		return nil, err
	}
	var out []string
	for n, st := range f.BranchesV {
		if st.Local {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out, nil
}

// Statuses returns stored statuses for names (empty: all); Current is set
// from CurrentV.
func (f *Fake) Statuses(names []string, remote, base string) (map[string]gitbranch.BranchStatus, error) {
	defer f.Mu.Unlock()
	if err := f.lock("Statuses", remote, base); err != nil {
		return nil, err
	}
	res := map[string]gitbranch.BranchStatus{}
	add := func(n string) {
		st, ok := f.BranchesV[n]
		if !ok {
			st = gitbranch.BranchStatus{}
		}
		st.Name = n
		st.Current = n == f.CurrentV
		res[n] = st
	}
	if len(names) == 0 {
		for n := range f.BranchesV {
			add(n)
		}
		return res, nil
	}
	for _, n := range names {
		add(n)
	}
	return res, nil
}

// AddWorktree records a branch checked out at path; one already in a
// worktree gives ErrWorktreeBusy.
func (f *Fake) AddWorktree(name, path string, o gitbranch.Options) (gitbranch.Result, error) {
	defer f.Mu.Unlock()
	res := gitbranch.Result{Name: name}
	if err := f.lock("AddWorktree", name, path); err != nil {
		return res, err
	}
	st, ok := f.BranchesV[name]
	if st.Worktree != "" {
		res.Path = st.Worktree
		return res, gitbranch.ErrWorktreeBusy
	}
	if name == f.CurrentV {
		res.Path = f.TopV
		return res, gitbranch.ErrWorktreeBusy
	}
	res.Created = !ok
	st.Name, st.Local, st.Worktree = name, true, path
	f.BranchesV[name] = st
	res.Path = path
	return res, nil
}

// Delete removes a local branch.
func (f *Fake) Delete(name string, force bool) error {
	defer f.Mu.Unlock()
	if err := f.lock("Delete", name, force); err != nil {
		return err
	}
	st, ok := f.BranchesV[name]
	if !ok {
		return fmt.Errorf("branch %q does not exist", name)
	}
	if name == f.CurrentV {
		return gitbranch.ErrCurrentBranch
	}
	if st.Worktree != "" {
		return gitbranch.ErrWorktreeBusy
	}
	delete(f.BranchesV, name)
	return nil
}

// WriteCommitMsg stores the message in CommitMsgs.
func (f *Fake) WriteCommitMsg(branch, msg string) error {
	defer f.Mu.Unlock()
	if err := f.lock("WriteCommitMsg", branch); err != nil {
		return err
	}
	f.CommitMsgs[branch] = msg
	return nil
}

// ReadCommitMsg returns the message from CommitMsgs.
func (f *Fake) ReadCommitMsg(branch string) (string, bool, error) {
	defer f.Mu.Unlock()
	if err := f.lock("ReadCommitMsg", branch); err != nil {
		return "", false, err
	}
	msg, ok := f.CommitMsgs[branch]
	return msg, ok, nil
}

// InstallHook sets HookInstalled; installed is false if it already was.
func (f *Fake) InstallHook() (string, bool, error) {
	defer f.Mu.Unlock()
	if err := f.lock("InstallHook"); err != nil {
		return "", false, err
	}
	was := f.HookInstalled
	f.HookInstalled = true
	return "hooks/prepare-commit-msg", !was, nil
}

// Push records "remote/name" and marks the branch as remote.
func (f *Fake) Push(remote, name string) error {
	defer f.Mu.Unlock()
	if err := f.lock("Push", remote, name); err != nil {
		return err
	}
	f.Pushed = append(f.Pushed, remote+"/"+name)
	st := f.BranchesV[name]
	st.Name, st.Remote = name, true
	f.BranchesV[name] = st
	return nil
}

// RemoteURL returns RemoteURLV.
func (f *Fake) RemoteURL(remote string) (string, error) {
	defer f.Mu.Unlock()
	err := f.lock("RemoteURL", remote)
	return f.RemoteURLV, err
}

// DiffStat returns Stats[branch]; a branch unknown to both maps is an error.
func (f *Fake) DiffStat(base, branch string) (gitbranch.Stat, error) {
	defer f.Mu.Unlock()
	if err := f.lock("DiffStat", base, branch); err != nil {
		return gitbranch.Stat{}, err
	}
	if st, ok := f.Stats[branch]; ok {
		return st, nil
	}
	if _, ok := f.BranchesV[branch]; !ok {
		return gitbranch.Stat{}, fmt.Errorf("branch %q does not exist", branch)
	}
	return gitbranch.Stat{}, nil
}
