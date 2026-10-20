package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
)

// statEntry is the app.DiffStat result for a task (loading: request in flight).
type statEntry struct {
	stat    gitbranch.Stat
	ok      bool
	loading bool
}

type diffStatMsg struct {
	key  string
	stat gitbranch.Stat
	ok   bool
	err  error
}

// statKey is the session cache key: task and update time.
func statKey(it plane.WorkItem) string { return fmt.Sprintf("%s|%d", it.ID, it.UpdatedAt.UnixNano()) }

// loadDiffStat requests branch statistics asynchronously.
func loadDiffStat(s *session, it plane.WorkItem) tea.Cmd {
	a, p := s.deps.App, *s.project
	return func() tea.Msg {
		st, ok, err := a.DiffStat(p, it)
		return diffStatMsg{key: statKey(it), stat: st, ok: ok, err: err}
	}
}

// wantStat: if the task has no cache entry yet, marks it as loading and returns the request.
func (c *cardView) wantStat(s *session, id string) tea.Cmd {
	it, ok := s.itemByID(id)
	if !ok || s.deps.App == nil || s.project == nil {
		return nil
	}
	k := statKey(it)
	if _, ok := c.stats[k]; ok {
		return nil
	}
	c.stats[k] = statEntry{loading: true}
	return loadDiffStat(s, it)
}

// onStat stores the result; an error yields a "no branch" entry (no line in the preview).
func (c *cardView) onStat(m diffStatMsg) {
	c.stats[m.key] = statEntry{stat: m.stat, ok: m.ok && m.err == nil}
}

// statLine returns "Branch: +ins −del (n files)"; empty if there is no branch or no data yet.
func (c *cardView) statLine(it plane.WorkItem) string {
	e, ok := c.stats[statKey(it)]
	if !ok || e.loading || !e.ok {
		return ""
	}
	return fmt.Sprintf("Branch: +%d −%d (%s)", e.stat.Insertions, e.stat.Deletions, filesWord(e.stat.Files))
}

func filesWord(n int) string {
	if n == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", n)
}
