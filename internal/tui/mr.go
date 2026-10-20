package tui

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/forge"
	"github.com/thegriglat/platui/internal/plane"
)

func init() {
	registerAction(action{ID: "git.mr", Title: "Go to task MR", Group: "git",
		Keys: []string{"M"}, Hint: "MR", Scopes: cardScopes,
		Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd { return goToMR(m.s, it) })})
}

func openMR(s *session, mr forge.MR) tea.Cmd {
	return func() tea.Msg {
		if err := s.deps.OpenURL(mr.URL); err != nil {
			return statusMsg{"browser: " + err.Error(), true}
		}
		return statusMsg{"opened: " + mr.URL, false}
	}
}

func goToMR(s *session, it plane.WorkItem) tea.Cmd {
	a, p := s.deps.App, *s.project
	s.setStatus("searching MRs…", false)
	return func() tea.Msg {
		mrs, err := a.MRs(context.Background(), p, it, false)
		switch {
		case errors.Is(err, forge.ErrNotFound):
			return statusMsg{"glab not installed", true}
		case err != nil:
			return statusMsg{err.Error(), true}
		case len(mrs) == 0:
			return statusMsg{"no MRs found", false}
		case len(mrs) == 1:
			return openMR(s, mrs[0])()
		}
		rows := make([]pickRow, len(mrs))
		for i, mr := range mrs {
			rows[i] = pickRow{Label: "!" + itoa(mr.IID) + " " + mr.Title + s.st.muted.Render("  "+mr.State+"  "+mr.SourceBranch)}
		}
		return pushScreenMsg{newListScreen("Task MRs "+itoaKey(s, it.Sequence), rows, 0, func(i int) tea.Cmd { return openMR(s, mrs[i]) })}
	}
}
