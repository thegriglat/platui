package tui

import (
	"context"
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/text"
)

// createDescMsg reports that the description editor was closed.
type createDescMsg struct {
	owner *inputScreen
	sess  *text.EditSession
	err   error
}

// createdMsg is the result of app.CreateQuick.
type createdMsg struct {
	owner *inputScreen
	item  plane.WorkItem
	err   error
}

func init() {
	registerAction(action{ID: "item.new", Title: "New task (title !priority @user #label; Ctrl+E: description)",
		Group: "task", Keys: []string{"n"}, Hint: "new", Scopes: []string{"main", "kanban"},
		Run: func(m *model) tea.Cmd {
			if m.s.project == nil {
				m.s.setStatus("no project selected", true)
				return nil
			}
			return pushScreen(newCreateScreen(m.s))
		}})
}

func newCreateScreen(s *session) *inputScreen {
	desc := ""
	c := &inputScreen{title: "New task: title !priority @user #label",
		hint: "Enter create   Ctrl+E description in $EDITOR   Esc cancel"}
	c.simCur = -1
	var simIDs []string
	c.similar = func(line string) []string {
		simIDs = simIDs[:0]
		q, err := text.ParseQuick(line)
		if err != nil || s.index == nil {
			return nil
		}
		var out []string
		for _, h := range s.index.Search(q.Title) {
			if it, ok := s.itemByID(h.ID); ok && len(out) < 5 {
				out = append(out, itoaKey(s, it.Sequence)+"  "+it.Name)
				simIDs = append(simIDs, it.ID)
			}
		}
		return out
	}
	c.onPick = func(c *inputScreen, i int) tea.Cmd {
		it, ok := s.itemByID(simIDs[i])
		if !ok {
			return nil
		}
		return tea.Sequence(popScreen, func() tea.Msg { return openItemMsg(s, it) })
	}
	c.extra = func() string {
		if desc != "" {
			return "description: " + strings.SplitN(desc, "\n", 2)[0]
		}
		return ""
	}
	c.onKey = func(c *inputScreen, key string) tea.Cmd {
		if key != "ctrl+e" {
			return nil
		}
		sess, err := text.NewEditSession(desc, "Task description (Markdown); empty means no description")
		if err != nil {
			s.setStatus(err.Error(), true)
			return func() tea.Msg { return nil }
		}
		return runEditor(sess, s.deps.Config.EditorCmd(), func(err error) tea.Msg {
			return createDescMsg{owner: c, sess: sess, err: err}
		})
	}
	c.onSubmit = func(c *inputScreen) tea.Cmd {
		if strings.TrimSpace(string(c.text)) == "" {
			s.setStatus("empty title", true)
			return nil
		}
		c.busy = true
		a, pid, line, d := s.deps.App, s.project.ID, string(c.text), desc
		return func() tea.Msg {
			it, err := a.CreateQuick(context.Background(), pid, line, d)
			return createdMsg{owner: c, item: it, err: err}
		}
	}
	c.onMsg = func(c *inputScreen, msg tea.Msg) tea.Cmd {
		switch m := msg.(type) {
		case createDescMsg:
			if m.owner != c {
				return nil
			}
			defer m.sess.Close()
			if m.err != nil {
				s.setStatus("editor: "+m.err.Error(), true)
				return nil
			}
			md, err := m.sess.Result()
			if err != nil && !errors.Is(err, text.ErrEmptyEdit) {
				s.setStatus(err.Error(), true)
				return nil
			}
			desc = md
		case createdMsg:
			if m.owner != c {
				return nil
			}
			c.busy = false
			if m.err != nil {
				s.setStatus(m.err.Error(), true)
				return nil
			}
			if s.project != nil && s.project.ID == m.item.ProjectID {
				s.mergeItem(m.item)
				s.cursor = m.item.ID
				s.revealTab(m.item)
			}
			s.setStatus("created "+itoaKey(s, m.item.Sequence)+": "+m.item.Name, false)
			return popScreen
		}
		return nil
	}
	return c
}
