package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const fullYAML = `
base_url: https://plane.example.com/
api_key: plane_api_secret
workspace: my-team
default_project: PROJ
web_url: https://web.example.com/
branch:
  template: "{identifier}-{sequence}-{slug}"
  base: main
  checkout: false
`

func TestContractConfigFull(t *testing.T) {
	c, err := Parse([]byte(fullYAML))
	if err != nil {
		t.Fatal(err)
	}
	f := false
	want := &Config{BaseURL: "https://plane.example.com", APIKey: "plane_api_secret", Workspace: "my-team",
		DefaultProject: "PROJ", WebURL: "https://web.example.com",
		Branch: Branch{Template: "{identifier}-{sequence}-{slug}", Base: "main", Checkout: &f}}
	// v2 fields get explicit defaults (C1 v2).
	d, tr := DefaultRefresh, true
	want.Branch.WorktreeDir, want.Branch.CommitTemplate, want.Branch.Remote = DefaultWorktreeDir, DefaultCommitTemplate, DefaultRemote
	m := DefaultMRCache
	want.GitLab = GitLab{Glab: "glab", MRCache: &m}
	want.Refresh = Refresh{Interval: &d, Notify: &tr}
	want.Cache = Cache{Enabled: &tr}
	want.Tabs = Tabs{HideGroups: []string{"cancelled"}, All: &tr}
	if !reflect.DeepEqual(c, want) {
		t.Fatalf("got %+v want %+v", c, want)
	}
	if c.Key() != "plane_api_secret" || c.CheckoutEnabled() {
		t.Fatal("key/checkout")
	}
}

func TestContractConfigDefaults(t *testing.T) {
	c, err := Parse([]byte("api_key: k\nworkspace: w\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseURL != DefaultBaseURL || c.Branch.Template != DefaultBranchTemplate ||
		c.Branch.Checkout == nil || !*c.Branch.Checkout || !c.CheckoutEnabled() {
		t.Fatalf("defaults: %+v", c)
	}
}

func TestContractConfigKeyEnv(t *testing.T) {
	t.Setenv("PT2_KEY", "envkey")
	c, err := Parse([]byte("api_key_env: PT2_KEY\nworkspace: w\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Key() != "envkey" {
		t.Fatal(c.Key())
	}
}

func TestContractConfigErrors(t *testing.T) {
	t.Setenv("PT2_KEY", "envkey")
	t.Setenv("PT2_EMPTY", "")
	cases := []struct{ name, yaml, sub string }{
		{"unknown key", "api_key: k\nworkspace: w\nbogus_key: 1\n", "bogus_key"},
		{"both keys", "api_key: SECRETVAL\napi_key_env: PT2_KEY\nworkspace: w\n", "both"},
		{"no key", "workspace: w\n", "api_key"},
		{"env empty", "api_key_env: PT2_EMPTY\nworkspace: w\n", "PT2_EMPTY"},
		{"env unset", "api_key_env: PT2_UNSET_XYZ\nworkspace: w\n", "PT2_UNSET_XYZ"},
		{"no workspace", "api_key: k\n", "workspace"},
		{"bad base_url", "api_key: k\nworkspace: w\nbase_url: ftp://x\n", "base_url"},
		{"bad web_url", "api_key: k\nworkspace: w\nweb_url: x.com\n", "web_url"},
		{"no placeholder", "api_key: k\nworkspace: w\nbranch:\n  template: fixed\n", "template"},
		{"unknown placeholder", "api_key: k\nworkspace: w\nbranch:\n  template: \"{slug}-{foo}\"\n", "{foo}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.yaml))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.sub) {
				t.Fatalf("error %q lacks %q", err, tc.sub)
			}
			if strings.Contains(err.Error(), "SECRETVAL") {
				t.Fatal("key leaked")
			}
		})
	}
}

func TestContractConfigLoadMissing(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nope.yml")
	_, err := Load(p)
	if err == nil || !strings.Contains(err.Error(), p) || !strings.Contains(err.Error(), "--config") {
		t.Fatalf("err: %v", err)
	}
}

func TestContractConfigLoadOK(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yml")
	if err := os.WriteFile(p, []byte(fullYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err != nil {
		t.Fatal(err)
	}
}

func TestContractConfigDefaultPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/x/cfg")
	p, _ := DefaultPath()
	if p != "/x/cfg/platui/config.yml" {
		t.Fatal(p)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/u")
	p, _ = DefaultPath()
	if p != "/home/u/.config/platui/config.yml" {
		t.Fatal(p)
	}
}
