package tui

import (
	"context"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
)

const kanbanMinCol = 22 // минимальная ширина колонки; уже — горизонтальная прокрутка

// kanban — вид «канбан»: колонки-состояния (без «Все»), задачи под курсором.
type kanban struct {
	sel      string // ID выбранной задачи
	col, row int
	off      int // первая видимая колонка
}

func init() {
	newKanban = func(s *session) screen {
		k := &kanban{sel: s.cursor}
		if tl := s.tabList(); k.sel == "" && s.tab < len(tl) && tl[s.tab].state != nil {
			k.col = slices.IndexFunc(s.tabs, func(st plane.State) bool { return st.ID == tl[s.tab].state.ID })
		}
		k.sync(s)
		return k
	}
	kb := []string{"kanban"}
	for _, a := range []action{
		{ID: "kanban.nav", Title: "Канбан: колонка, задача", Group: "канбан", Scopes: kb,
			Keys: []string{"left", "right", "up", "down", "pgup", "pgdown", "home", "end"}, Hint: "←→↑↓|ход",
			Run: func(m *model) tea.Cmd { return m.forward() }},
		{ID: "kanban.move.left", Title: "Перенос в предыдущее состояние", Group: "канбан", Scopes: kb,
			Keys: []string{"shift+left"}, Hint: "Shift+←→|перенос", Run: func(m *model) tea.Cmd { return m.forward() }},
		{ID: "kanban.move.right", Title: "Перенос в следующее состояние", Group: "канбан", Scopes: kb,
			Keys: []string{"shift+right"}, Run: func(m *model) tea.Cmd { return m.forward() }},
		{ID: "kanban.open", Title: "Открыть карточку", Group: "канбан", Scopes: kb,
			Keys: []string{"enter"}, Hint: "карточка", Run: func(m *model) tea.Cmd { return m.forward() }},
		{ID: "kanban.back", Title: "Назад к таблице", Group: "канбан", Scopes: kb,
			Keys: []string{"v"}, Hint: "таблица", Run: func(m *model) tea.Cmd { m.pop(); return nil }},
	} {
		registerAction(a)
	}
}

func (k *kanban) Title() string       { return "канбан" }
func (k *kanban) Scope() string       { return "kanban" }
func (k *kanban) CapturesInput() bool { return false }

// column — задачи колонки i с учётом фильтров и сортировки.
func (k *kanban) column(s *session, i int) []plane.WorkItem {
	if i < 0 || i >= len(s.tabs) {
		return nil
	}
	var out []plane.WorkItem
	for _, it := range s.items {
		if it.StateID == s.tabs[i].ID && s.pass(it) {
			out = append(out, it)
		}
	}
	if s.hits != nil {
		slices.SortStableFunc(out, func(a, b plane.WorkItem) int { return s.hits[b.ID].Score - s.hits[a.ID].Score })
	} else {
		s.sortItems(out)
	}
	return out
}

// sync согласует позицию с данными: выбранная задача ищется по ID, иначе позиция зажимается.
func (k *kanban) sync(s *session) {
	if len(s.tabs) == 0 {
		k.col, k.row, k.sel = 0, 0, ""
		return
	}
	if k.sel != "" {
		for c := range s.tabs {
			if i := slices.IndexFunc(k.column(s, c), func(it plane.WorkItem) bool { return it.ID == k.sel }); i >= 0 {
				k.col, k.row = c, i
				s.cursor = k.sel
				return
			}
		}
	}
	k.col = max(0, min(k.col, len(s.tabs)-1))
	col := k.column(s, k.col)
	k.row = max(0, min(k.row, len(col)-1))
	k.sel = ""
	if len(col) > 0 {
		k.sel = col[k.row].ID
		s.cursor = k.sel
	}
}

func (k *kanban) selectAt(s *session, col, row int) {
	k.col = max(0, min(col, len(s.tabs)-1))
	k.row = max(row, 0)
	k.sel = ""
	k.sync(s)
}

// kanbanMovedMsg — итог неудачного переноса (успех приходит как itemMsg).
type kanbanMovedMsg struct {
	id, from, to string
	err          error
}

func moveItem(s *session, id, from, to string) tea.Cmd {
	if s.project == nil {
		return nil
	}
	a, pid := s.deps.App, s.project.ID
	return func() tea.Msg {
		it, err := a.Update(context.Background(), pid, id, plane.WorkItemPatch{StateID: &to})
		if err != nil {
			return kanbanMovedMsg{id: id, from: from, to: to, err: err}
		}
		return itemMsg{item: it}
	}
}

// setItemState меняет состояние задачи в сессии (копия среза).
func (s *session) setItemState(id, state string) {
	if i := slices.IndexFunc(s.items, func(x plane.WorkItem) bool { return x.ID == id }); i >= 0 {
		s.items = slices.Clone(s.items)
		s.items[i].StateID = state
	}
}

