package gitbranch

import (
	"path/filepath"
	"strconv"
	"strings"
)

// BranchStatus describes where a branch lives and how far it is from base.
type BranchStatus struct {
	Name          string
	Local         bool   // refs/heads/<name>
	Remote        bool   // refs/remotes/<remote>/<name>
	Current       bool   // branch of HEAD in this working tree
	Worktree      string // other worktree where the branch is checked out
	Ahead, Behind int    // commits relative to base; 0, 0 if unknown
}

// Statuses returns statuses for names (empty: all local branches). base
// empty means refs/remotes/<remote>/HEAD, then main, then master.
func (r *Repo) Statuses(names []string, remote, base string) (map[string]BranchStatus, error) {
	if len(names) == 0 {
		var err error
		if names, err = r.Branches(); err != nil {
			return nil, err
		}
	}
	local, remotes, err := r.refSets(remote)
	if err != nil {
		return nil, err
	}
	wts, err := r.worktrees()
	if err != nil {
		return nil, err
	}
	top, _ := r.Toplevel()
	cur, _ := r.Current()
	baseRef := r.resolveBase(remote, base)
	res := make(map[string]BranchStatus, len(names))
	for _, n := range names {
		st := BranchStatus{Name: n, Local: local[n], Remote: remotes[n], Current: n == cur}
		if p, ok := wts[n]; ok && !samePath(p, top) {
			st.Worktree = p
		}
		if st.Local && baseRef != "" {
			st.Ahead, st.Behind = r.aheadBehind(baseRef, n)
		}
		res[n] = st
	}
	return res, nil
}

func (r *Repo) refSets(remote string) (local, remotes map[string]bool, err error) {
	prefix := "refs/remotes/" + remote + "/"
	out, err := r.gitOut("for-each-ref", "--format=%(refname)", "refs/heads", prefix)
	if err != nil {
		return nil, nil, err
	}
	local, remotes = map[string]bool{}, map[string]bool{}
	for _, l := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(l, "refs/heads/"):
			local[strings.TrimPrefix(l, "refs/heads/")] = true
		case strings.HasPrefix(l, prefix) && l != prefix+"HEAD":
			remotes[strings.TrimPrefix(l, prefix)] = true
		}
	}
	return local, remotes, nil
}

func (r *Repo) resolveBase(remote, base string) string {
	var cands []string
	if base != "" {
		cands = []string{base}
	} else {
		if out, err := r.gitOut("symbolic-ref", "-q", "refs/remotes/"+remote+"/HEAD"); err == nil && out != "" {
			cands = append(cands, out)
		}
		cands = append(cands, "refs/heads/main", "refs/heads/master")
	}
	for _, c := range cands {
		if _, _, err := r.git("rev-parse", "--verify", "--quiet", c+"^{commit}"); err == nil {
			return c
		}
	}
	return ""
}

func (r *Repo) aheadBehind(base, name string) (ahead, behind int) {
	out, err := r.gitOut("rev-list", "--left-right", "--count", base+"...refs/heads/"+name)
	if err != nil {
		return 0, 0
	}
	f := strings.Fields(out)
	if len(f) != 2 {
		return 0, 0
	}
	behind, _ = strconv.Atoi(f[0])
	ahead, _ = strconv.Atoi(f[1])
	return ahead, behind
}

func samePath(a, b string) bool {
	if ea, err := filepath.EvalSymlinks(a); err == nil {
		a = ea
	}
	if eb, err := filepath.EvalSymlinks(b); err == nil {
		b = eb
	}
	return filepath.Clean(a) == filepath.Clean(b)
}
