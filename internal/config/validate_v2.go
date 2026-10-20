package config

import (
	"fmt"
	"strconv"
	"strings"
)

func (c *Config) validateV2() error {
	checks := []func() error{c.validateBranch, c.validateGitLab, c.validateFilters, c.validateTheme, c.validateRefresh, c.validateTabs, c.validateUI, c.validateCommands}
	for _, f := range checks {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) validateBranch() error {
	b := c.Branch
	if b.Template != "" {
		if err := checkTemplate("branch.template", b.Template, []string{"{identifier}", "{sequence}", "{slug}"}, true); err != nil {
			return err
		}
	}
	if err := checkTemplate("branch.worktree_dir", b.WorktreeDir, []string{"{repo}", "{branch}"}, false); err != nil {
		return err
	}
	return checkTemplate("branch.commit_template", b.CommitTemplate,
		[]string{"{identifier}", "{sequence}", "{title}", "{slug}"}, b.CommitTemplate != "")
}

func (c *Config) validateGitLab() error {
	g := c.GitLab
	if strings.Contains(g.Host, "://") || strings.ContainsAny(g.Host, "/ ") {
		return fmt.Errorf("gitlab.host: %q must be a host name[:port] without scheme", g.Host)
	}
	if g.Repo != "" {
		parts := strings.Split(g.Repo, "/")
		ok := len(parts) >= 2
		for _, p := range parts {
			if p == "" {
				ok = false
			}
		}
		if !ok {
			return fmt.Errorf("gitlab.repo: %q must look like group/project", g.Repo)
		}
	}
	if d := g.MRCache; d != nil && *d < 0 {
		return fmt.Errorf("gitlab.mr_cache: %s is negative", *d)
	}
	return nil
}

func (c *Config) validateFilters() error {
	if len(c.Filters) > MaxFilters {
		return fmt.Errorf("filters: %d filters, at most %d allowed", len(c.Filters), MaxFilters)
	}
	for i, f := range c.Filters {
		if f.Name == "" {
			return fmt.Errorf("filters[%d]: name is required", i)
		}
		for _, g := range f.StateGroups {
			if !contains(stateGroups, g) {
				return fmt.Errorf("filter %q: unknown state group %q", f.Name, g)
			}
		}
		for _, p := range f.Priorities {
			if !contains(priorities, p) {
				return fmt.Errorf("filter %q: unknown priority %q", f.Name, p)
			}
		}
	}
	return nil
}

func (c *Config) validateTheme() error {
	for k, v := range c.Theme.Colors {
		if !contains(ColorKeys, k) {
			return fmt.Errorf("theme.colors: unknown color key %q", k)
		}
		if !validColor(v) {
			return fmt.Errorf("theme.colors.%s: %q is not #RRGGBB or 0..255", k, v)
		}
	}
	return nil
}

func validColor(v string) bool {
	if len(v) == 7 && v[0] == '#' {
		_, err := strconv.ParseUint(v[1:], 16, 32)
		return err == nil
	}
	n, err := strconv.Atoi(v)
	return err == nil && n >= 0 && n <= 255 && v == strconv.Itoa(n)
}

func (c *Config) validateRefresh() error {
	if d := c.Refresh.Interval; d != nil && (*d < 0 || (*d > 0 && *d < MinRefresh)) {
		return fmt.Errorf("refresh.interval: %s is invalid; use 0 (off) or at least %s", *d, MinRefresh)
	}
	return nil
}