func (k *kanban) move(s *session, d int) tea.Cmd {
	it, ok := s.itemByID(k.sel)
	to := k.col + d
	if !ok || to < 0 || to >= len(s.tabs) {
		return nil
	}
	run := func() tea.Cmd {
		from := it.StateID
		s.setItemState(it.ID, s.tabs[to].ID)
		s.setStatus(itoaKey(s, it.Sequence)+" → "+s.tabs[to].Name, false)
		k.sync(s)
		return moveItem(s, it.ID, from, s.tabs[to].ID)
	}
	if destructiveState(s.tabs[to]) {
		return confirmWith("Перевести "+itoaKey(s, it.Sequence)+" в "+s.tabs[to].Name+"?", run)
	}
	return run()
}

func (k *kanban) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case kanbanMovedMsg:
		// откат, если задача ещё в целевом состоянии
		if it, ok := s.itemByID(msg.id); ok && it.StateID == msg.to {
			s.setItemState(msg.id, msg.from)
		}
		s.setStatus("перенос не удался: "+msg.err.Error(), true)
		k.sync(s)
		return k, nil
	case tea.KeyMsg:
		k.sync(s)
		col := k.column(s, k.col)
		page := max(s.height-5, 2)
		switch keyName(msg) {
		case "left":
			k.selectAt(s, k.col-1, k.row)
		case "right":
			k.selectAt(s, k.col+1, k.row)
		case "up":
			k.selectAt(s, k.col, k.row-1)
		case "down":
			k.selectAt(s, k.col, min(k.row+1, len(col)-1))
		case "pgup":
			k.selectAt(s, k.col, k.row-page)
		case "pgdown":
			k.selectAt(s, k.col, min(k.row+page, len(col)-1))
		case "home":
			k.selectAt(s, k.col, 0)
		case "end":
			k.selectAt(s, k.col, len(col)-1)
		case "shift+left":
			return k, k.move(s, -1)
		case "shift+right":
			return k, k.move(s, 1)
		case "enter":
			if it, ok := s.itemByID(k.sel); ok {
				return k, openItem(s, it)
			}
		}
		return k, nil
	}
	k.sync(s)
	return k, nil
}

func (k *kanban) View(s *session) string {
	k.sync(s)
	head := s.st.accent.Bold(true).Render(" канбан ")
	if s.project != nil {
		head = s.st.accent.Bold(true).Render(" "+s.project.Identifier+" ") + head
	}
	if s.mine {
		head += s.st.accent.Render("[Мои]")
	}
	rows := []string{clip(head, s.width)}
	h := max(s.height-3, 1)
	n := len(s.tabs)
	if n == 0 {
		rows = append(rows, s.st.muted.Render("нет данных"))
	} else {
		vis := n
		if s.width < n*kanbanMinCol+n-1 {
			vis = max(1, (s.width+1)/(kanbanMinCol+1))
		}
		if k.col < k.off {
			k.off = k.col
		}
		if k.col >= k.off+vis {
			k.off = k.col - vis + 1
		}
		k.off = max(0, min(k.off, n-vis))
		w := max((s.width-(vis-1))/vis, 1)
		blocks := make([][]string, vis)
		for i := range blocks {
			blocks[i] = k.renderColumn(s, k.off+i, w, h-1)
		}
		sep := s.st.border.Render("│")
		for r := 0; r < h; r++ {
			parts := make([]string, vis)
			for i := range blocks {
				parts[i] = blocks[i][r]
			}
			rows = append(rows, strings.Join(parts, sep))
		}
	}
	for len(rows) < s.height-2 {
		rows = append(rows, "")
	}
	rows = append(rows[:min(len(rows), s.height-2)], hintsLine(s, "kanban", s.width), s.statusView(s.width))
	return strings.Join(rows, "\n")
}

// renderColumn — заголовок и карточки колонки ровно height+1 строк шириной w.
func (k *kanban) renderColumn(s *session, c, w, height int) []string {
	items := k.column(s, c)
	title := " " + s.tabs[c].Name + " " + itoa(len(items)) + " "
	hs := s.st.tabOff
	if c == k.col {
		hs = s.st.tabOn
	}
	lines := []string{hs.Render(clip(title, w))}
	top := 0
	if c == k.col {
		top = max(0, k.row-height+1)
	}
	for i := top; i < min(len(items), top+height); i++ {
		it := items[i]
		mark := " "
		if s.marks[it.ID] {
			mark = "*"
		}
		line := clip(mark+itoaKey(s, it.Sequence)+" "+it.Name, w)
		line += strings.Repeat(" ", max(0, w-lipglossWidth(line)))
		switch {
		case c == k.col && it.ID == k.sel:
			line = s.st.sel.Render(line)
		case !s.fresh[it.ID].IsZero():
			line = s.st.accent.Render(line)
		}
		lines = append(lines, line)
	}
	pad := strings.Repeat(" ", w)
	for len(lines) < height+1 {
		lines = append(lines, pad)
	}
	return fitBlock(strings.Join(lines, "\n"), w, height+1)
}
