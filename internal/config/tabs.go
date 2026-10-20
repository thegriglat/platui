package config

import (
	"fmt"
	"strings"
)

// DefaultHideGroup is the state group hidden from tabs by default.
const DefaultHideGroup = "cancelled"

// Tabs configures the state tabs (C1 v3).
type Tabs struct {
	States     []string `yaml:"states"`
	HideGroups []string `yaml:"hide_groups"` // nil in file → ["cancelled"] after Parse; explicit [] keeps all
	All        *bool    `yaml:"all"`         // nil → true
}

// TabsAll reports whether the first "All" tab is shown.
func (c *Config) TabsAll() bool {
	return c.Tabs.All == nil || *c.Tabs.All
}

func (c *Config) applyTabsDefaults() {
	if c.Tabs.HideGroups == nil {
		c.Tabs.HideGroups = []string{DefaultHideGroup}
	}
	if c.Tabs.All == nil {
		c.Tabs.All = boolPtr(true)
	}
}

func (c *Config) validateTabs() error {
	for _, g := range c.Tabs.HideGroups {
		if !contains(stateGroups, g) {
			return fmt.Errorf("tabs.hide_groups: unknown state group %q", g)
		}
	}
	seen := map[string]bool{}
	for i, s := range c.Tabs.States {
		k := strings.ToLower(strings.TrimSpace(s))
		if k == "" {
			return fmt.Errorf("tabs.states[%d]: empty state name", i)
		}
		if seen[k] {
			return fmt.Errorf("tabs.states: duplicate state %q", s)
		}
		seen[k] = true
	}
	return nil
}
