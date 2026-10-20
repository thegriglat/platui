package tui

import (
	"context"
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/search"
)

func init() {
	registerAction(action{ID: "search.start", Title: "Поиск по заголовку", Group: "поиск", Keys: []string{"/"},
		Hint: "поиск", Scopes: []string{"main"}, Run: func(m *model) tea.Cmd {
			m.s.searching = true
			return pushScreen(&searchScreen{under: m.top(), text: []rune(m.s.query)})
		}})
	registerAction(action{ID: "search.goto", Title: "Перейти по ID", Group: "поиск", Keys: []string{"ctrl+g"},
		Hint: "Ctrl+G|по ID", Scopes: []string{"main", "kanban", "item"}, Run: func(m *model) tea.Cmd {
			return pushScreen(&searchScreen{under: m.top(), gotoMode: true})
		}})
}

// searchScreen — строка ввода поиска (`/`) или перехода по ключу (Ctrl+G) поверх экрана under.
type searchScreen struct {
	under    screen
	gotoMode bool
	text     []rune
}

func (c *searchScreen) Title() string       { return "поиск" }
func (c *searchScreen) Scope() string       { return "dialog" }
func (c *searchScreen) CapturesInput() bool { return true }

func (c *searchScreen) isKey() (ident string, seq int, ok bool) {
	return search.ParseKey(string(c.text))
}

func (c *searchScreen) line(s *session) string {
	if c.gotoMode {
		return "Перейти к: " + string(c.text) + "▏  (Enter — открыть, Esc — отмена)"
	}
	if id, n, ok := c.isKey(); ok {
		return "/ " + string(c.text) + "▏  Enter — открыть " + keyText(s, id, n)
	}
	return "/ " + string(c.text) + "▏  найдено " + itoa(len(s.visible()))
}

func keyText(s *session, ident string, seq int) string {
	if ident == "" && s.project != nil {
		ident = s.project.Identifier
	}
	return ident + "-" + itoa(seq)
}

func (c *searchScreen) View(s *session) string {
	s.searchLine = c.line(s)
	defer func() { s.searchLine = "" }()
	return c.under.View(s)
}

func (c *searchScreen) close(s *session) tea.Cmd {
	s.searching = false
	return popScreen
}

func (c *searchScreen) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); !ok {
		// остальные сообщения model уже раздаёт всему стеку, включая under
		return c, nil
	}
	k := msg.(tea.KeyMsg)
	switch name := keyName(k); name {
	case "esc":
		if !c.gotoMode {
			s.setQuery("")
			s.cursor = firstID(s)
		}
		return c, c.close(s)
	case "enter":
		if id, n, ok := c.isKey(); ok {
			if !c.gotoMode {
				s.setQuery("")
			}
			return c, tea.Sequence(c.close(s), gotoKey(s, id, n))
		}
		if c.gotoMode {
			s.setStatus("введите ID задачи: PROJ-42 или 42", true)
			return c, nil
		}
		return c, c.close(s)
	case "backspace":
		if len(c.text) > 0 {
			c.text = c.text[:len(c.text)-1]
			c.changed(s)
		}
	case "up", "down", "pgup", "pgdown", "home", "end":
		if !c.gotoMode {
			var cmd tea.Cmd
			c.under, cmd = c.under.Update(s, msg)
			return c, cmd
		}
	default:
		if k.Type == tea.KeyRunes || k.Type == tea.KeySpace {
			r := k.Runes
			if k.Type == tea.KeySpace {
				r = []rune{' '}
			}
			if room := search.MaxQuery - len(c.text); room > 0 {
				c.text = append(c.text, r[:min(room, len(r))]...)
				c.changed(s)
			}
		}
	}
	return c, nil
}

func (c *searchScreen) changed(s *session) {
	if c.gotoMode {
		return
	}
	s.setQuery(string(c.text))
	s.cursor = firstID(s)
}

func firstID(s *session) string {
	if v := s.visible(); len(v) > 0 {
		return v[0].ID
	}
	return ""
}

// foundMsg — итог перехода по ключу.
type foundMsg struct {
	ident string
	seq   int
	p     plane.Project
	it    plane.WorkItem
	err   error
}

// gotoKey: задача текущего проекта — по индексу, иначе app.Find (любой проект workspace).
func gotoKey(s *session, ident string, seq int) tea.Cmd {
	if s.project == nil {
		return func() tea.Msg { return statusMsg{"проект не выбран", true} }
	}
	if ident == "" {
		ident = s.project.Identifier
	}
	if strings.EqualFold(ident, s.project.Identifier) && s.index != nil {
		if id, ok := s.index.Lookup(ident, seq); ok {
			if it, ok := s.itemByID(id); ok {
				s.cursor = it.ID
				s.revealTab(it)
				return openItem(s, it)
			}
		}
	}
	a := s.deps.App
	return func() tea.Msg {
		p, it, err := a.Find(context.Background(), ident, seq)
		return foundMsg{ident: ident, seq: seq, p: p, it: it, err: err}
	}
}

// revealTab переключает таб на состояние задачи, если она не видна в текущем.
func (s *session) revealTab(it plane.WorkItem) {
	tl := s.tabList()
	if s.tab < len(tl) && s.inTab(it, tl[s.tab]) {
		return
	}
	for i, t := range tl {
		if s.inTab(it, t) {
			s.tab = i
			return
		}
	}
}

func (s *session) onFound(f foundMsg) tea.Cmd {
	if f.err != nil {
		if errors.Is(f.err, app.ErrItemNotFound) || errors.Is(f.err, app.ErrProjectNotFound) {
			s.setStatus(f.ident+"-"+itoa(f.seq)+" не найдена", true)
		} else {
			s.setStatus(f.err.Error(), true)
		}
		return nil
	}
	if s.project != nil && s.project.ID == f.p.ID {
		s.mergeItem(f.it)
		s.cursor = f.it.ID
		s.revealTab(f.it)
		return openItem(s, f.it)
	}
	return func() tea.Msg { return openProjectMsg{project: f.p, itemID: f.it.ID} }
}

func (s *session) mergeItem(it plane.WorkItem) {
	if _, ok := s.itemByID(it.ID); !ok {
		s.items = append(append([]plane.WorkItem(nil), s.items...), it)
		s.rebuildIndex()
	}
}
