package config

import "strings"

func boolPtr(b bool) *bool { return &b }

func (c *Config) applyDefaults() {
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	c.WebURL = strings.TrimRight(c.WebURL, "/")
	if c.Branch.Template == "" {
		c.Branch.Template = DefaultBranchTemplate
	}
	if c.Branch.Checkout == nil {
		c.Branch.Checkout = boolPtr(true)
	}
	if c.Branch.WorktreeDir == "" {
		c.Branch.WorktreeDir = DefaultWorktreeDir
	}
	if c.Branch.CommitTemplate == "" {
		c.Branch.CommitTemplate = DefaultCommitTemplate
	}
	if c.Branch.Remote == "" {
		c.Branch.Remote = DefaultRemote
	}
	if c.GitLab.Glab == "" {
		c.GitLab.Glab = DefaultGlab
	}
	if c.GitLab.MRCache == nil {
		d := DefaultMRCache
		c.GitLab.MRCache = &d
	}
	if c.Refresh.Interval == nil {
		d := DefaultRefresh
		c.Refresh.Interval = &d
	}
	if c.Refresh.Notify == nil {
		c.Refresh.Notify = boolPtr(true)
	}
	if c.Cache.Enabled == nil {
		c.Cache.Enabled = boolPtr(true)
	}
	c.applyTabsDefaults()
	c.applyCommandsDefaults()
}
