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
	// click on a tab
	sp := s.tabSpans(s.width)
	if sp[2][1] <= sp[2][0] {
		t.Fatalf("spans=%v", sp)
	}
	feed(t, m, click(sp[2][0]+1, 0))
	if s.tab != 2 {
		t.Fatalf("tab=%d", s.tab)
	}
	// click on a row: cursor (view is rendered — as between events in the program)
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
		t.Fatal("single click opened the card")
	}
	// double click on the same row within 400 ms — card
	now := t0
	s.deps.Now = func() time.Time { return now }
	feed(t, m, click(5, s.bodyTop()+1))
	now = now.Add(100 * time.Millisecond)
	feed(t, m, click(5, s.bodyTop()+1))
	if s.openItem != vis[0].ID || len(m.stack) != 2 {
		t.Fatalf("openItem=%q stack=%d", s.openItem, len(m.stack))
	}
	feed(t, m, click(5, s.bodyTop()+2)) // mouse on the card screen is ignored
	if len(m.stack) != 2 {
		t.Fatal("click changed the stack")
	}
	m.pop()
	// slow second click — not a double click
	feed(t, m, click(5, s.bodyTop()+2))
	now = now.Add(time.Second)
	feed(t, m, click(5, s.bodyTop()+2))
	if s.openItem != "" {
		t.Fatal("slow second click opened the card")
	}
	view(m)
	// click on the preview — focus
	lw, _ := s.paneWidths()
	feed(t, m, click(lw+5, 5))
	if s.focus != 1 {
		t.Fatalf("focus=%d", s.focus)
	}
	// wheel over the table moves the cursor
	feed(t, m, click(5, s.bodyTop()+1))
	feed(t, m, wheel(5, 5, false))
	if s.cursor != s.visible()[len(s.visible())-1].ID {
		t.Fatalf("cursor after wheel=%s", s.cursor)
	}
	feed(t, m, wheel(5, 5, true))
	if s.cursor != s.visible()[0].ID {
		t.Fatalf("cursor after wheel up=%s", s.cursor)
	}
	// non-press events and areas outside the table
	cur := s.cursor
	feed(t, m, tea.MouseMsg{X: 5, Y: 6, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease})
	feed(t, m, click(5, s.height-1))
	if s.cursor != cur {
		t.Fatal("cursor moved")
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
	// full screen and back
	press(t, m, "z")
	if l, r := s.paneWidths(); l != 0 || r != s.width || s.focus != 1 {
		t.Fatalf("zoom l=%d r=%d", l, r)
	}
	if v := view(m); len(v) == 0 {
		t.Fatal("empty view")
	}
	feed(t, m, click(3, 5))
	if len(m.stack) != 1 {
		t.Fatal("click in preview mode")
	}
	press(t, m, "esc")
	if s.panel.zoom {
		t.Fatal("esc did not close zoom")
	}
	press(t, m, "z", "z")
	if s.panel.zoom {
		t.Fatal("second z")
	}
	// narrow terminal: one pane, z is preview full screen
	feed(t, m, tea.WindowSizeMsg{Width: 80, Height: 30})
	if l, r := s.paneWidths(); r != 0 || l != 80 {
		t.Fatalf("narrow l=%d r=%d", l, r)
	}
	press(t, m, "z")
	if l, r := s.paneWidths(); l != 0 || r != 80 {
		t.Fatalf("narrow zoom l=%d r=%d", l, r)
	}
}

// A click on each shown tab selects it; bounds come from the same layout as tabsLine, at two widths.
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
					t.Fatalf("w=%d: hidden tab %d has span %v", w, i, sp[i])
				}
				continue
			}
			feed(t, m, click(sp[i][0], 0))
			if s.tab != i {
				t.Fatalf("w=%d: click on tab %d gave tab=%d (spans %v)", w, i, s.tab, sp)
			}
			l = s.tabsLayout(w)
		}
	}
}
