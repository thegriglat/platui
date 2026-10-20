package tui

import (
	"context"
	"errors"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/todo"
)

const todoChangedStatus = "todo.md changed outside platui — reloaded"

// todoScreen is the personal todo screen (C4 v7); every change is saved immediately.
type todoScreen struct {
	path string
	list *todo.List
	cur  cursorList
}

// todoTaskMsg is the result of app.CreateQuick for an item.
type todoTaskMsg struct {
	owner *todoScreen
	text  string
	item  plane.WorkItem
	err   error
}

func newTodoScreen(path string, l *todo.List) *todoScreen {
	t := &todoScreen{path: path, list: l}
	t.cur.SetLen(len(l.Items()))
	return t
}

func (t *todoScreen) Title() string       { return "todo" }
func (t *todoScreen) Scope() string       { return "todo" }
func (t *todoScreen) CapturesInput() bool { return false }

// selected is the item under the cursor.
func (t *todoScreen) selected() (todo.Item, bool) {
	its := t.list.Items()
	if t.cur.Pos() >= len(its) {
		return todo.Item{}, false
	}
	return its[t.cur.Pos()], true
}

// apply applies a change and saves; ErrChanged — re-reads the file, the change is not applied.
func (t *todoScreen) apply(s *session, change func(l *todo.List) error) bool {
	err := change(t.list)
	if err == nil {
		err = t.list.Save()
	}
	switch {
	case err == nil:
		t.cur.SetLen(len(t.list.Items()))
		return true
	case errors.Is(err, todo.ErrChanged):
		t.reload(s)
		s.setStatus(todoChangedStatus, true)
	default:
		s.setStatus("todo: "+err.Error(), true)
	}
	return false
}

func (t *todoScreen) reload(s *session) {
	l, err := todo.Load(t.path)
	if err != nil {
		s.setStatus("todo: "+err.Error(), true)
		return
	}
	t.list = l
	t.cur.SetLen(len(l.Items()))
}

func (t *todoScreen) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	if m, ok := msg.(todoTaskMsg); ok {
		return t, t.onTask(s, m)
	}
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return t, nil
	}
	n := keyName(k)
	it, has := t.selected()
	switch n {
	case "n":
		return t, pushScreen(t.input("New item", "", func(text string) {
			if t.apply(s, func(l *todo.List) error { l.Add(text); return nil }) {
				t.cur.Set(len(t.list.Items()) - 1)
			}
		}))
	case "e":
		if has {
			return t, pushScreen(t.input("Edit item", it.Text, func(text string) {
				t.apply(s, func(l *todo.List) error { return l.Edit(it.N, text) })
			}))
		}
	case "space":
		if has {
			t.apply(s, func(l *todo.List) error { return l.SetDone(it.N, !it.Done) })
		}
	case "d":
		if has {
			return t, confirmWith("Delete item \""+it.Text+"\"?", func() tea.Cmd {
				t.apply(s, func(l *todo.List) error { return l.Remove(it.N) })
				return nil
			})
		}
	case "shift+up", "shift+down":
		if has {
			to := it.N - 1
			if n == "shift+down" {
				to = it.N + 1
			}
			if to >= 1 && to <= len(t.list.Items()) && t.apply(s, func(l *todo.List) error { return l.Move(it.N, to) }) {
				t.cur.Set(to - 1)
			}
		}
	case "C":
		if has {
			return t, t.makeTask(s, it)
		}
	case "enter":
		if has {
			return t, t.gotoKey(s, it)
		}
	default:
		t.cur.Key(n)
	}
	return t, nil
}

// input is the input line; empty text is rejected.
func (t *todoScreen) input(title, initial string, done func(text string)) *inputScreen {
	c := &inputScreen{title: title, hint: "Enter save   Esc cancel", text: []rune(initial), simCur: -1}
	c.onSubmit = func(c *inputScreen) tea.Cmd {
		text := strings.TrimSpace(string(c.text))
		if text == "" {
			return nil
		}
		done(text)
		return popScreen
	}
	return c
}

func (t *todoScreen) makeTask(s *session, it todo.Item) tea.Cmd {
	switch {
	case s.project == nil:
		s.setStatus("no project selected", true)
		return nil
	case len(todo.Keys(it.Text)) > 0:
		s.setStatus("item already links to "+todo.Keys(it.Text)[0], true)
		return nil
	}
	a, pid := s.deps.App, s.project.ID
	s.setStatus("creating task…", false)
	return func() tea.Msg {
		task, err := a.CreateQuick(context.Background(), pid, it.Text, "")
		return todoTaskMsg{owner: t, text: it.Text, item: task, err: err}
	}
}

func (t *todoScreen) onTask(s *session, m todoTaskMsg) tea.Cmd {
	if m.owner != t {
		return nil
	}
	if m.err != nil {
		s.setStatus(m.err.Error(), true)
		return nil
	}
	if s.project != nil && s.project.ID == m.item.ProjectID {
		s.mergeItem(m.item)
	}
	key := itoaKey(s, m.item.Sequence)
	// the item is looked up by text; a file change during the request is detected by Save (ErrChanged)
	n := -1
	for _, it := range t.list.Items() {
		if it.Text == m.text {
			n = it.N
			break
		}
	}
	if n < 0 {
		s.setStatus("created "+key+", item not found in todo", true)
		return nil
	}
	if t.apply(s, func(l *todo.List) error { return l.Edit(n, m.text+" ("+key+")") }) {
		s.setStatus("created "+key+": "+m.item.Name, false)
	}
	return nil
}

// gotoKey jumps to the first task in the item.
func (t *todoScreen) gotoKey(s *session, it todo.Item) tea.Cmd {
	keys := todo.Keys(it.Text)
	if len(keys) == 0 {
		s.setStatus("no task link in the item", false)
		return nil
	}
	i := strings.LastIndex(keys[0], "-")
	seq, _ := strconv.Atoi(keys[0][i+1:])
	return gotoKey(s, keys[0][:i], seq)
}

func (t *todoScreen) View(s *session) string {
	rows := []string{s.st.accent.Bold(true).Render(" Todo  " + t.path)}
	its := t.list.Items()
	t.cur.SetHeight(max(s.height-3, 1))
	if len(its) == 0 {
		rows = append(rows, s.st.muted.Render("  empty — n: new item"))
	}
	from, to := t.cur.Range()
	for i := from; i < to; i++ {
		box := "[ ] "
		if its[i].Done {
			box = "[x] "
		}
		line := "  " + box + its[i].Text
		switch {
		case i == t.cur.Pos():
			line = s.st.sel.Render(clip("> "+box+its[i].Text, s.width))
		case its[i].Done:
			line = s.st.muted.Render(clip(line, s.width))
		default:
			line = clip(line, s.width)
		}
		rows = append(rows, line)
	}
	for len(rows) < s.height-2 {
		rows = append(rows, "")
	}
	return clip2(append(rows, hintsLine(s, "todo", s.width), s.statusView(s.width)), s.height)
}
