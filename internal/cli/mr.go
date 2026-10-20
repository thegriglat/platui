package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/cli/browser"
	"github.com/spf13/cobra"

	"github.com/thegriglat/platui/internal/forge"
)

// openURL открывает адрес в браузере; подменяется в тестах.
var openURL = func(u string) error {
	browser.Stdout, browser.Stderr = io.Discard, io.Discard
	return browser.OpenURL(u)
}

type mrOut struct {
	IID   int    `json:"iid"`
	State string `json:"state"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

func mrCmd(s *session) *cobra.Command {
	var open, asJSON bool
	c := &cobra.Command{
		Use:   "mr [ref]",
		Short: "Find merge requests of a work item (default: item of the current branch)",
		Args:  argsRange(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMR(s, cmd.OutOrStdout(), args, open, asJSON)
		},
	}
	c.Flags().BoolVar(&open, "open", false, "open the MR in the browser (if exactly one)")
	c.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	return c
}

func runMR(s *session, w io.Writer, args []string, open, asJSON bool) error {
	var mrs []forge.MR
	if len(args) == 1 {
		a, p, it, err := resolve(s, args[0])
		if err != nil {
			return err
		}
		if mrs, err = a.MRs(s.ctx, p, it, true); err != nil {
			return err
		}
	} else {
		a, err := s.load()
		if err != nil {
			return err
		}
		p, it, err := a.CurrentItem(s.ctx)
		if err != nil {
			return err
		}
		if mrs, err = a.MRs(s.ctx, p, it, true); err != nil {
			return err
		}
	}
	if asJSON {
		out := []mrOut{}
		for _, m := range mrs {
			out = append(out, mrOut{m.IID, m.State, m.Title, m.URL})
		}
		if err := writeJSON(w, out); err != nil {
			return err
		}
	} else {
		for _, m := range mrs {
			fmt.Fprintf(w, "!%d  %s  %s  %s\n", m.IID, m.State, m.Title, m.URL)
		}
	}
	if len(mrs) == 0 {
		return errors.New("MR не найдены")
	}
	if open && len(mrs) == 1 {
		return openURL(mrs[0].URL)
	}
	return nil
}
