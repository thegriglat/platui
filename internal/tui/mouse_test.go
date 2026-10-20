package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/config"
)

func click(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}
}

func wheel(x, y int, up bool) tea.MouseMsg {
	b := tea.MouseButtonWheelDown
	if up {
		b = tea.MouseButtonWheelUp
	}
	return tea.MouseMsg{X: x, Y: y, Button: b, Action: tea.MouseActionPress}
}

func TestContractTUIV6Mouse(t *testing.T) {
	m, _, _, _ := newTestModel(t)
	s := m.s
	// клик по табу
	sp := s.tabSpans(s.width)
	if sp[2][1] <= sp[2][0] {
		t.Fatalf("spans=%v", sp)
	}
	feed(t, m, click(sp[2][0]+1, 0))
	if s.tab != 2 {
		t.Fatalf("tab=%d", s.tab)
	}
	// клик по строке: курсор (вид отрисован — как между событиями в программе)
	view(m)
	vis := s.visible()
	if len(vis) < 3 {
		t.Fatalf("vis=%d", len(vis))
	}
	feed(t, m, click(5, s.bodyTop()+1+2))
	if s.cursor != vis[2].ID || s.focus != 0 {
		t.Fatalf("cursor=%s want %s", s.cursor, vis[2].ID)
	}
	if s.openItem != "" {
		t.Fatal("одиночный клик открыл карточку")
	}
	// двойной клик по той же строке в пределах 400 мс — карточка
	now := t0
	s.deps.Now = func() time.Time { return now }
	feed(t, m, click(5, s.bodyTop()+1))
	now = now.Add(100 * time.Millisecond)
	feed(t, m, click(5, s.bodyTop()+1))
	if s.openItem != vis[0].ID || len(m.stack) != 2 {
		t.Fatalf("openItem=%q stack=%d", s.openItem, len(m.stack))
	}
	feed(t, m, click(5, s.bodyTop()+2)) // мышь на экране карточки игнорируется
	if len(m.stack) != 2 {
		t.Fatal("клик изменил стек")
	}
	m.pop()
	// медленный второй клик — не двойной
	feed(t, m, click(5, s.bodyTop()+2))
	now = now.Add(time.Second)
	feed(t, m, click(5, s.bodyTop()+2))
	if s.openItem != "" {
		t.Fatal("медленный второй клик открыл карточку")
	}
	view(m)
	// клик по превью — фокус
	lw, _ := s.paneWidths()
	feed(t, m, click(lw+5, 5))
	if s.focus != 1 {
		t.Fatalf("focus=%d", s.focus)
	}
	// колёсико над таблицей двигает курсор
	feed(t, m, click(5, s.bodyTop()+1))
	feed(t, m, wheel(5, 5, false))
	if s.cursor != s.visible()[len(s.visible())-1].ID {
		t.Fatalf("cursor after wheel=%s", s.cursor)
	}
	feed(t, m, wheel(5, 5, true))
	if s.cursor != s.visible()[0].ID {
		t.Fatalf("cursor after wheel up=%s", s.cursor)
	}
	// не-нажатия и области вне таблицы
	cur := s.cursor
	feed(t, m, tea.MouseMsg{X: 5, Y: 6, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease})
	feed(t, m, click(5, s.height-1))
	if s.cursor != cur {
		t.Fatal("курсор сдвинулся")
	}
}

func TestContractTUIV6MouseOptions(t *testing.T) {
	on := programOptions(&config.Config{})
	f := false
	off := programOptions(&config.Config{UI: config.UI{Mouse: &f}})
	if len(on) != len(off)+1 {
		t.Fatalf("on=%d off=%d", len(on), len(off))
	}
}

func TestContractTUIV6MousePanels(t *testing.T) {
	m, _ := newTestModelCfg(t, StartAt{Project: "PROJ"}, func(c *config.Config) { c.UI.PreviewWidth = 25 })
	s := m.s
	l0, r0 := s.paneWidths()
	if l0+r0+1 != s.width || r0 != s.width-s.width*75/100-1 {
		t.Fatalf("l=%d r=%d", l0, r0)
	}
	press(t, m, ">")
	if _, r := s.paneWidths(); r <= r0 || s.panel.pct != 30 {
		t.Fatalf("pct=%d r=%d", s.panel.pct, r)
	}
	for range 20 {
		press(t, m, ">")
	}
	if s.panel.pct != 80 {
		t.Fatalf("max pct=%d", s.panel.pct)
	}
	for range 20 {
		press(t, m, "<")
	}
	if s.panel.pct != 20 {
		t.Fatalf("min pct=%d", s.panel.pct)
	}
	// на весь экран и обратно
	press(t, m, "z")
	if l, r := s.paneWidths(); l != 0 || r != s.width || s.focus != 1 {
		t.Fatalf("zoom l=%d r=%d", l, r)
	}
	if v := view(m); len(v) == 0 {
		t.Fatal("пустой вид")
	}
	feed(t, m, click(3, 5))
	if len(m.stack) != 1 {
		t.Fatal("клик в режиме превью")
	}
	press(t, m, "esc")
	if s.panel.zoom {
		t.Fatal("esc не закрыл zoom")
	}
	press(t, m, "z", "z")
	if s.panel.zoom {
		t.Fatal("повторное z")
	}
	// узкий терминал: одна панель, z — превью на весь экран
	feed(t, m, tea.WindowSizeMsg{Width: 80, Height: 30})
	if l, r := s.paneWidths(); r != 0 || l != 80 {
		t.Fatalf("narrow l=%d r=%d", l, r)
	}
	press(t, m, "z")
	if l, r := s.paneWidths(); l != 0 || r != 80 {
		t.Fatalf("narrow zoom l=%d r=%d", l, r)
	}
}

// Клик по каждому показанному табу выбирает его; границы — из той же раскладки, что и tabsLine, на двух ширинах.
func TestMouseTabSpansBothWidths(t *testing.T) {
	for _, w := range []int{120, 60} {
		m, _, _, _ := newTestModel(t)
		s := m.s
		feed(t, m, tea.WindowSizeMsg{Width: w, Height: 30})
		l := s.tabsLayout(w)
		for i := range s.tabList() {
			sp := s.tabSpans(w)
			if i < l.lo || i > l.hi {
				if sp[i] != [2]int{} {
					t.Fatalf("w=%d: невидимый таб %d имеет span %v", w, i, sp[i])
				}
				continue
			}
			feed(t, m, click(sp[i][0], 0))
			if s.tab != i {
				t.Fatalf("w=%d: клик по табу %d дал tab=%d (spans %v)", w, i, s.tab, sp)
			}
			l = s.tabsLayout(w)
		}
	}
}
