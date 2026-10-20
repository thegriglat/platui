package tui

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
)

func init() {
	registerAction(action{ID: "git.branch", Title: "Task branch: name from template, create/switch", Group: "git",
		Keys: []string{"b"}, Hint: "branch", Scopes: cardScopes,
		Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
			s := m.s
			name, err := s.deps.App.BranchName(*s.project, it)
			if err != nil {
				s.setStatus(err.Error(), true)
				return nil
			}
			c := &inputScreen{title: "Branch for " + itoaKey(s, it.Sequence), text: []rune(name),
				hint: "Enter create/switch   Esc cancel"}
			c.onSubmit = func(c *inputScreen) tea.Cmd {
				n := strings.TrimSpace(string(c.text))
				if err := gitbranch.ValidName(n); err != nil {
					s.setStatus("branch name: "+err.Error(), true)
					return nil
				}
				return tea.Sequence(popScreen, branchOnly(s, it, n, false))
			}
			return pushScreen(c)
		})})
}

// gitErr returns the error text for a git scenario.
func gitErr(err error) string {
	if errors.Is(err, app.ErrNoRepo) {
		return "directory is not a git repository"
	}
	return err.Error()
}

// branchOnly runs app.Branch: git only, no changes in Plane. An existing branch
// is switched to only after confirmation (switchExisting).
func branchOnly(s *session, it plane.WorkItem, name string, switchExisting bool) tea.Cmd {
	a, p := s.deps.App, *s.project
	return func() tea.Msg {
		res, err := a.Branch(p, it, name, nil, switchExisting)
		if errors.Is(err, gitbranch.ErrBranchExists) && !switchExisting {
			return confirmWith("branch "+name+" exists, switch to it?",
				func() tea.Cmd { return branchOnly(s, it, name, true) })()
		}
		if err != nil {
			return statusMsg{gitErr(err), true}
		}
		verb := "branch"
		switch {
		case res.Created:
			verb = "created branch"
		case res.Switched:
			verb = "switched to branch"
		}
		return tea.BatchMsg{loadBranches(s), func() tea.Msg { return statusMsg{verb + " " + res.Name, false} }}
	}
}
