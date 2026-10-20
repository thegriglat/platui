package tui

import (
	"context"
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/search"
)

func init() {
	registerAction(action{ID: "palette.open", Title: "Command palette", Group: "search", Keys: []string{"ctrl+p"},
		Hint: "Ctrl+P|palette", Scopes: []string{"main", "kanban", "item", "projects"},
		Run: func(m *model) tea.Cmd { return pushScreen(newPalette(m.s, m.top().Scope())) }})
}

const paletteMaxQuery = 64

// palEntry is a palette row; run is executed in the palette Update (synchronously) and returns a command
// that runs after the palette closes.
type palEntry struct {
	kind, title, keys string
	score             int
	pos               []int
	run               func() tea.Cmd
}

type paletteScreen struct {
	under   string
	query   string
	entries []palEntry
	cur     int
}

func newPalette(s *session, under string) *paletteScreen {
	p := &paletteScreen{under: under}
	p.refresh(s)
	return p
}

func (p *paletteScreen) Title() string       { return "palette" }
func (p *paletteScreen) Scope() string       { return "palette" }
func (p *paletteScreen) CapturesInput() bool { return true }

// keyMsgFor is the inverse of keyName: key name → tea.KeyMsg.
func keyMsgFor(name string) (tea.KeyMsg, bool) {
	named := map[string]tea.KeyType{"up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight,
		"esc": tea.KeyEsc, "enter": tea.KeyEnter, "tab": tea.KeyTab, "shift+tab": tea.KeyShiftTab, "pgup": tea.KeyPgUp,
		"pgdown": tea.KeyPgDown, "home": tea.KeyHome, "end": tea.KeyEnd, "ctrl+left": tea.KeyCtrlLeft,
		"ctrl+right": tea.KeyCtrlRight, "shift+left": tea.KeyShiftLeft, "shift+right": tea.KeyShiftRight,
		"shift+up": tea.KeyShiftUp, "shift+down": tea.KeyShiftDown}
	if t, ok := named[name]; ok {
		return tea.KeyMsg{Type: t}, true
	}
	if name == "space" {
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}, true
	}
	if r := []rune(name); len(r) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: r}, true
	}
	if r := []rune(strings.TrimPrefix(name, "alt+")); strings.HasPrefix(name, "alt+") && len(r) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: r, Alt: true}, true
	}
	if r := []rune(strings.TrimPrefix(name, "ctrl+")); strings.HasPrefix(name, "ctrl+") && len(r) == 1 && r[0] >= 'a' && r[0] <= 'z' {
		return tea.KeyMsg{Type: tea.KeyCtrlA + tea.KeyType(r[0]-'a')}, true
	}
	return tea.KeyMsg{}, false
}

var paletteSkip = []string{"palette.open", "app.quit", "nav.move", "nav.tab.n", "nav.focus", "nav.back"}

// baseEntries returns all candidates without a query.
func (p *paletteScreen) baseEntries(s *session) []palEntry {
	closeThen := func(c tea.Cmd) tea.Cmd { return tea.Sequence(popScreen, c) }
	var out []palEntry
	marked := len(s.marks) > 0
	acts := actionsFor(p.under)
	for _, a := range acts {
		if len(a.Keys) == 0 || a.Run == nil || slices.Contains(paletteSkip, a.ID) {
			continue
		}
		if a.Bulk != marked {
			twin := slices.ContainsFunc(acts, func(b action) bool {
				return b.Bulk == marked && slices.ContainsFunc(b.Keys, func(k string) bool { return slices.Contains(a.Keys, k) })
			})
			if twin {
				continue
			}
		}
		km, ok := keyMsgFor(a.Keys[0])
		title := a.Title
		if a.Bulk {
			title += " (marked)"
		}
		out = append(out, palEntry{kind: "action", title: title, keys: keysLabel(a.Keys), run: func() tea.Cmd {
			if !ok {
				s.setStatus("action unavailable from the palette", true)
				return popScreen
			}
			return closeThen(func() tea.Msg { return km })
		}})
	}
	out = append(out, customEntries(s, p.under, closeThen)...)
	if marked {
		out = append(out, palEntry{kind: "action", title: "Clear all marks", run: func() tea.Cmd {
			s.marks = map[string]bool{}
			return popScreen
		}})
	}
	for i, t := range s.tabList() {
		out = append(out, palEntry{kind: "tab", title: t.label, run: func() tea.Cmd { s.setTab(i); return popScreen }})
	}
	if s.filter != 0 {
		out = append(out, palEntry{kind: "filter", title: "No filter", run: func() tea.Cmd { s.filter = 0; return popScreen }})
	}
	for i, f := range s.deps.Config.Filters {
		out = append(out, palEntry{kind: "filter", title: f.Name, run: func() tea.Cmd { s.filter = i + 1; return popScreen }})
	}
	for _, pr := range s.projects {
		out = append(out, palEntry{kind: "project", title: pr.Identifier + " " + pr.Name, run: func() tea.Cmd {
			return closeThen(func() tea.Msg { return openProjectMsg{project: pr} })
		}})
	}
	return out
}

func (p *paletteScreen) itemEntries(s *session) []palEntry {
	var out []palEntry
	for _, it := range s.items {
		out = append(out, palEntry{kind: "task", title: itoaKey(s, it.Sequence) + " " + it.Name, run: func() tea.Cmd {
			return p.gotoItem(s, it.ID)
		}})
	}
	return out
}

