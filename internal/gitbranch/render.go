package gitbranch

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
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

// ValidName checks name by the rules of `git check-ref-format --branch` (go-git, no process per call).
func ValidName(name string) error {
	if name == "" {
		return errors.New("invalid branch name: empty")
	}
	if strings.HasPrefix(name, "-") || name == "HEAD" {
		return fmt.Errorf("invalid branch name %q", name)
	}
	if err := plumbing.NewBranchReferenceName(name).Validate(); err != nil {
		return fmt.Errorf("invalid branch name %q: %w", name, err)
	}
	return nil
}
