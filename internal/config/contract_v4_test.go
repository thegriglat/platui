package config

import (
	"strings"
	"testing"
	"time"
)

func TestContractConfigV4Defaults(t *testing.T) {
	c, err := Parse([]byte("api_key: k\nworkspace: w\n"))
	if err != nil {
		t.Fatal(err)
	}
	g := c.GitLab
	if g.Glab != "glab" || g.MRCache == nil || *g.MRCache != 10*time.Minute || g.Draft || g.Host != "" || g.Repo != "" || g.TargetBranch != "" {
		t.Fatalf("%+v", g)
	}
}

func TestContractConfigV4Full(t *testing.T) {
	c, err := Parse([]byte("api_key: k\nworkspace: w\ngitlab:\n  host: gitlab.example.com:8443\n  repo: grp/sub/proj\n  target_branch: develop\n  draft: true\n  glab: /usr/bin/glab\n  mr_cache: 0s\n"))
	if err != nil {
		t.Fatal(err)
	}
	g := c.GitLab
	if g.Host != "gitlab.example.com:8443" || g.Repo != "grp/sub/proj" || g.TargetBranch != "develop" || !g.Draft ||
		g.Glab != "/usr/bin/glab" || g.MRCache == nil || *g.MRCache != 0 {
		t.Fatalf("%+v", g)
	}
}

func TestContractConfigV4Errors(t *testing.T) {
	cases := []struct{ name, yaml, sub string }{
		{"pr removed", "pr:\n  tool: glab\n", "pr"},
		{"repo one segment", "gitlab:\n  repo: proj\n", "gitlab.repo"},
		{"repo empty segment", "gitlab:\n  repo: a//b\n", "gitlab.repo"},
		{"host scheme", "gitlab:\n  host: https://gitlab.com\n", "gitlab.host"},
		{"mr_cache negative", "gitlab:\n  mr_cache: -1m\n", "mr_cache"},
	}
	for _, tc := range cases {
		_, err := Parse([]byte("api_key: k\nworkspace: w\n" + tc.yaml))
		if err == nil || !strings.Contains(err.Error(), tc.sub) {
			t.Errorf("%s: %v", tc.name, err)
		}
	}
}
