package tui

import (
	"slices"
	"strings"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/plane"
)

type tabInfo struct {
	label string
	state *plane.State // nil — «Все»
}

// tabList — «Все» (если включён) и состояния-табы.
func (s *session) tabList() []tabInfo {
	var out []tabInfo
	if s.deps.Config.TabsAll() {
		out = append(out, tabInfo{label: "Все"})
	}
	for i := range s.tabs {
		out = append(out, tabInfo{label: s.tabs[i].Name, state: &s.tabs[i]})
	}
	return out
}

func (s *session) setTab(i int) {
	s.tab = max(0, min(i, len(s.tabList())-1))
}

func (s *session) inTab(it plane.WorkItem, t tabInfo) bool {
	if t.state != nil {
		return it.StateID == t.state.ID
	}
	return slices.ContainsFunc(s.tabs, func(st plane.State) bool { return st.ID == it.StateID })
}

// pass — фильтры «Мои», сохранённый фильтр и поиск (без учёта таба).
func (s *session) pass(it plane.WorkItem) bool {
	if s.mine && !slices.Contains(it.AssigneeIDs, s.me.ID) {
		return false
	}
	if cf := s.deps.Config.Filters; s.filter >= 1 && s.filter <= len(cf) &&
		!app.Match(cf[s.filter-1], it, s.meta, s.me.ID) {
		return false
	}
	if s.hits != nil {
		_, ok := s.hits[it.ID]
		return ok
	}
	return true
}

// visible — задачи текущего таба с учётом mine/filter/query/sortBy.
func (s *session) visible() []plane.WorkItem {
	tl := s.tabList()
	if s.tab >= len(tl) {
		return nil
	}
	var out []plane.WorkItem
	for _, it := range s.items {
		if s.inTab(it, tl[s.tab]) && s.pass(it) {
			out = append(out, it)
		}
	}
	if s.hits != nil {
		slices.SortStableFunc(out, func(a, b plane.WorkItem) int { return s.hits[b.ID].Score - s.hits[a.ID].Score })
		return out
	}
	s.sortItems(out)
	return out
}

// tabCount — число задач таба i с учётом mine/filter/query.
func (s *session) tabCount(i int) int {
	tl := s.tabList()
	if i < 0 || i >= len(tl) {
		return 0
	}
	n := 0
	for _, it := range s.items {
		if s.inTab(it, tl[i]) && s.pass(it) {
			n++
		}
	}
	return n
}

var prioRank = map[string]int{"urgent": 0, "high": 1, "medium": 2, "low": 3, "none": 4}

func (s *session) sortItems(l []plane.WorkItem) {
	rank := func(p string) int {
		if r, ok := prioRank[p]; ok {
			return r
		}
		return 5
	}
	stateRank := func(id string) int { return slices.IndexFunc(s.tabs, func(t plane.State) bool { return t.ID == id }) }
	switch s.sortBy {
	case sortPriority:
		slices.SortStableFunc(l, func(a, b plane.WorkItem) int { return rank(a.Priority) - rank(b.Priority) })
	case sortState:
		slices.SortStableFunc(l, func(a, b plane.WorkItem) int { return stateRank(a.StateID) - stateRank(b.StateID) })
	case sortUpdated:
		slices.SortStableFunc(l, func(a, b plane.WorkItem) int { return b.UpdatedAt.Compare(a.UpdatedAt) })
	case sortID:
		slices.SortStableFunc(l, func(a, b plane.WorkItem) int { return a.Sequence - b.Sequence })
	}
}

// tabsLayout — раскладка строки табов: единственный источник и для tabsLine, и для мыши (tabSpans).
type tabsLayout struct {
	prefix string   // идентификатор проекта
	right  string   // признаки «Мои» и фильтра
	cells  []string // отрисованные табы (все)
	lo, hi int      // окно показанных табов, включительно
}

func (s *session) tabsLayout(width int) tabsLayout {
	ident := ""
	if s.project != nil {
		ident = s.project.Identifier
	}
	l := tabsLayout{prefix: s.st.accent.Bold(true).Render(" " + ident + " ")}
	var marks []string
	if s.mine {
		marks = append(marks, "[Мои]")
	}
	if cf := s.deps.Config.Filters; s.filter >= 1 && s.filter <= len(cf) {
		marks = append(marks, "[фильтр: "+cf[s.filter-1].Name+"]")
	}
	l.right = strings.Join(marks, " ")
	avail := width - lipglossWidth(l.prefix) - lipglossWidth(l.right) - 1
	tl := s.tabList()
	l.cells = make([]string, len(tl))
	for i, t := range tl {
		txt := " " + t.label + " " + itoa(s.tabCount(i)) + " "
		if i == s.tab {
			l.cells[i] = s.st.tabOn.Render(txt)
		} else {
			l.cells[i] = s.st.tabOff.Render(txt)
		}
	}
	l.lo, l.hi = s.tab, s.tab
	if len(l.cells) == 0 {
		return l
	}
	used := lipglossWidth(l.cells[s.tab])
	for grown := true; grown; {
		grown = false
		if l.hi+1 < len(l.cells) && used+1+lipglossWidth(l.cells[l.hi+1]) <= avail {
			l.hi++
			used += 1 + lipglossWidth(l.cells[l.hi])
			grown = true
		}
		if l.lo > 0 && used+1+lipglossWidth(l.cells[l.lo-1]) <= avail {
			l.lo--
			used += 1 + lipglossWidth(l.cells[l.lo])
			grown = true
		}
	}
	return l
}

// tabsLine — строка табов со счётчиками, признаками «Мои» и фильтра.
func tabsLine(s *session, width int) string {
	l := s.tabsLayout(width)
	line := l.prefix
	if len(l.cells) > 0 {
		line += strings.Join(l.cells[l.lo:l.hi+1], s.st.border.Render("│"))
	}
	if l.right != "" {
		line += strings.Repeat(" ", max(1, width-lipglossWidth(line)-lipglossWidth(l.right))) + s.st.accent.Render(l.right)
	}
	return clip(line, width)
}
