package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestContractConfigExample(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "config.example.yml"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if c.Workspace != "my-team" || c.Key() != "plane_api_xxx" || c.Branch.Template != DefaultBranchTemplate ||
		c.GitLab.Glab != "glab" || *c.GitLab.MRCache != 10*time.Minute ||
		*c.Refresh.Interval != DefaultRefresh || len(c.Filters) != 1 || !c.TabsAll() {
		t.Fatalf("unexpected: %+v", c)
	}
}
