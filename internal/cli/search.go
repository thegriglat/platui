package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/search"
)

type hitOut struct {
	Key     string `json:"key"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Project string `json:"project"`
}

func searchCmd(s *session) *cobra.Command {
	var asJSON, server bool
	c := &cobra.Command{
		Use:   "search <query...>",
		Short: "Search work items by title",
		Long: "Search work items by title (fuzzy, local). A query like PROJ-42 or 42 shows that item.\n" +
			"With --server the Plane server searches the whole workspace; it also matches descriptions.",
		Args: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(strings.Join(args, " ")) == "" {
				return usagef("%s: missing query", cmd.CommandPath())
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			q := strings.Join(args, " ")
			a, err := s.load()
			if err != nil {
				return err
			}
			if id, seq, ok := search.ParseKey(q); ok {
				ref := strconv.Itoa(seq)
				if id != "" {
					ref = id + "-" + ref
				}
				return showRef(s, cmd.OutOrStdout(), ref, asJSON, false)
			}
			if server {
				return serverSearch(s, cmd, a, q, asJSON)
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
			byID := map[string]int{}
			for i, it := range items {
				byID[it.ID] = i
			}
			rows := []row{}
			for _, h := range search.New(app.SearchDocs(m, items)).Search(q) {
				rows = append(rows, buildRow(a, m, items[byID[h.ID]]))
			}
			return printRows(cmd, rows, asJSON)
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	c.Flags().BoolVar(&server, "server", false, "search the whole workspace on the server (also matches descriptions)")
	return c
}

func serverSearch(s *session, cmd *cobra.Command, a *app.App, q string, asJSON bool) error {
	hits, err := a.ServerSearch(s.ctx, q, "")
	if err != nil {
		return err
	}
	out := []hitOut{}
	w := 0
	for _, h := range hits {
		w = max(w, len(h.ProjectIdentifier+"-"+strconv.Itoa(h.Sequence)))
	}
	for _, h := range hits {
		key := h.ProjectIdentifier + "-" + strconv.Itoa(h.Sequence)
		out = append(out, hitOut{key, h.ID, h.Name, h.ProjectIdentifier})
	}
	if asJSON {
		return writeJSON(cmd.OutOrStdout(), out)
	}
	for _, h := range out {
		fmt.Fprintf(cmd.OutOrStdout(), "%-*s  %s\n", w, h.Key, h.Name)
	}
	return nil
}
