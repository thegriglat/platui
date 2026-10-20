package gitbranch

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Item is the task data for the branch name template.
type Item struct {
	Identifier string
	Sequence   int
	Title      string
}

var placeholderRe = regexp.MustCompile(`\{[^{}]*\}`)

// Render fills the template placeholders {identifier}, {sequence}, {slug}
// and validates the resulting name.
func Render(template string, it Item) (string, error) {
	for _, m := range placeholderRe.FindAllString(template, -1) {
		switch m {
		case "{identifier}", "{sequence}", "{slug}":
		default:
			return "", fmt.Errorf("unknown placeholder %s in template", m)
		}
	}
	name := strings.NewReplacer(
		"{identifier}", it.Identifier,
		"{sequence}", strconv.Itoa(it.Sequence),
		"{slug}", Slug(it.Title, 40),
	).Replace(template)
	if err := ValidName(name); err != nil {
		return "", err
	}
	return name, nil
}

// CommitMessage fills {identifier}, {sequence}, {title} (single line) and
// {slug}; an unknown placeholder is an error.
func CommitMessage(template string, it Item) (string, error) {
	for _, m := range placeholderRe.FindAllString(template, -1) {
		switch m {
		case "{identifier}", "{sequence}", "{title}", "{slug}":
		default:
			return "", fmt.Errorf("unknown placeholder %s in commit template", m)
		}
	}
	title := strings.Join(strings.Fields(it.Title), " ")
	return strings.NewReplacer(
		"{identifier}", it.Identifier,
		"{sequence}", strconv.Itoa(it.Sequence),
		"{title}", title,
		"{slug}", Slug(it.Title, 40),
	).Replace(template), nil
}

// WorktreePath expands {repo} (base name of toplevel) and {branch} (name with
// "/" replaced by "-"); relative results are taken from toplevel. The result
// is absolute and cleaned.
func WorktreePath(template, toplevel, branch string) string {
	p := strings.NewReplacer(
		"{repo}", filepath.Base(toplevel),
		"{branch}", strings.ReplaceAll(branch, "/", "-"),
	).Replace(template)
	if !filepath.IsAbs(p) {
		p = filepath.Join(toplevel, p)
	}
	return filepath.Clean(p)
}

// ValidName checks name with `git check-ref-format --branch`.
func ValidName(name string) error {
	if name == "" {
		return errors.New("invalid branch name: empty")
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("invalid branch name %q", name)
	}
	cmd := exec.Command("git", "check-ref-format", "--branch", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return fmt.Errorf("invalid branch name %q: %s", name, strings.TrimSpace(stderr.String()))
		}
		return fmt.Errorf("git check-ref-format: %w", err)
	}
	return nil
}
