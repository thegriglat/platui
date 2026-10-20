package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/gitbranch"
)

// openGit — открытие репозитория; подменяется в тестах.
var openGit = func(dir string) (gitbranch.Git, error) { return gitbranch.Open(dir) }

func branchCmd(s *session) *cobra.Command {
	var printOnly, worktree bool
	c := &cobra.Command{
		Use:   "branch <ref>",
		Short: "Create or switch to the branch of a work item",
		Args:  argsRange(1, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, p, it, err := resolve(s, args[0])
			if err != nil {
				return err
			}
			name, err := a.BranchName(p, it)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			if printOnly {
				fmt.Fprintln(w, name)
				return nil
			}
			dir, err := s.repoDir()
			if err != nil {
				return err
			}
			g, err := openGit(dir)
			if err != nil {
				return err
			}
			return makeBranch(w, g, s.cfg, name, worktree || s.cfg.Branch.Worktree)
		},
	}
	c.Flags().BoolVar(&printOnly, "print", false, "only print the branch name")
	c.Flags().BoolVar(&worktree, "worktree", false, "create a worktree")
	return c
}

func makeBranch(w io.Writer, g gitbranch.Git, cfg *config.Config, name string, wt bool) error {
	opts := gitbranch.Options{Base: cfg.Branch.Base, Checkout: cfg.CheckoutEnabled()}
	var res gitbranch.Result
	var err error
	if wt {
		top, terr := g.Toplevel()
		if terr != nil {
			return terr
		}
		dir := cfg.Branch.WorktreeDir
		if dir == "" {
			dir = config.DefaultWorktreeDir
		}
		res, err = g.AddWorktree(name, gitbranch.WorktreePath(dir, top, name), opts)
	} else {
		res, err = g.Create(name, opts)
		if errors.Is(err, gitbranch.ErrBranchExists) {
			if opts.Checkout {
				res, err = g.Switch(name)
			} else {
				res, err = gitbranch.Result{Name: name}, nil
			}
		}
	}
	if err != nil {
		return err
	}
	printBranch(w, res)
	if res.Dirty {
		fmt.Fprintln(w, "warning: working tree has uncommitted changes")
	}
	return nil
}

func printBranch(w io.Writer, r gitbranch.Result) {
	switch {
	case r.Path != "":
		fmt.Fprintf(w, "branch: %s (worktree %s)\n", r.Name, r.Path)
	case r.Created:
		fmt.Fprintf(w, "branch: %s (created)\n", r.Name)
	case r.Switched:
		fmt.Fprintf(w, "branch: %s (switched)\n", r.Name)
	default:
		fmt.Fprintf(w, "branch: %s\n", r.Name)
	}
}

func startCmd(s *session) *cobra.Command {
	var worktree bool
	c := &cobra.Command{
		Use:   "start <ref>",
		Short: "Start work: branch, commit template, state and assignee",
		Args:  argsRange(1, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, p, it, err := resolve(s, args[0])
			if err != nil {
				return err
			}
			var o app.StartOptions
			if cmd.Flags().Changed("worktree") {
				o.Worktree = &worktree
			}
			res, err := a.Start(s.ctx, p, it, o)
			w := cmd.OutOrStdout()
			if res.Branch.Name != "" {
				printBranch(w, res.Branch)
			}
			if res.State != "" {
				fmt.Fprintf(w, "state: %s\n", res.State)
			}
			if res.Assigned {
				fmt.Fprintln(w, "assigned: me")
			}
			printWarnings(w, res.Warnings)
			return err
		},
	}
	c.Flags().BoolVar(&worktree, "worktree", false, "create a worktree")
	return c
}

func reviewCmd(s *session) *cobra.Command {
	return &cobra.Command{
		Use:   "review [ref]",
		Short: "Ready for review: push, MR, state (default: item of the current branch)",
		Args:  argsRange(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var a *app.App
			var err error
			var res app.ReviewResult
			if len(args) == 1 {
				a, p, it, rerr := resolve(s, args[0])
				if rerr != nil {
					return rerr
				}
				res, err = a.Review(s.ctx, p, it)
			} else {
				if a, err = s.load(); err != nil {
					return err
				}
				p, it, cerr := a.CurrentItem(s.ctx)
				if cerr != nil {
					return cerr
				}
				res, err = a.Review(s.ctx, p, it)
			}
			w := cmd.OutOrStdout()
			if res.MRURL != "" {
				fmt.Fprintf(w, "MR: %s\n", res.MRURL)
			}
			if res.State != "" {
				fmt.Fprintf(w, "state: %s\n", res.State)
			}
			printWarnings(w, res.Warnings)
			return err
		},
	}
}

func printWarnings(w io.Writer, ws []string) {
	for _, x := range ws {
		fmt.Fprintf(w, "warning: %s\n", x)
	}
}
