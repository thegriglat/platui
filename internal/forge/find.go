package forge

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type mrJSON struct {
	IID          int       `json:"iid"`
	Title        string    `json:"title"`
	WebURL       string    `json:"web_url"`
	State        string    `json:"state"`
	SourceBranch string    `json:"source_branch"`
	TargetBranch string    `json:"target_branch"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// list runs `glab mr list <flags> -F json` and parses the output.
func (g *GitLab) list(ctx context.Context, dir string, flags ...string) ([]MR, error) {
	args := append([]string{"mr", "list"}, flags...)
	args = append(args, "-F", "json")
	out, err := g.run(ctx, dir, args...)
	if err != nil {
		return nil, wrap(out, err)
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}
	var raw []mrJSON
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, fmt.Errorf("glab: parse mr list: %w", err)
	}
	res := make([]MR, len(raw))
	for i, m := range raw {
		res[i] = MR{m.IID, m.Title, m.WebURL, m.State, m.SourceBranch, m.TargetBranch, m.UpdatedAt}
	}
	return res, nil
}

// FindMRs finds open and merged MRs by branches and issue key.
func (g *GitLab) FindMRs(ctx context.Context, dir string, branches []string, key string) ([]MR, error) {
	seen := map[int]bool{}
	var all []MR
	add := func(mrs []MR, keep func(MR) bool) {
		for _, m := range mrs {
			if m.State == "closed" || seen[m.IID] || (keep != nil && !keep(m)) {
				continue
			}
			seen[m.IID] = true
			all = append(all, m)
		}
	}
	for _, b := range branches {
		mrs, err := g.list(ctx, dir, "-s", b, "-A")
		if err != nil {
			return nil, err
		}
		add(mrs, nil)
	}
	if key != "" {
		mrs, err := g.list(ctx, dir, "-A", "--search", key)
		if err != nil {
			return nil, err
		}
		add(mrs, nil)
		mrs, err = g.list(ctx, dir, "-A", "-o", "updated_at", "-P", "100")
		if err != nil {
			return nil, err
		}
		lk := strings.ToLower(key)
		add(mrs, func(m MR) bool { return strings.Contains(strings.ToLower(m.SourceBranch), lk) })
	}
	sort.SliceStable(all, func(i, j int) bool {
		oi, oj := all[i].State == "opened", all[j].State == "opened"
		if oi != oj {
			return oi
		}
		return all[i].UpdatedAt.After(all[j].UpdatedAt)
	})
	return all, nil
}
