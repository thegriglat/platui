package app

import (
	"errors"

	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
)

// Branch создает ветку задачи (или worktree) без изменений в Plane (клавиша b, C5 v4).
// name пусто — BranchName; worktree nil — cfg.Branch.Worktree; существующая ветка
// переключается только при switchExisting, иначе — Result{Name} и ErrBranchExists.
func (a *App) Branch(p plane.Project, it plane.WorkItem, name string, worktree *bool, switchExisting bool) (gitbranch.Result, error) {
	g, err := a.git()
	if err != nil {
		return gitbranch.Result{}, err
	}
	if name == "" {
		if name, err = a.BranchName(p, it); err != nil {
			return gitbranch.Result{}, err
		}
	}
	c := a.cfg()
	wt := c.Branch.Worktree
	if worktree != nil {
		wt = *worktree
	}
	opts := gitbranch.Options{Base: c.Branch.Base, Checkout: c.CheckoutEnabled()}
	if wt {
		top, err := g.Toplevel()
		if err != nil {
			return gitbranch.Result{Name: name}, err
		}
		dir := c.Branch.WorktreeDir
		if dir == "" {
			dir = config.DefaultWorktreeDir
		}
		return g.AddWorktree(name, gitbranch.WorktreePath(dir, top, name), opts)
	}
	res, err := g.Create(name, opts)
	if errors.Is(err, gitbranch.ErrBranchExists) {
		if !switchExisting {
			return gitbranch.Result{Name: name}, err
		}
		if opts.Checkout {
			return g.Switch(name)
		}
		return gitbranch.Result{Name: name}, nil
	}
	return res, err
}
