// Command platui is a TUI and CLI for Plane; this file only wires dependencies (C0, C10).
package main

import (
	"context"
	"errors"
	"os"
	"os/signal"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/cache"
	"github.com/thegriglat/platui/internal/cli"
	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/httpclient"
	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/tui"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := cli.Run(ctx, os.Args[1:], cli.Env{
		Stdout: os.Stdout, Stderr: os.Stderr,
		Getenv: os.Getenv, Getwd: os.Getwd,
		Stdin: os.Stdin, IsTTY: stdinIsTTY,
		Build:  build,
		RunTUI: runTUI,
	})
	stop()
	os.Exit(code)
}

func build(cfg *config.Config, repoDir string) (*app.App, error) {
	hc, err := httpclient.New(httpclient.Options{Timeout: cfg.HTTPTimeout(), CAFile: cfg.HTTPCAFile(), Headers: cfg.HTTPHeaders()})
	if err != nil {
		return nil, err
	}
	d := app.Deps{
		API: plane.New(plane.Options{
			BaseURL: cfg.BaseURL, APIKey: cfg.Key(), Workspace: cfg.Workspace, WebURL: cfg.WebURL,
			HTTPClient: hc, MaxRetries: maxRetries(cfg.HTTPRetries()),
		}),
		Config: cfg,
	}
	repo, err := gitbranch.Open(repoDir)
	switch {
	case err == nil:
		d.Git = repo
	case !errors.Is(err, gitbranch.ErrNotRepo):
		return nil, err
	}
	if cfg.CacheEnabled() {
		dir, err := cfg.CacheDir()
		if err != nil {
			return nil, err
		}
		d.Store = cache.NewFileStore(dir)
	}
	return app.New(d), nil
}

func runTUI(a *app.App, cfg *config.Config, s cli.StartAt) error {
	if err := cfg.CheckCommandKeys(tui.BuiltinKeys()); err != nil {
		return err
	}
	return tui.Run(tui.Deps{App: a, Config: cfg, Start: tui.StartAt{Project: s.Project, Item: s.Item}})
}

func stdinIsTTY() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// maxRetries maps the config retries value to plane.Options.MaxRetries: 0 (no retries) becomes -1.
func maxRetries(n int) int {
	if n == 0 {
		return -1
	}
	return n
}
