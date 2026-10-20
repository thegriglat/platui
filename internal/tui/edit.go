package tui

import (
	"context"
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/text"
)

// runEditor is replaced in tests.
var runEditor = func(sess *text.EditSession, editor string, fn tea.ExecCallback) tea.Cmd {
	return tea.ExecProcess(sess.Command(editor), fn)
}

// editDoneMsg reports that the editor was closed; sess is read and removed in finishEdit.
type editDoneMsg struct {
	kind      string // "desc" | "comment"
	projectID string
	itemID    string
	loaded    time.Time // task UpdatedAt when the editor was opened
	sess      *text.EditSession
	err       error
}

// startEdit opens $EDITOR on a temporary file with initial; if the previous submit
// of this edit failed, it first offers the saved text.
func startEdit(s *session, kind string, it plane.WorkItem, initial, hint string) tea.Cmd {
	if d, ok := s.savedDraft(kind, it.ID); ok && s.project != nil {
		pid := s.project.ID
		msg := editDoneMsg{kind: kind, projectID: pid, itemID: it.ID, loaded: d.loaded, sess: d.sess}
		c := &confirmScreen{question: "Use the saved text? y/N\n" + d.sess.Path,
			yes: func() tea.Cmd { return finishEdit(s, msg) },
			// refusal: the file is removed and the edit starts afresh
			no: func() tea.Cmd {
				d.sess.Close()
				return openEditor(s, kind, it, initial, hint)
			}}
		return pushScreen(c)
	}
	return openEditor(s, kind, it, initial, hint)
}

func openEditor(s *session, kind string, it plane.WorkItem, initial, hint string) tea.Cmd {
	sess, err := text.NewEditSession(initial, hint)
	if err != nil {
		s.setStatus(err.Error(), true)
		return nil
	}
	pid := s.project.ID
	return runEditor(sess, s.deps.Config.EditorCmd(), func(err error) tea.Msg {
		return editDoneMsg{kind: kind, projectID: pid, itemID: it.ID, loaded: it.UpdatedAt, sess: sess, err: err}
	})
}

// finishEdit reads the edit result and sends it to Plane. The file is removed on
// success, cancel and editor error; on a submit error it is kept (editFailMsg).
func finishEdit(s *session, m editDoneMsg) tea.Cmd {
	if m.err != nil {
		m.sess.Close()
		return errCmd(errors.New("editor: " + m.err.Error()))
	}
	md, err := m.sess.Result()
	if err != nil {
		m.sess.Close()
		if errors.Is(err, text.ErrEmptyEdit) {
			s.setStatus("empty text, cancelled", false)
			return nil
		}
		return errCmd(err)
	}
	a := s.deps.App
	fail := func(err error) tea.Msg {
		return editFailMsg{kind: m.kind, itemID: m.itemID, sess: m.sess, loaded: m.loaded, err: err}
	}
	return func() tea.Msg {
		ctx := context.Background()
		if m.kind == "comment" {
			if _, err := a.Comment(ctx, m.projectID, m.itemID, md); err != nil {
				return fail(err)
			}
			m.sess.Close()
			return tea.BatchMsg{func() tea.Msg { return statusMsg{"comment added", false} },
				func() tea.Msg { return loadComments(s, m.projectID, m.itemID)() }}
		}
		cur, err := a.Item(ctx, m.projectID, m.itemID)
		if err != nil {
			return fail(err)
		}
		save := func() tea.Cmd { return saveDescription(a, m, md, fail) }
		if cur.UpdatedAt.After(m.loaded) {
			ask := pushScreen(&confirmScreen{question: "Description changed in Plane since loading, overwrite?",
				yes: save,
				// refusal keeps the text: the session goes to drafts
				no: func() tea.Cmd { return func() tea.Msg { return fail(errors.New("cancelled")) } }})
			return tea.BatchMsg{func() tea.Msg { return itemMsg{item: cur} }, ask}
		}
		return save()()
	}
}

func saveDescription(a *app.App, m editDoneMsg, md string, fail func(error) tea.Msg) tea.Cmd {
	return func() tea.Msg {
		it, err := a.SetDescription(context.Background(), m.projectID, m.itemID, md)
		if err != nil {
			return fail(err)
		}
		m.sess.Close()
		return tea.BatchMsg{func() tea.Msg { return itemMsg{item: it} },
			func() tea.Msg { return statusMsg{"description updated", false} }}
	}
}

func errCmd(err error) tea.Cmd { return func() tea.Msg { return errStatus(err) } }

func init() {
	registerAction(action{ID: "item.edit", Title: "Description in $EDITOR", Group: "task", Keys: []string{"e"}, Hint: "description", Scopes: cardScopes,
		Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
			return startEdit(m.s, "desc", it, text.HTMLToMarkdown(it.DescHTML), "")
		})})
	registerAction(action{ID: "item.comment", Title: "Comment in $EDITOR", Group: "task", Keys: []string{"c"}, Hint: "comment", Scopes: cardScopes,
		Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
			return startEdit(m.s, "comment", it, "", "Comment on "+itoaKey(m.s, it.Sequence)+" (Markdown); empty cancels")
		})})
}
