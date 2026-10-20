package config

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

const v2YAML = `
api_key: k
workspace: w
branch:
  worktree: true
  worktree_dir: "../{repo}-{branch}"
  commit_template: "{identifier}-{sequence}: {title}"
  remote: upstream
workflow:
  start_state: Doing
  review_state: Review
filters:
  - name: Мои
    mine: true
    state_groups: [started]
    priorities: [urgent, high]
    text: bug
theme:
  no_color: true
  colors:
    accent: "#abcdef"
    muted: "123"
editor: nano
refresh:
  interval: 90s
  notify: false
cache:
  enabled: false
  dir: /tmp/c
`

func TestContractConfigV2Full(t *testing.T) {
	c, err := Parse([]byte(v2YAML))
	if err != nil {
		t.Fatal(err)
	}
	d := 90 * time.Second
	f := false
	want := Config{
		Branch:   Branch{Worktree: true, WorktreeDir: "../{repo}-{branch}", CommitTemplate: "{identifier}-{sequence}: {title}", Remote: "upstream"},
		Workflow: Workflow{"Doing", "Review"},
		Filters:  []Filter{{Name: "Мои", Mine: true, StateGroups: []string{"started"}, Priorities: []string{"urgent", "high"}, Text: "bug"}},
		Theme:    Theme{true, map[string]string{"accent": "#abcdef", "muted": "123"}},
		Editor:   "nano",
		Refresh:  Refresh{&d, &f},
		Cache:    Cache{&f, "/tmp/c"},
	}
	got := *c
	if !reflect.DeepEqual(got.Branch.Worktree, true) || got.Branch.Remote != "upstream" ||
		!reflect.DeepEqual(got.Workflow, want.Workflow) ||
		!reflect.DeepEqual(got.Filters, want.Filters) || !reflect.DeepEqual(got.Theme, want.Theme) ||
		got.Editor != "nano" || !reflect.DeepEqual(got.Refresh, want.Refresh) || !reflect.DeepEqual(got.Cache, want.Cache) {
		t.Fatalf("got %+v", got)
	}
	if c.RefreshInterval() != d || c.NotifyEnabled() || c.CacheEnabled() || c.Color("accent") != "#abcdef" || c.Color("muted") != "123" {
		t.Fatal("accessors")
	}
}

