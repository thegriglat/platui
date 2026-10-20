package app

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
)

// StartOptions — параметры «старта работы».
type StartOptions struct {
	Branch   string // пусто — BranchName
	Worktree *bool  // nil — cfg.Branch.Worktree
}

// StartResult — итог «старта работы».
type StartResult struct {
	Branch   gitbranch.Result
	State    string // имя нового состояния; "" — не менялось
	Assigned bool
	Warnings []string
}

// Start создаёт ветку (или worktree), пишет шаблон коммита и переводит задачу в работу.
func (a *App) Start(ctx context.Context, p plane.Project, it plane.WorkItem, o StartOptions) (StartResult, error) {
	var res StartResult
	g, err := a.git()
	if err != nil {
		return res, err
	}
	name := o.Branch
	if name == "" {
		if name, err = a.BranchName(p, it); err != nil {
			return res, err
		}
	}
	c := a.cfg()
	wt := c.Branch.Worktree
	if o.Worktree != nil {
		wt = *o.Worktree
	}
	opts := gitbranch.Options{Base: c.Branch.Base, Checkout: c.CheckoutEnabled()}
	if wt {
		top, err := g.Toplevel()
		if err != nil {
			return res, err
		}
		dir := c.Branch.WorktreeDir
		if dir == "" {
			dir = config.DefaultWorktreeDir
		}
		res.Branch, err = g.AddWorktree(name, gitbranch.WorktreePath(dir, top, name), opts)
	} else {
		res.Branch, err = g.Create(name, opts)
		if errors.Is(err, gitbranch.ErrBranchExists) {
			if opts.Checkout {
				res.Branch, err = g.Switch(name)
			} else {
				res.Branch, err = gitbranch.Result{Name: name}, nil
			}
		}
	}
	if err != nil {
		return res, err
	}
	if res.Branch.Dirty {
		res.Warnings = append(res.Warnings, "working tree has uncommitted changes")
	}
	a.prepareCommit(g, name, p, it, &res)
	return res, a.startPlane(ctx, p, it, &res)
}

func (a *App) prepareCommit(g gitbranch.Git, name string, p plane.Project, it plane.WorkItem, res *StartResult) {
	msg, err := gitbranch.CommitMessage(a.commitTemplate(), gitItem(p, it))
	if err == nil {
		var old string
		var ok bool
		if old, ok, err = g.ReadCommitMsg(name); err == nil && ok && old != msg {
			res.Warnings = append(res.Warnings, "сообщение коммита для "+name+" уже есть — не перезаписано")
		} else if err == nil {
			err = g.WriteCommitMsg(name, msg)
		}
	}
	if err != nil {
		res.Warnings = append(res.Warnings, "commit message: "+err.Error())
		return
	}
	if _, _, err := g.InstallHook(); err != nil {
		res.Warnings = append(res.Warnings, "prepare-commit-msg hook: "+err.Error())
	}
}

// findState: по имени без учёта регистра.
func findState(states []plane.State, name string) (plane.State, bool) {
	i := slices.IndexFunc(states, func(s plane.State) bool { return strings.EqualFold(s.Name, strings.TrimSpace(name)) })
	if i < 0 || strings.TrimSpace(name) == "" {
		return plane.State{}, false
	}
	return states[i], true
}

func (a *App) startPlane(ctx context.Context, p plane.Project, it plane.WorkItem, res *StartResult) error {
	m, err := a.Meta(ctx, p.ID)
	if err != nil {
		return err
	}
	me, err := a.Me(ctx)
	if err != nil {
		return err
	}
	var patch plane.WorkItemPatch
	st, ok := findState(m.States, a.cfg().Workflow.StartState)
	if !ok {
		i := slices.IndexFunc(m.States, func(s plane.State) bool { return strings.EqualFold(s.Group, "started") })
		if i >= 0 {
			st, ok = m.States[i], true
		}
	}
	if ok && st.ID != it.StateID {
		patch.StateID = &st.ID
	}
	if !slices.Contains(it.AssigneeIDs, me.ID) {
		as := append(slices.Clone(it.AssigneeIDs), me.ID)
		patch.AssigneeIDs = &as
	}
	if patch.StateID == nil && patch.AssigneeIDs == nil {
		return nil
	}
	if _, err := a.Update(ctx, p.ID, it.ID, patch); err != nil {
		return err
	}
	if patch.StateID != nil {
		res.State = st.Name
	}
	res.Assigned = patch.AssigneeIDs != nil
	return nil
}
