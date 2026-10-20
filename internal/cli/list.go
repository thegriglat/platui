package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/plane"
)

func lsCmd(s *session) *cobra.Command {
	var mine, all, asJSON bool
	var state string
	c := &cobra.Command{
		Use:   "ls",
		Short: "List work items of the project",
		Args:  argsRange(0, 0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := s.load()
			if err != nil {
				return err
			}
			p, err := a.Project(s.ctx, s.projectIdent())
			if err != nil {
				return projectErr(s, err)
			}
			m, err := a.Meta(s.ctx, p.ID)
			if err != nil {
				return err
			}
			items, _, err := a.RefreshItems(s.ctx, p.ID)
			if err != nil {
				return err
			}
			if mine {
				me, err := a.Me(s.ctx)
				if err != nil {
					return err
				}
				items = filterItems(items, func(it plane.WorkItem) bool { return contains(it.AssigneeIDs, me.ID) })
			}
			items = orderByStates(s, items, m, state, all)
			rows := make([]row, 0, len(items))
			for _, it := range items {
				rows = append(rows, buildRow(a, m, it))
			}
			return printRows(cmd, rows, asJSON)
		},
	}
	f := c.Flags()
	f.BoolVar(&mine, "mine", false, "only items assigned to me")
	f.StringVar(&state, "state", "", "only this state (by name)")
	f.BoolVar(&all, "all", false, "include hidden state groups")
	f.BoolVar(&asJSON, "json", false, "JSON output")
	return c
}

func printRows(cmd *cobra.Command, rows []row, asJSON bool) error {
	if asJSON {
		return writeJSON(cmd.OutOrStdout(), rows)
	}
	writeRows(cmd.OutOrStdout(), rows)
	return nil
}

// orderByStates keeps visible states and sorts by StateTabs, then by number.
func orderByStates(s *session, items []plane.WorkItem, m app.Meta, only string, all bool) []plane.WorkItem {
	cfg := *s.cfg
	if all || only != "" {
		cfg.Tabs.States = nil
		cfg.Tabs.HideGroups = []string{}
	}
	var out []plane.WorkItem
	used := map[string]bool{}
	for _, st := range app.StateTabs(&cfg, m.States) {
		if only != "" && !strings.EqualFold(st.Name, strings.TrimSpace(only)) {
			continue
		}
		used[st.ID] = true
		part := filterItems(items, func(it plane.WorkItem) bool { return it.StateID == st.ID })
		sortBySeq(part)
		out = append(out, part...)
	}
	if all && only == "" {
		rest := filterItems(items, func(it plane.WorkItem) bool { return !used[it.StateID] })
		sortBySeq(rest)
		out = append(out, rest...)
	}
	return out
}

func filterItems(items []plane.WorkItem, keep func(plane.WorkItem) bool) []plane.WorkItem {
	var out []plane.WorkItem
	for _, it := range items {
		if keep(it) {
			out = append(out, it)
		}
	}
	return out
}

func contains(l []string, v string) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}