func TestContractConfigV2Defaults(t *testing.T) {
	c, err := Parse([]byte("api_key: k\nworkspace: w\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Branch.WorktreeDir != DefaultWorktreeDir || c.Branch.CommitTemplate != DefaultCommitTemplate ||
		c.Branch.Remote != "origin" || c.Branch.Worktree ||
		c.RefreshInterval() != 2*time.Minute || !c.NotifyEnabled() || !c.CacheEnabled() ||
		c.Refresh.Interval == nil || c.Refresh.Notify == nil || c.Cache.Enabled == nil {
		t.Fatalf("defaults: %+v", c)
	}
	for _, k := range ColorKeys {
		if c.Color(k) == "" {
			t.Fatal("no default color", k)
		}
	}
	if c.Color("nope") != "" {
		t.Fatal("unknown color")
	}
}

func TestContractConfigV2IntervalOff(t *testing.T) {
	c, err := Parse([]byte("api_key: k\nworkspace: w\nrefresh:\n  interval: 0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.RefreshInterval() != 0 {
		t.Fatal(c.RefreshInterval())
	}
}

func TestContractConfigV2Errors(t *testing.T) {
	base := "api_key: SECRETVAL\nworkspace: w\n"
	many := ""
	for i := 0; i < 10; i++ {
		many += "  - name: f\n"
	}
	cases := []struct{ name, yaml, sub string }{
		{"profile", "profile: x\n", "profile"},
		{"profiles", "profiles: {}\n", "profiles"},
		{"worktree_dir ph", "branch:\n  worktree_dir: \"{slug}\"\n", "{slug}"},
		{"commit ph", "branch:\n  commit_template: \"{foo}\"\n", "{foo}"},
		{"commit none", "branch:\n  commit_template: fixed\n", "commit_template"},
		{"too many filters", "filters:\n" + many, "filters"},
		{"filter no name", "filters:\n  - mine: true\n", "name"},
		{"state group", "filters:\n  - name: F1\n    state_groups: [x]\n", "F1"},
		{"priority", "filters:\n  - name: F2\n    priorities: [x]\n", "F2"},
		{"color key", "theme:\n  colors:\n    bogus: \"1\"\n", "bogus"},
		{"color value", "theme:\n  colors:\n    accent: red\n", "accent"},
		{"color range", "theme:\n  colors:\n    accent: \"256\"\n", "accent"},
		{"interval small", "refresh:\n  interval: 10s\n", "refresh.interval"},
		{"interval neg", "refresh:\n  interval: -1m\n", "refresh.interval"},
		{"interval bad", "refresh:\n  interval: soon\n", "interval"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(base + tc.yaml))
			if err == nil || !strings.Contains(err.Error(), tc.sub) {
				t.Fatalf("err %v lacks %q", err, tc.sub)
			}
			if strings.Contains(err.Error(), "SECRETVAL") {
				t.Fatal("key leaked")
			}
		})
	}
}

func TestContractConfigV2Env(t *testing.T) {
	c := &Config{}
	t.Setenv("NO_COLOR", "")
	if c.ColorsDisabled() {
		t.Fatal("no_color")
	}
	t.Setenv("NO_COLOR", "1")
	if !c.ColorsDisabled() {
		t.Fatal("NO_COLOR")
	}
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	if c.EditorCmd() != "vi" {
		t.Fatal(c.EditorCmd())
	}
	t.Setenv("EDITOR", "ed")
	if c.EditorCmd() != "ed" {
		t.Fatal("EDITOR")
	}
	t.Setenv("VISUAL", "vis")
	if c.EditorCmd() != "vis" {
		t.Fatal("VISUAL")
	}
	c.Editor = "me"
	if c.EditorCmd() != "me" {
		t.Fatal("editor")
	}
	t.Setenv("XDG_CACHE_HOME", "/x/cache")
	if d, _ := c.CacheDir(); d != "/x/cache/platui" {
		t.Fatal(d)
	}
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("HOME", "/home/u")
	if d, _ := c.CacheDir(); d != "/home/u/.cache/platui" {
		t.Fatal(d)
	}
	c.Cache.Dir = "/custom"
	if d, _ := c.CacheDir(); d != "/custom" {
		t.Fatal(d)
	}
}

func TestContractConfigV2NilAccessors(t *testing.T) {
	c := &Config{}
	if !c.CacheEnabled() || !c.NotifyEnabled() || c.RefreshInterval() != DefaultRefresh {
		t.Fatal("nil defaults")
	}
}

func TestContractConfigV2UnknownRefreshKey(t *testing.T) {
	_, err := Parse([]byte("api_key: k\nworkspace: w\nrefresh:\n  intervl: 1m\n"))
	if err == nil || !strings.Contains(err.Error(), "intervl") {
		t.Fatalf("err: %v", err)
	}
	_, err = Parse([]byte("api_key: k\nworkspace: w\ncache:\n  dirr: x\n"))
	if err == nil || !strings.Contains(err.Error(), "dirr") {
		t.Fatalf("err: %v", err)
	}
}

func TestContractConfigV2Palette(t *testing.T) {
	want := map[string]string{"accent": "#7D56F4", "muted": "245", "text": "252", "error": "#FF5F87",
		"success": "#5FD787", "warning": "#FFD75F", "selection": "#3A3A5A", "border": "240",
		"priority_urgent": "#FF5F5F", "priority_high": "#FFAF5F", "priority_medium": "#FFD75F", "priority_low": "#87AFFF"}
	c := &Config{}
	if len(want) != len(ColorKeys) {
		t.Fatal("key count")
	}
	for k, v := range want {
		if c.Color(k) != v {
			t.Fatalf("%s: %q want %q", k, c.Color(k), v)
		}
	}
}
