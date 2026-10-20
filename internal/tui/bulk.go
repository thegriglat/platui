package tui

import (
	"context"
	"fmt"
	"slices"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
)

func init() {
	main := []string{"main"}
	for _, a := range []action{
		{ID: "bulk.state", Title: "Состояние отмеченных", Group: "задача", Keys: []string{"s"}, Bulk: true, Scopes: main,
			Run: func(m *model) tea.Cmd { return bulkState(m.s) }},
		{ID: "bulk.assign", Title: "Назначить на себя / снять (отмеченные)", Group: "задача", Keys: []string{"a"}, Bulk: true,
			Scopes: main, Run: func(m *model) tea.Cmd { return bulkAssign(m.s) }},
		{ID: "bulk.label", Title: "Метка отмеченным", Group: "задача", Keys: []string{"L"}, Bulk: true, Scopes: main,
			Run: func(m *model) tea.Cmd { return bulkLabel(m.s) }},
	} {
		registerAction(a)
	}
}

// markedItems — отмеченные задачи проекта в порядке API.
func (s *session) markedItems() []plane.WorkItem {
	var out []plane.WorkItem
	for _, it := range s.items {
		if s.marks[it.ID] {
			out = append(out, it)
		}
	}
	return out
}

// bulkRun выполняет массовую операцию в фоне; итог «n ок, m ошибок» — в статусе, затем перечитывание задач.
func bulkRun(s *session, items []plane.WorkItem, op func(ctx context.Context, pid string) map[string]error) tea.Cmd {
	s.marks = map[string]bool{}
	pid := s.project.ID
	return func() tea.Msg {
		res := op(context.Background(), pid)
		ok, fail := 0, 0
		var first error
		for _, it := range items {
			if err := res[it.ID]; err != nil {
				fail++
				first = cmpFirst(first, err)
			} else {
				ok++
			}
		}
		text := fmt.Sprintf("%d ок, %d ошибок", ok, fail)
		if first != nil {
			text += ": " + first.Error()
		}
		return tea.BatchMsg{func() tea.Msg { return statusMsg{text, fail > 0} }, loadItems(s, false)}
	}
}

func cmpFirst(a, b error) error {
	if a != nil {
		return a
	}
	return b
}

func bulkPrecheck(s *session) ([]plane.WorkItem, bool) {
	items := s.markedItems()
	if s.project == nil || len(items) == 0 {
		s.setStatus("нет отмеченных задач", true)
		return nil, false
	}
	return items, true
}

func bulkState(s *session) tea.Cmd {
	items, ok := bulkPrecheck(s)
	if !ok {
		return nil
	}
	states := s.meta.States
	names := make([]string, len(states))
	for i, st := range states {
		names[i] = st.Name
	}
	ids := make([]string, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	return pushScreen(&bulkChoice{title: fmt.Sprintf("Состояние для %d задач", len(items)), opts: names,
		pick: func(i int) tea.Cmd {
			id := states[i].ID
			q := fmt.Sprintf("Изменить %d задач?", len(items))
			if destructiveState(states[i]) {
				q = fmt.Sprintf("Перевести %d задач в %s?", len(items), states[i].Name)
			}
			return confirmWith(q, func() tea.Cmd {
				return bulkRun(s, items, func(ctx context.Context, pid string) map[string]error {
					return s.deps.App.BulkPatch(ctx, pid, ids, plane.WorkItemPatch{StateID: &id})
				})
			})
		}})
}

func bulkAssign(s *session) tea.Cmd {
	items, ok := bulkPrecheck(s)
	if !ok {
		return nil
	}
	me := s.me.ID
	add := slices.ContainsFunc(items, func(it plane.WorkItem) bool { return !slices.Contains(it.AssigneeIDs, me) })
	return confirmWith(fmt.Sprintf("Изменить %d задач?", len(items)), func() tea.Cmd {
		return bulkRun(s, items, func(ctx context.Context, pid string) map[string]error {
			return s.deps.App.BulkAssign(ctx, pid, items, me, add)
		})
	})
}

func bulkLabel(s *session) tea.Cmd {
	items, ok := bulkPrecheck(s)
	if !ok {
		return nil
	}
	labels := s.meta.Labels
	var names []string
	for _, l := range labels {
		names = append(names, "добавить: "+l.Name)
	}
	for _, l := range labels {
		names = append(names, "снять: "+l.Name)
	}
	return pushScreen(&bulkChoice{title: fmt.Sprintf("Метка для %d задач", len(items)), opts: names,
		pick: func(i int) tea.Cmd {
			add, id := i < len(labels), labels[i%len(labels)].ID
			return confirmWith(fmt.Sprintf("Изменить %d задач?", len(items)), func() tea.Cmd {
				return bulkRun(s, items, func(ctx context.Context, pid string) map[string]error {
					return s.deps.App.BulkLabel(ctx, pid, items, id, add)
				})
			})
		}})
}

// bulkChoice — список выбора для массовых действий.
type bulkChoice struct {
	title string
	opts  []string
	cur   int
	pick  func(i int) tea.Cmd
}

func (c *bulkChoice) Title() string       { return c.title }
func (c *bulkChoice) Scope() string       { return "dialog" }
func (c *bulkChoice) CapturesInput() bool { return true }

func (c *bulkChoice) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return c, nil
	}
	switch keyName(k) {
	case "esc", "q":
		return c, popScreen
	case "up":
		c.cur = max(c.cur-1, 0)
	case "down":
		c.cur = max(min(c.cur+1, len(c.opts)-1), 0)
	case "home":
		c.cur = 0
	case "end":
		c.cur = max(len(c.opts)-1, 0)
	case "enter":
		if c.cur < len(c.opts) {
			// выбор вычисляется синхронно (сбрасывает отметки), закрытие — до запуска
			return c, tea.Sequence(popScreen, c.pick(c.cur))
		}
	}
	return c, nil
}

func (c *bulkChoice) View(s *session) string {
	rows := []string{s.st.accent.Bold(true).Render(" " + c.title)}
	if len(c.opts) == 0 {
		rows = append(rows, s.st.muted.Render(" нет вариантов"))
	}
	for i, o := range c.opts {
		if i == c.cur {
			rows = append(rows, s.st.sel.Render(clip(" "+o, s.width)))
		} else {
			rows = append(rows, " "+o)
		}
	}
	rows = append(rows, s.st.muted.Render(" ↑↓ выбор   Enter применить   Esc отмена"))
	return clip2(rows, s.height)
}

func clip2(rows []string, h int) string {
	if len(rows) > h && h > 0 {
		rows = rows[:h]
	}
	out := ""
	for i, r := range rows {
		if i > 0 {
			out += "\n"
		}
		out += r
	}
	return out
}
