package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	doubleClick = 400 * time.Millisecond // два клика по той же строке
	wheelStep   = 3                      // строк за один шаг колёсика
)

// rowClickMsg — клик по строке таблицы (row — индекс в видимом списке); open — двойной клик.
type rowClickMsg struct {
	row  int
	open bool
}

type clickMark struct {
	at time.Time
	id string
}

type zone int

const (
	zoneNone zone = iota
	zoneTabs
	zoneTable
	zonePreview
)

// bodyTop — первая строка тела главного экрана (после табов и строки поиска).
func (s *session) bodyTop() int {
	if s.searchRow() != "" {
		return 2
	}
	return 1
}

// hit определяет область под точкой: для табов — индекс таба, для таблицы — номер строки данных.
func (s *session) hit(x, y int) (z zone, idx int) {
	if y == 0 {
		for i, sp := range s.tabSpans(s.width) {
			if x >= sp[0] && x < sp[1] {
				return zoneTabs, i
			}
		}
		return zoneNone, 0
	}
	top := s.bodyTop()
	if y < top || y >= s.height-2 {
		return zoneNone, 0
	}
	lw, _ := s.paneWidths()
	switch {
	case x > lw || lw == 0:
		return zonePreview, 0
	case x == lw:
		return zoneNone, 0
	}
	return zoneTable, y - top - 1 // строка 0 тела — заголовок таблицы
}

// tabSpans — границы [x0, x1) показанных табов (по индексу таба; невидимые — {0, 0}); из той же раскладки, что tabsLine.
func (s *session) tabSpans(width int) [][2]int {
	l := s.tabsLayout(width)
	out := make([][2]int, len(l.cells))
	x := lipglossWidth(l.prefix)
	for i := l.lo; i <= l.hi && i < len(l.cells); i++ {
		w := lipglossWidth(l.cells[i])
		out[i] = [2]int{x, x + w}
		x += w + 1
	}
	return out
}

// mouse обрабатывает tea.MouseMsg главного экрана: клик по табу/строке/панели, колёсико.
func (ms *mainScreen) mouse(s *session, m tea.MouseMsg) tea.Cmd {
	z, idx := s.hit(m.X, m.Y)
	if m.Action != tea.MouseActionPress {
		return nil
	}
	switch m.Button {
	case tea.MouseButtonLeft:
		return ms.click(s, z, idx)
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		k := tea.KeyMsg{Type: tea.KeyDown}
		if m.Button == tea.MouseButtonWheelUp {
			k.Type = tea.KeyUp
		}
		var cmds []tea.Cmd
		for range wheelStep {
			switch z {
			case zoneTable:
				var c tea.Cmd
				ms.table, c = ms.table.Update(s, k)
				cmds = append(cmds, c)
			case zonePreview:
				if ms.preview != nil {
					var c tea.Cmd
					ms.preview, c = ms.preview.Update(s, k)
					cmds = append(cmds, c)
				}
			}
		}
		return tea.Batch(cmds...)
	}
	return nil
}

func (ms *mainScreen) click(s *session, z zone, idx int) tea.Cmd {
	switch z {
	case zoneTabs:
		s.setTab(idx)
	case zonePreview:
		s.focus = 1
	case zoneTable:
		s.focus = 0
		return ms.clickRow(s, idx)
	}
	return nil
}

// clickRow: курсор на строку; второй клик ≤ 400 мс по той же задаче — открыть карточку.
func (ms *mainScreen) clickRow(s *session, row int) tea.Cmd {
	t, ok := ms.table.(*table)
	if !ok {
		return nil
	}
	vis := s.visible()
	i := t.list.top + row
	if row < 0 || i >= len(vis) {
		return nil
	}
	now, id := s.deps.Now(), vis[i].ID
	open := ms.last.id == id && now.Sub(ms.last.at) <= doubleClick
	ms.last = clickMark{now, id}
	if open {
		ms.last = clickMark{}
	}
	var c tea.Cmd
	ms.table, c = ms.table.Update(s, rowClickMsg{row: i, open: open})
	return c
}
