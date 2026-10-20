package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/plane"
)

// projectErr makes a project error readable.
func projectErr(s *session, err error) error {
	if s.projectIdent() == "" {
		return errors.New("no project: use --project or default_project in the config")
	}
	return err
}

type commentOut struct {
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
	Text      string `json:"text"`
}

type showOut struct {
	row
	Description string       `json:"description"`
	Comments    []commentOut `json:"comments,omitempty"`
}

func showCmd(s *session) *cobra.Command {
	var asJSON, comments bool
	c := &cobra.Command{
		Use:   "show <ref>",
		Short: "Show a work item (PROJ-42 or 42)",
		Args:  argsRange(1, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showRef(s, cmd.OutOrStdout(), args[0], asJSON, comments)
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	c.Flags().BoolVar(&comments, "comments", false, "include comments")
	return c
}

// resolve finds an issue by reference, PROJ-42 or 42.
func resolve(s *session, ref string) (*app.App, plane.Project, plane.WorkItem, error) {
	a, err := s.load()
	if err != nil {
		return nil, plane.Project{}, plane.WorkItem{}, err
	}
	ident, seq, ok := app.ParseRef(ref, s.projectIdent())
	if !ok {
		return nil, plane.Project{}, plane.WorkItem{}, usagef("invalid work item reference %q (use PROJ-42 or 42 with a default project)", ref)
	}
	p, it, err := a.Find(s.ctx, ident, seq)
	return a, p, it, err
}

func showRef(s *session, w io.Writer, ref string, asJSON, withComments bool) error {
	a, p, it, err := resolve(s, ref)
	if err != nil {
		return err
	}
	m, err := a.Meta(s.ctx, p.ID)
	if err != nil {
		return err
	}
	out := showOut{row: buildRow(a, m, it), Description: strings.TrimSpace(plane.HTMLToText(it.DescHTML))}
	if withComments {
		cs, err := a.Comments(s.ctx, p.ID, it.ID)
		if err != nil {
			return err
		}
		out.Comments = []commentOut{}
		for _, c := range cs {
			out.Comments = append(out.Comments, commentOut{c.Author, c.CreatedAt.UTC().Format(time.RFC3339), strings.TrimSpace(plane.HTMLToText(c.HTML))})
		}
	}
	if asJSON {
		return writeJSON(w, out)
	}
	fmt.Fprintf(w, "%s  %s\n", out.Key, out.Name)
	fmt.Fprintf(w, "State:     %s\n", out.State)
	fmt.Fprintf(w, "Priority:  %s\n", prio(out.Priority))
	fmt.Fprintf(w, "Assignees: %s\n", strings.Join(out.Assignees, ", "))
	fmt.Fprintf(w, "Labels:    %s\n", strings.Join(out.Labels, ", "))
	fmt.Fprintf(w, "URL:       %s\n", out.URL)
	if out.Description != "" {
		fmt.Fprintf(w, "\n%s\n", out.Description)
	}
	for _, c := range out.Comments {
		fmt.Fprintf(w, "\n--- %s, %s\n%s\n", c.Author, c.CreatedAt, c.Text)
	}
	return nil
}
