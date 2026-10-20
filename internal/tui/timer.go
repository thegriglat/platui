package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/plane"
)

// timerOf returns the task timer start time.
func (s *session) timerOf(id string) (time.Time, bool) {
	for _, t := range s.deps.App.Timers() {
		if t.ItemID == id {
			return t.Started, true
		}
	}
	return time.Time{}, false
}

func init() {
	registerAction(action{ID: "item.timer", Title: "Timer: start/stop (stop posts a time comment)", Group: "task",
		Keys: []string{"t"}, Hint: "timer", Scopes: cardScopes,
		Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
			s, a, pid := m.s, m.s.deps.App, m.s.project.ID
			key := itoaKey(s, it.Sequence)
			if _, running := s.timerOf(it.ID); !running {
				if err := a.StartTimer(pid, it.ID); err != nil {
					s.setStatus(err.Error(), true)
					return nil
				}
				s.setStatus("⏱ timer started: "+key, false)
				return nil
			}
			return func() tea.Msg {
				d, err := a.StopTimer(context.Background(), pid, it.ID, "")
				if err != nil {
					return statusMsg{err.Error(), true}
				}
				return tea.BatchMsg{
					func() tea.Msg {
						return statusMsg{"⏱ " + key + ": " + app.FormatDuration(d) + ", comment added", false}
					},
					func() tea.Msg { return loadComments(s, pid, it.ID)() },
				}
			}
		})})
}
