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
	registerAction(action{ID: "git.branch", Title: "Ветка задачи: имя по шаблону, создать/переключить", Group: "git",
		Keys: []string{"b"}, Hint: "ветка", Scopes: cardScopes,
		Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
			s := m.s
			name, err := s.deps.App.BranchName(*s.project, it)
			if err != nil {
				s.setStatus(err.Error(), true)
				return nil
			}
			c := &inputScreen{title: "Ветка для " + itoaKey(s, it.Sequence), text: []rune(name),
				hint: "Enter создать/переключить   Esc отмена"}
			c.onSubmit = func(c *inputScreen) tea.Cmd {
				n := strings.TrimSpace(string(c.text))
				if err := gitbranch.ValidName(n); err != nil {
					s.setStatus("имя ветки: "+err.Error(), true)
					return nil
				}
				return tea.Sequence(popScreen, branchOnly(s, it, n, false))
			}
			return pushScreen(c)
		})})
}

// gitErr — текст ошибки git-сценария.
func gitErr(err error) string {
	if errors.Is(err, app.ErrNoRepo) {
		return "каталог не git-репозиторий"
	}
	return err.Error()
}

// branchOnly — app.Branch: только git, без изменений в Plane. Существующая ветка
// переключается только после подтверждения (switchExisting).
func branchOnly(s *session, it plane.WorkItem, name string, switchExisting bool) tea.Cmd {
	a, p := s.deps.App, *s.project
	return func() tea.Msg {
		res, err := a.Branch(p, it, name, nil, switchExisting)
		if errors.Is(err, gitbranch.ErrBranchExists) && !switchExisting {
			return confirmWith("ветка "+name+" есть, переключиться?",
				func() tea.Cmd { return branchOnly(s, it, name, true) })()
		}
		if err != nil {
			return statusMsg{gitErr(err), true}
		}
		verb := "ветка"
		switch {
		case res.Created:
			verb = "создана ветка"
		case res.Switched:
			verb = "переключено на ветку"
		}
		return tea.BatchMsg{loadBranches(s), func() tea.Msg { return statusMsg{verb + " " + res.Name, false} }}
	}
}
