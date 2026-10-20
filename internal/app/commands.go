package app

import (
	"context"
	"strings"

	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/runner"
)

// CommandVars — значения плейсхолдеров своих команд.
type CommandVars struct{ ID, Key, Branch, URL, Title, MRURL string }

// CommandVars собирает плейсхолдеры: ID — идентификатор задачи в Plane,
// Key — PROJ-N, MRURL — первый MR из кэша или "".
func (a *App) CommandVars(p plane.Project, it plane.WorkItem) CommandVars {
	br, _ := a.BranchName(p, it)
	v := CommandVars{ID: it.ID, Key: p.Identifier + "-" + itoa(it.Sequence), Branch: br,
		URL: a.d.API.WebURL(p, it), Title: it.Name}
	if mrs, ok := a.CachedMRs(it.ID); ok && len(mrs) > 0 {
		v.MRURL = mrs[0].URL
	}
	return v
}

func shQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// RenderCommand подставляет плейсхолдеры; значения экранируются для sh
// (одинарные кавычки), подставленный текст не интерпретируется повторно.
func RenderCommand(tmpl string, v CommandVars) string {
	return strings.NewReplacer(
		"{id}", shQuote(v.ID), "{key}", shQuote(v.Key), "{branch}", shQuote(v.Branch),
		"{url}", shQuote(v.URL), "{title}", shQuote(v.Title), "{mr_url}", shQuote(v.MRURL),
	).Replace(tmpl)
}

// RunCommand запускает команду в корне репозитория (Git nil — в текущем каталоге).
func (a *App) RunCommand(ctx context.Context, c config.Command, v CommandVars) (runner.Result, error) {
	dir := "."
	if a.d.Git != nil {
		if top, err := a.d.Git.Toplevel(); err == nil && top != "" {
			dir = top
		}
	}
	sh := a.d.Shell
	if sh == nil {
		sh = runner.Default()
	}
	return sh(ctx, dir, RenderCommand(c.Command, v), nil)
}
