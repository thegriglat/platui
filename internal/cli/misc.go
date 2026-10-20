package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func promptCmd(s *session) *cobra.Command {
	var format string
	c := &cobra.Command{
		Use:   "prompt",
		Short: "Print the current item for a shell prompt or tmux (cache only, no network)",
		Long:  "Placeholders: {key}, {state}, {title}. Prints nothing and exits 0 when there is no data or on any error.",
		Args:  argsRange(0, 0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if a, err := s.load(); err == nil {
				key, state, _ := strings.Cut(a.Prompt(), " ")
				if key != "" {
					r := strings.NewReplacer("{key}", key, "{state}", state, "{title}", "")
					if out := strings.TrimSpace(r.Replace(format)); out != "" {
						fmt.Fprintln(cmd.OutOrStdout(), out)
					}
				}
			}
			return nil
		},
	}
	c.Flags().StringVar(&format, "format", "{key} {state}", "output template")
	return c
}

func hookCmd(s *session) *cobra.Command {
	h := &cobra.Command{Use: "hook", Short: "Git hooks", Args: argsRange(0, 0),
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() }}
	h.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Install the prepare-commit-msg hook",
		Args:  argsRange(0, 0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := s.repoDir()
			if err != nil {
				return err
			}
			g, err := openGit(dir)
			if err != nil {
				return err
			}
			path, installed, err := g.InstallHook()
			if err != nil {
				return err
			}
			if installed {
				fmt.Fprintf(cmd.OutOrStdout(), "hook installed: %s\n", path)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "hook already installed: %s\n", path)
			}
			return nil
		},
	})
	return h
}

func versionCmd(*session) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  argsRange(0, 0),
		Run:   func(cmd *cobra.Command, _ []string) { fmt.Fprintf(cmd.OutOrStdout(), "platui %s\n", versionString()) },
	}
}
