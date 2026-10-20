package tui

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
)

// pickRow is a picker list row; Hot is the quick-select key (optional).
type pickRow struct{ Label, Hot string }

// listScreen is a modal picker list: ↑/↓, Enter, Esc, quick-select keys.
type listScreen struct {
	title    string
	rows     []pickRow
	list     cursorList
	onSelect func(i int) tea.Cmd // called after the list closes
}

func newListScreen(title string, rows []pickRow, cur int, onSelect func(int) tea.Cmd) *listScreen {
	l := &listScreen{title: title, rows: rows, onSelect: onSelect}
	l.list.SetLen(len(rows))
	l.list.Set(cur)
	return l
}

func (l *listScreen) Title() string       { return l.title }
func (l *listScreen) Scope() string       { return "dialog" }
func (l *listScreen) CapturesInput() bool { return false }

func (l *listScreen) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return l, nil
	}
	name := keyName(k)
	l.list.SetHeight(max(s.height-3, 1))
	if l.list.Key(name) {
		return l, nil
	}
	switch name {
	case "esc":
		return l, popScreen
	case "enter":
		if len(l.rows) > 0 {
			return l, l.choose(l.list.Pos())
		}
	default:
		if i := slices.IndexFunc(l.rows, func(r pickRow) bool { return r.Hot != "" && r.Hot == name }); i >= 0 {
			return l, l.choose(i)
		}
	}
	return l, nil
}

func (l *listScreen) choose(i int) tea.Cmd {
	return tea.Sequence(popScreen, l.onSelect(i))
}

func (l *listScreen) View(s *session) string {
	h := max(s.height-3, 1)
	l.list.SetHeight(h)
	out := []string{s.st.accent.Bold(true).Render(" " + l.title)}
	from, to := l.list.Range()
	for i := from; i < to; i++ {
		line := "  " + l.rows[i].Label
		if h := l.rows[i].Hot; h != "" {
			line = "  " + s.st.bold.Render(h) + " " + l.rows[i].Label
		}
		if i == l.list.Pos() {
			line = s.st.sel.Render(clip(line, s.width))
		}
		out = append(out, clip(line, s.width))
	}
	for len(out) < h+1 {
		out = append(out, "")
	}
	out = append(out, s.st.muted.Render(" ↑↓ select   Enter choose   Esc cancel"), s.statusView(s.width))
	return strings.Join(out, "\n")
}

// pickLater — the onPick callback may return a command (e.g. patchItem): the picker returns it
// after the list closes. Called only from onPick (Update is single-threaded).
var queuedCmd []tea.Cmd

func pickLater(c tea.Cmd) { queuedCmd = append(queuedCmd, c) }

func runPick(call func()) tea.Cmd {
	queuedCmd = nil
	call()
	c := tea.Batch(queuedCmd...)
	queuedCmd = nil
	return c
}

// priorities are the Plane priorities in descending order.
var priorities = []string{"urgent", "high", "medium", "low", "none"}

func init() {
	// state order is the same as the tabs (app.StateTabs), then hidden ones.
	pickState = func(s *session, onPick func(plane.State)) screen {
		states := slices.Clone(s.tabs)
		for _, st := range s.meta.States {
			if !slices.ContainsFunc(states, func(x plane.State) bool { return x.ID == st.ID }) {
				states = append(states, st)
			}
		}
		rows := make([]pickRow, len(states))
		for i, st := range states {
			rows[i] = pickRow{Label: st.Name + s.st.muted.Render("  "+st.Group)}
		}
		return newListScreen("State", rows, 0, func(i int) tea.Cmd {
			return runPick(func() { onPick(states[i]) })
		})
	}
	pickPriority = func(s *session, onPick func(string)) screen {
		rows := make([]pickRow, len(priorities))
		for i, p := range priorities {
			rows[i] = pickRow{Label: s.st.priority(p).Render(p)}
		}
		return newListScreen("Priority", rows, 0, func(i int) tea.Cmd {
			return runPick(func() { onPick(priorities[i]) })
		})
	}
	pickLabel = func(s *session, onPick func(plane.Label)) screen {
		labels := s.meta.Labels
		rows := make([]pickRow, len(labels))
		for i, l := range labels {
			rows[i] = pickRow{Label: l.Name}
		}
		return newListScreen("Label", rows, 0, func(i int) tea.Cmd {
			return runPick(func() { onPick(labels[i]) })
		})
	}
}
