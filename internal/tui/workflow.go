package tui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/plane"
)

func init() {
	registerAction(action{ID: "git.start", Title: "Старт работы: ветка, состояние, на себя", Group: "git",
		Keys: []string{"S"}, Hint: "старт", Scopes: cardScopes,
		Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd { return startWork(m.s, it, "") })})
	registerAction(action{ID: "git.review", Title: "Готово к ревью: push, MR, состояние", Group: "git",
		Keys: []string{"W"}, Hint: "ревью", Scopes: cardScopes,
		Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd { return reviewWork(m.s, it) })})
}

func tail(msg string, warns []string) string {
	if len(warns) > 0 {
		msg += " (" + strings.Join(warns, "; ") + ")"
	}
	return msg
}

// refreshAfter — перечитывание задачи и статусов веток после git-действия.
func refreshAfter(s *session, pid, id string) tea.Cmd {
	return tea.Batch(func() tea.Msg { return loadItem(s, pid, id)() }, loadBranches(s))
}

// startWork — app.Start; branch пусто — имя по шаблону.
func startWork(s *session, it plane.WorkItem, branch string) tea.Cmd {
	a, p := s.deps.App, *s.project
	s.setStatus("старт работы…", false)
	return func() tea.Msg {
		res, err := a.Start(context.Background(), p, it, app.StartOptions{Branch: branch})
		var out []tea.Cmd
		if res.Branch.Name != "" || err == nil {
			out = append(out, refreshAfter(s, p.ID, it.ID))
		}
		verb := "ветка"
		switch {
		case res.Branch.Created:
			verb = "создана ветка"
		case res.Branch.Switched:
			verb = "переключено на ветку"
		}
		msg := statusMsg{tail(verb+" "+res.Branch.Name+extraStart(res), res.Warnings), false}
		if err != nil {
			msg = statusMsg{gitErr(err), true}
		}
		out = append(out, func() tea.Msg { return msg })
		return tea.BatchMsg(out)
	}
}

func extraStart(r app.StartResult) string {
	out := ""
	if r.State != "" {
		out += ", состояние: " + r.State
	}
	if r.Assigned {
		out += ", назначена на вас"
	}
	return out
}

func reviewWork(s *session, it plane.WorkItem) tea.Cmd {
	a, p := s.deps.App, *s.project
	s.setStatus("отправка ветки и MR…", false)
	return func() tea.Msg {
		res, err := a.Review(context.Background(), p, it)
		out := []tea.Cmd{refreshAfter(s, p.ID, it.ID)}
		msg := "готово к ревью"
		if res.MRURL != "" {
			msg += ", MR: " + res.MRURL
		}
		if res.State != "" {
			msg += ", состояние: " + res.State
		}
		st := statusMsg{tail(msg, res.Warnings), false}
		if err != nil {
			st = statusMsg{gitErr(err), true}
		}
		out = append(out, func() tea.Msg { return st })
		return tea.BatchMsg(out)
	}
}
