package tui

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/gitbranch"
)

func init() {
	registerAction(action{ID: "git.stale", Title: "Clean up branches of finished tasks", Group: "git",
		Keys: []string{"X"}, Hint: "cleanup", Scopes: []string{"main", "kanban"},
		Run: func(m *model) tea.Cmd {
			s := m.s
			if s.project == nil {
				s.setStatus("no project selected", true)
				return nil
			}
			a, p := s.deps.App, *s.project
			s.setStatus("searching branches…", false)
			return func() tea.Msg {
				list, err := a.StaleBranches(context.Background(), p)
				switch {
				case err != nil:
					return statusMsg{gitErr(err), true}
				case len(list) == 0:
					return statusMsg{"no branches of finished tasks", false}
				}
				return pushScreenMsg{&staleList{list: list, marked: map[int]bool{}}}
			}
		}})
}

// staleList is the list of stale branches with marks (space); nothing is marked by default.
type staleList struct {
	list   []app.Stale
	marked map[int]bool
	cur    int
}

func (c *staleList) Title() string       { return "branch cleanup" }
func (c *staleList) Scope() string       { return "dialog" }
func (c *staleList) CapturesInput() bool { return true }

// staleGoneMsg means branches were deleted: the list is rebuilt, marks are cleared.
type staleGoneMsg struct{ names []string }

func (c *staleList) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	if g, ok := msg.(staleGoneMsg); ok {
		c.list = slices.DeleteFunc(c.list, func(st app.Stale) bool { return slices.Contains(g.names, st.Branch) })
		c.marked = map[int]bool{}
		c.cur = max(min(c.cur, len(c.list)-1), 0)
		return c, nil
	}
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return c, nil
	}
	switch keyName(k) {
	case "esc":
		return c, popScreen
	case "up":
		c.cur = max(c.cur-1, 0)
	case "down":
		c.cur = min(c.cur+1, len(c.list)-1)
	case "pgup", "home":
		c.cur = 0
	case "pgdown", "end":
		c.cur = len(c.list) - 1
	case "space":
		c.marked[c.cur] = !c.marked[c.cur]
	case "enter":
		var names []string
		for i, st := range c.list {
			if c.marked[i] {
				names = append(names, st.Branch)
			}
		}
		if len(names) == 0 {
			s.setStatus("nothing marked (space)", false)
			return c, nil
		}
		return c, confirmWith("delete "+itoa(len(names))+" branches?\n"+strings.Join(names, "\n"),
			func() tea.Cmd { return deleteBranchesCmd(s, names, false, true) })
	}
	return c, nil
}

// deleteBranchesCmd deletes branches; ErrNotMerged leads to a separate question about forced deletion.
func deleteBranchesCmd(s *session, names []string, force, askForce bool) tea.Cmd {
	a := s.deps.App
	return func() tea.Msg {
		res := a.DeleteBranches(names, force)
		ok, fail, first := 0, 0, ""
		var unmerged []string
		for _, n := range names {
			err := res[n]
			switch {
			case err == nil:
				ok++
			case askForce && errors.Is(err, gitbranch.ErrNotMerged):
				unmerged = append(unmerged, n)
			default:
				fail++
				if first == "" {
					first = n + ": " + err.Error()
				}
			}
		}
		text := "branches deleted: " + strconv.Itoa(ok) + ", errors: " + strconv.Itoa(fail)
		if first != "" {
			text += " (" + first + ")"
		}
		var gone []string
		for _, n := range names {
			if res[n] == nil {
				gone = append(gone, n)
			}
		}
		msgs := tea.BatchMsg{func() tea.Msg { return statusMsg{text, fail > 0} }, loadBranches(s),
			func() tea.Msg { return staleGoneMsg{gone} }}
		if len(unmerged) > 0 {
			msgs = append(msgs, confirmWith("force delete "+itoa(len(unmerged))+" branches?\nnot merged:\n"+strings.Join(unmerged, "\n"),
				func() tea.Cmd { return deleteBranchesCmd(s, unmerged, true, false) }))
		}
		return msgs
	}
}

func (c *staleList) View(s *session) string {
	rows := []string{s.st.accent.Bold(true).Render(" Branches of finished tasks (space — mark)")}
	for i, st := range c.list {
		mark, cur := "[ ] ", "  "
		if c.marked[i] {
			mark = "[x] "
		}
		if i == c.cur {
			cur = "> "
		}
		rows = append(rows, cur+mark+st.Branch+s.st.muted.Render("  "+itoaKey(s, st.Item.Sequence)+" "+st.Item.Name))
	}
	rows = append(rows, "", s.st.muted.Render(" space mark   Enter delete marked   Esc cancel"))
	for len(rows) < s.height-1 {
		rows = append(rows, "")
	}
	rows = append(rows, s.statusView(s.width))
	return clip2(rows, s.height)
}
