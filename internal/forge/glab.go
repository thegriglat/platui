package forge

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var urlRe = regexp.MustCompile(`https?://\S+`)

// Options — настройки GitLab-инструмента.
type Options struct {
	Glab string // бинарь; "" → "glab"
	Host string // "" — не задавать; иначе env GITLAB_HOST=<Host>
	Repo string // "" — из remote каталога; иначе -R <Repo>
	Run  Runner // nil → ExecRunner()
}

// GitLab реализует Tool через glab.
type GitLab struct{ o Options }

var _ Tool = (*GitLab)(nil)

// New создаёт инструмент GitLab.
func New(o Options) *GitLab {
	if o.Glab == "" {
		o.Glab = "glab"
	}
	if o.Run == nil {
		o.Run = ExecRunner()
	}
	return &GitLab{o}
}

func (g *GitLab) run(ctx context.Context, dir string, args ...string) (string, error) {
	var env []string
	if g.o.Host != "" {
		env = []string{"GITLAB_HOST=" + g.o.Host}
	}
	if g.o.Repo != "" {
		args = append(args, "-R", g.o.Repo)
	}
	out, err := g.o.Run(ctx, dir, env, g.o.Glab, args...)
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, ErrNotFound) {
		return out, fmt.Errorf("glab: %w", ErrNotFound)
	}
	return out, err
}

func wrap(out string, err error) error {
	if errors.Is(err, ErrNotFound) {
		return err
	}
	return fmt.Errorf("glab: %w\n%s", err, strings.TrimSpace(out))
}

func exists(out string, err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()+out), "already exists")
}

// CreateMR создаёт MR; если он уже есть — возвращает URL существующего.
func (g *GitLab) CreateMR(ctx context.Context, r MRRequest) (string, error) {
	args := []string{"mr", "create", "--source-branch", r.Head, "--title", r.Title, "--description", r.Body}
	if r.Target != "" {
		args = append(args, "--target-branch", r.Target)
	}
	if r.Draft {
		args = append(args, "--draft")
	}
	args = append(args, "--yes")
	out, err := g.run(ctx, r.Dir, args...)
	if exists(out, err) {
		mrs, lerr := g.list(ctx, r.Dir, "-s", r.Head, "-A")
		if lerr != nil {
			return "", lerr
		}
		if len(mrs) == 0 || mrs[0].URL == "" {
			return "", fmt.Errorf("glab: existing MR not found for %s", r.Head)
		}
		return mrs[0].URL, nil
	}
	if err != nil {
		return "", wrap(out, err)
	}
	if u := urlRe.FindString(out); u != "" {
		return u, nil
	}
	return "", fmt.Errorf("glab: URL not found in output: %s", strings.TrimSpace(out))
}
