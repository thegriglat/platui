// Package cli — режим без интерфейса и разбор командной строки (контракт C10 v1).
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/plane"
)

// Version задаётся -ldflags "-X github.com/thegriglat/platui/internal/cli.Version=…".
var Version = "dev"

// StartAt — куда открыть TUI.
type StartAt struct{ Project, Item string }

// Env — внешние зависимости CLI.
type Env struct {
	Stdout, Stderr io.Writer
	Getenv         func(string) string
	Getwd          func() (string, error)
	Stdin          io.Reader   // ответы на вопросы подтверждения
	IsTTY          func() bool // stdin — терминал
	Yes            bool        // глобальный флаг --yes (заполняется разбором)
	Build          func(cfg *config.Config, repoDir string) (*app.App, error)
	RunTUI         func(a *app.App, cfg *config.Config, start StartAt) error
}

// usageError — ошибка использования (код 2).
type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }

func usagef(format string, a ...any) error { return usageError{fmt.Sprintf(format, a...)} }

// session — состояние одного запуска.
type session struct {
	ctx context.Context
	env Env

	configPath, repo, project string
	item                      string

	cfg *config.Config
	app *app.App
}

func (s *session) repoDir() (string, error) {
	if s.repo != "" {
		return s.repo, nil
	}
	return s.env.Getwd()
}

func (s *session) loadConfig() (*config.Config, error) {
	if s.cfg != nil {
		return s.cfg, nil
	}
	path := s.configPath
	if path == "" {
		p, err := config.DefaultPath()
		if err != nil {
			return nil, err
		}
		path = p
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	s.cfg = cfg
	return cfg, nil
}

func (s *session) load() (*app.App, error) {
	if s.app != nil {
		return s.app, nil
	}
	cfg, err := s.loadConfig()
	if err != nil {
		return nil, err
	}
	dir, err := s.repoDir()
	if err != nil {
		return nil, err
	}
	a, err := s.env.Build(cfg, dir)
	if err != nil {
		return nil, err
	}
	s.app = a
	return a, nil
}

// projectIdent — ключ проекта: --project, иначе default_project.
func (s *session) projectIdent() string {
	if s.project != "" {
		return s.project
	}
	if s.cfg != nil {
		return s.cfg.DefaultProject
	}
	return ""
}

func argsRange(lo, hi int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < lo {
			return usagef("%s: missing argument", cmd.CommandPath())
		}
		if len(args) > hi {
			return usagef("%s: too many arguments", cmd.CommandPath())
		}
		return nil
	}
}

func newRoot(s *session) *cobra.Command {
	root := &cobra.Command{
		Use:           "platui",
		Short:         "TUI and CLI for Plane",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return usagef("unknown command %q", args[0])
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := s.load()
			if err != nil {
				return err
			}
			if s.env.RunTUI == nil {
				return errors.New("TUI is not available")
			}
			return s.env.RunTUI(a, s.cfg, StartAt{Project: s.project, Item: s.item})
		},
	}
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageError{err.Error()} })
	f := root.PersistentFlags()
	f.StringVar(&s.configPath, "config", "", "path to config file")
	f.StringVar(&s.repo, "repo", "", "repository directory (default: current)")
	f.BoolVar(&s.env.Yes, "yes", false, "confirm destructive actions without asking")
	f.StringVar(&s.project, "project", "", "project identifier (default: default_project)")
	root.Flags().StringVar(&s.item, "item", "", "open the TUI on this work item (PROJ-42)")
	root.AddCommand(lsCmd(s), showCmd(s), searchCmd(s), branchCmd(s), startCmd(s), reviewCmd(s),
		promptCmd(s), hookCmd(s), versionCmd(s), mrCmd(s), todoCmd(s))
	return root
}

// Run выполняет platui с аргументами args и возвращает код выхода.
func Run(ctx context.Context, args []string, env Env) int {
	if env.Stdout == nil {
		env.Stdout = io.Discard
	}
	if env.Stderr == nil {
		env.Stderr = io.Discard
	}
	if env.Getenv == nil {
		env.Getenv = os.Getenv
	}
	if env.Getwd == nil {
		env.Getwd = os.Getwd
	}
	s := &session{ctx: ctx, env: env}
	root := newRoot(s)
	root.SetArgs(args)
	root.SetOut(env.Stdout)
	root.SetErr(env.Stderr)
	root.SetIn(nil)
	c, err := root.ExecuteC()
	if err == nil {
		return 0
	}
	var ue usageError
	if errors.As(err, &ue) {
		fmt.Fprintf(env.Stderr, "platui: %s\n\n%s", ue.msg, c.UsageString())
		return 2
	}
	switch {
	case errors.Is(err, plane.ErrUnauthorized):
		fmt.Fprintf(env.Stderr, "platui: ключ API отклонён — проверьте api_key/api_key_env (%v)\n", err)
		return 3
	case app.IsNetworkError(err):
		fmt.Fprintf(env.Stderr, "platui: Plane недоступен: %v\n", err)
		return 4
	}
	fmt.Fprintf(env.Stderr, "platui: %v\n", err)
	return 1
}