// gotoItem: cursor to the task (clearing filters in the way) and open the card.
func (p *paletteScreen) gotoItem(s *session, id string) tea.Cmd {
	it, ok := s.itemByID(id)
	if !ok {
		return popScreen
	}
	if tl := s.tabList(); s.tab >= len(tl) || !s.inTab(it, tl[s.tab]) {
		if i := slices.IndexFunc(tl, func(t tabInfo) bool { return s.inTab(it, t) }); i >= 0 {
			s.setTab(i)
		}
	}
	if !s.visibleHas(id) {
		s.setQuery("")
		s.mine, s.filter, s.searching, s.searchLine = false, 0, false, ""
	}
	s.cursor = id
	return tea.Sequence(popScreen, openItem(s, it))
}

func (s *session) visibleHas(id string) bool {
	for _, it := range s.visible() {
		if it.ID == id {
			return true
		}
	}
	return false
}

// keyEntry is the jump by key PROJ-N / N.
func (p *paletteScreen) keyEntry(s *session) (palEntry, bool) {
	ident, seq, ok := search.ParseKey(p.query)
	if !ok {
		return palEntry{}, false
	}
	if ident == "" && s.project != nil {
		ident = strings.ToUpper(s.project.Identifier)
	}
	if ident == "" {
		return palEntry{}, false
	}
	key := fmt.Sprintf("%s-%d", ident, seq)
	return palEntry{kind: "jump", title: "Open " + key, score: 1 << 20, run: func() tea.Cmd {
		if s.project != nil && strings.EqualFold(s.project.Identifier, ident) && s.index != nil {
			if id, ok := s.index.Lookup(ident, seq); ok {
				return p.gotoItem(s, id)
			}
		}
		a := s.deps.App
		return tea.Sequence(popScreen, func() tea.Msg {
			pr, it, err := a.Find(context.Background(), ident, seq)
			if err != nil {
				return statusMsg{key + " not found", true}
			}
			return openProjectMsg{project: pr, itemID: it.ID}
		})
	}}, true
}

func (p *paletteScreen) refresh(s *session) {
	base := p.baseEntries(s)
	if p.query == "" {
		p.entries, p.cur = base, 0
		return
	}
	base = append(base, p.itemEntries(s)...)
	var out []palEntry
	for _, e := range base {
		if sc, pos, ok := search.Fuzzy(p.query, e.title); ok {
			e.score, e.pos = sc, pos
			out = append(out, e)
		}
	}
	slices.SortStableFunc(out, func(a, b palEntry) int { return b.score - a.score })
	if ke, ok := p.keyEntry(s); ok {
		out = append([]palEntry{ke}, out...)
	}
	p.entries, p.cur = out, 0
}

func (p *paletteScreen) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return p, nil
	}
	switch name := keyName(k); name {
	case "esc":
		return p, popScreen
	case "up":
		p.cur = max(p.cur-1, 0)
	case "down":
		p.cur = max(min(p.cur+1, len(p.entries)-1), 0)
	case "pgup":
		p.cur = max(p.cur-8, 0)
	case "pgdown":
		p.cur = max(min(p.cur+8, len(p.entries)-1), 0)
	case "home":
		p.cur = 0
	case "end":
		p.cur = max(len(p.entries)-1, 0)
	case "enter":
		if p.cur < len(p.entries) {
			return p, p.entries[p.cur].run()
		}
	case "backspace":
		if r := []rune(p.query); len(r) > 0 {
			p.query = string(r[:len(r)-1])
			p.refresh(s)
		}
	case "space":
		p.add(s, " ")
	default:
		if k.Type == tea.KeyRunes && !k.Alt {
			p.add(s, string(k.Runes))
		}
	}
	return p, nil
}

func (p *paletteScreen) add(s *session, t string) {
	if len([]rune(p.query+t)) > paletteMaxQuery {
		return
	}
	p.query += t
	p.refresh(s)
}

func (p *paletteScreen) View(s *session) string {
	h := max(s.height-3, 1)
	head := s.st.accent.Bold(true).Render(" Palette: ") + p.query + "▏"
	from := max(0, min(p.cur-h/2, len(p.entries)-h))
	var rows []string
	for i := from; i < len(p.entries) && i < from+h; i++ {
		e := p.entries[i]
		title := e.title
		if len(e.pos) > 0 && e.kind != "jump" {
			title = search.Highlight(e.title, e.pos, func(x string) string { return s.st.accent.Bold(true).Render(x) })
		}
		line := " " + s.st.muted.Render(fmt.Sprintf("%-8s", e.kind)) + " " + title
		if e.keys != "" {
			line += "  " + s.st.muted.Render(e.keys)
		}
		if i == p.cur {
			line = s.st.sel.Render(clip(" "+e.kind+" "+e.title+"  "+e.keys, s.width))
		} else {
			line = clip(line, s.width)
		}
		rows = append(rows, line)
	}
	if len(rows) == 0 {
		rows = append(rows, s.st.muted.Render(" nothing found"))
	}
	foot := s.st.muted.Render(" ↑↓ select   Enter run   Esc close")
	return head + "\n" + strings.Join(rows, "\n") + "\n" + foot
}
