package config

import "time"

// Defaults for C1 v2 fields.
const (
	DefaultWorktreeDir    = "../{repo}-{branch}"
	DefaultCommitTemplate = "{identifier}-{sequence}: {title}"
	DefaultRemote         = "origin"
	DefaultRefresh        = 2 * time.Minute
	MinRefresh            = 30 * time.Second
	DefaultGlab           = "glab"
	DefaultMRCache        = 10 * time.Minute
)

// Workflow names the states used by the "start work" and "ready for review" scenarios.
type Workflow struct {
	StartState  string `yaml:"start_state"`
	ReviewState string `yaml:"review_state"`
}

// GitLab configures merge request creation and lookup through glab (C1 v4).
type GitLab struct {
	Host         string         `yaml:"host"`
	Repo         string         `yaml:"repo"`
	TargetBranch string         `yaml:"target_branch"`
	Draft        bool           `yaml:"draft"`
	Glab         string         `yaml:"glab"`
	MRCache      *time.Duration `yaml:"mr_cache"`
}

// Filter is a saved issue filter bound to a key 1..9 by list order.
type Filter struct {
	Name        string   `yaml:"name"`
	Project     string   `yaml:"project"`
	Mine        bool     `yaml:"mine"`
	StateGroups []string `yaml:"state_groups"`
	States      []string `yaml:"states"`
	Priorities  []string `yaml:"priorities"`
	Labels      []string `yaml:"labels"`
	Text        string   `yaml:"text"`
}

// Theme configures colors.
type Theme struct {
	NoColor bool              `yaml:"no_color"`
	Colors  map[string]string `yaml:"colors"`
}

// Refresh configures background polling.
type Refresh struct {
	Interval *time.Duration `yaml:"interval"`
	Notify   *bool          `yaml:"notify"`
}

// Cache configures the on-disk cache.
type Cache struct {
	Enabled *bool  `yaml:"enabled"`
	Dir     string `yaml:"dir"`
}

// MaxFilters is the maximum number of saved filters.
const MaxFilters = 9

var (
	stateGroups = []string{"backlog", "unstarted", "started", "completed", "cancelled"}
	priorities  = []string{"urgent", "high", "medium", "low", "none"}
)
