package gitbranch

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Stat is the size of a branch's change relative to its base.
type Stat struct{ Files, Insertions, Deletions int }

var statRe = regexp.MustCompile(`(\d+) (file|insertion|deletion)`)

// parseShortstat parses `git diff --shortstat` output; parts may be absent.
func parseShortstat(out string) Stat {
	var s Stat
	for _, m := range statRe.FindAllStringSubmatch(out, -1) {
		n, _ := strconv.Atoi(m[1])
		switch m[2] {
		case "file":
			s.Files = n
		case "insertion":
			s.Insertions = n
		case "deletion":
			s.Deletions = n
		}
	}
	return s
}

// DiffStat returns `git diff --shortstat base...branch`. Empty base is
// resolved as in Statuses (remote HEAD, main, master).
func (r *Repo) DiffStat(base, branch string) (Stat, error) {
	if err := ValidName(branch); err != nil {
		return Stat{}, err
	}
	ref := r.resolveBase("origin", base)
	if ref == "" {
		return Stat{}, fmt.Errorf("base branch not found")
	}
	if _, _, err := r.git("rev-parse", "--verify", "--quiet", "refs/heads/"+branch+"^{commit}"); err != nil {
		return Stat{}, fmt.Errorf("branch %q does not exist", branch)
	}
	out, _, err := r.git("diff", "--shortstat", ref+"...refs/heads/"+branch)
	if err != nil {
		return Stat{}, err
	}
	return parseShortstat(strings.TrimSpace(out)), nil
}
