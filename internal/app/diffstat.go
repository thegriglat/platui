package app

import (
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
)

// DiffStat is the size of the task branch changes relative to cfg.Branch.Base.
// The branch is determined as in BranchStatuses; no local branch or nil Git → ok=false.
func (a *App) DiffStat(p plane.Project, it plane.WorkItem) (gitbranch.Stat, bool, error) {
	if a.d.Git == nil {
		return gitbranch.Stat{}, false, nil
	}
	sts, err := a.BranchStatuses(p, []plane.WorkItem{it})
	if err != nil {
		return gitbranch.Stat{}, false, err
	}
	st, ok := sts[it.ID]
	if !ok || !st.Local {
		return gitbranch.Stat{}, false, nil
	}
	s, err := a.d.Git.DiffStat(a.cfg().Branch.Base, st.Name)
	if err != nil {
		return gitbranch.Stat{}, false, err
	}
	return s, true, nil
}
