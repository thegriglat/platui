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

// execProcess подменяется в тестах.
var runEditor = func(sess *text.EditSession, editor string, fn tea.ExecCallback) tea.Cmd {
	return tea.ExecProcess(sess.Command(editor), fn)
}

// editDoneMsg — редактор закрыт; sess читается и удаляется в finishEdit.
type editDoneMsg struct {
	kind      string // "desc" | "comment"
	projectID string
	itemID    string
	loaded    time.Time // UpdatedAt задачи на момент открытия редактора
	sess      *text.EditSession
	err       error
}

// startEdit открывает $EDITOR на временном файле с initial; если прошлая отправка
// этой правки не удалась — сначала предлагает сохранённый текст.
func startEdit(s *session, kind string, it plane.WorkItem, initial, hint string) tea.Cmd {
	if d, ok := s.savedDraft(kind, it.ID); ok && s.project != nil {
		pid := s.project.ID
		msg := editDoneMsg{kind: kind, projectID: pid, itemID: it.ID, loaded: d.loaded, sess: d.sess}
		c := &confirmScreen{question: "Использовать сохранённый текст? y/N\n" + d.sess.Path,
			yes: func() tea.Cmd { return finishEdit(s, msg) },
			// отказ — файл удаляется, правка как в первый раз
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

// finishEdit читает результат правки и отправляет в Plane. Файл удаляется при
// успехе, отмене и ошибке редактора; при ошибке отправки остаётся (editFailMsg).
func finishEdit(s *session, m editDoneMsg) tea.Cmd {
	if m.err != nil {
		m.sess.Close()
		return errCmd(errors.New("редактор: " + m.err.Error()))
	}
	md, err := m.sess.Result()
	if err != nil {
		m.sess.Close()
		if errors.Is(err, text.ErrEmptyEdit) {
			s.setStatus("пустой текст — отменено", false)
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
			return tea.BatchMsg{func() tea.Msg { return statusMsg{"комментарий добавлен", false} },
				func() tea.Msg { return loadComments(s, m.projectID, m.itemID)() }}
		}
		cur, err := a.Item(ctx, m.projectID, m.itemID)
		if err != nil {
			return fail(err)
		}
		save := func() tea.Cmd { return saveDescription(a, m, md, fail) }
		if cur.UpdatedAt.After(m.loaded) {
			ask := pushScreen(&confirmScreen{question: "Описание изменилось в Plane после загрузки, перезаписать?",
				yes: save,
				// отказ — текст не теряется: сессия в черновики
				no: func() tea.Cmd { return func() tea.Msg { return fail(errors.New("отменено")) } }})
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
			func() tea.Msg { return statusMsg{"описание обновлено", false} }}
	}
}

func errCmd(err error) tea.Cmd { return func() tea.Msg { return errStatus(err) } }

func init() {
	registerAction(action{ID: "item.edit", Title: "Описание в $EDITOR", Group: "задача", Keys: []string{"e"}, Hint: "описание", Scopes: cardScopes,
		Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
			return startEdit(m.s, "desc", it, text.HTMLToMarkdown(it.DescHTML), "")
		})})
	registerAction(action{ID: "item.comment", Title: "Комментарий в $EDITOR", Group: "задача", Keys: []string{"c"}, Hint: "коммент", Scopes: cardScopes,
		Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
			return startEdit(m.s, "comment", it, "", "Комментарий к "+itoaKey(m.s, it.Sequence)+" (Markdown); пустой — отмена")
		})})
}
