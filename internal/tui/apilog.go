package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
)

// apiTickEvery is how often the screen is redrawn to show API activity.
const apiTickEvery = 300 * time.Millisecond

type apiTickMsg struct{}

// apiTick redraws the screen periodically while a monitor is set.
func apiTick(s *session) tea.Cmd {
	if s.deps.Monitor == nil {
		return nil
	}
	return tea.Tick(apiTickEvery, func(time.Time) tea.Msg { return apiTickMsg{} })
}

// apiActivity describes what the API is doing now: list progress or requests in flight; "" — idle.
func (s *session) apiActivity(snap plane.MonitorSnapshot) string {
	var parts []string
	for _, l := range snap.Lists {
		p := fmt.Sprintf("%s %d", l.Resource, l.Page)
		if l.Pages > 0 {
			p += fmt.Sprintf("/%d", l.Pages)
		}
		if l.Count > 0 {
			p += fmt.Sprintf(" (%d/%d)", l.Got, l.Count)
		}
		el := s.deps.Now().Sub(l.Start)
		p += " " + fmtDur(el)
		if l.Pages > l.Page && l.Page > 0 {
			p += ", ~" + fmtDur(el/time.Duration(l.Page)*time.Duration(l.Pages-l.Page)) + " left"
		}
		parts = append(parts, p)
	}
	if len(parts) == 0 {
		for _, r := range snap.InFlight {
			if r.Page == 0 {
				parts = append(parts, r.Method+" "+r.Path)
			}
		}
	}
	if len(parts) == 0 {
		return ""
	}
	mark := "⟳ "
	if s.polling {
		mark = "↻ poll: "
	}
	return mark + strings.Join(parts, " · ")
}

// apiCounters is the right part of the status line: requests, traffic, errors, rate limit.
func apiCounters(snap plane.MonitorSnapshot) string {
	t := fmt.Sprintf("API %d req · %s", snap.Total, fmtBytes(snap.Bytes))
	if snap.Errors > 0 {
		t += fmt.Sprintf(" · %d err", snap.Errors)
	}
	if snap.RateRemaining >= 0 {
		t += fmt.Sprintf(" · limit %d", snap.RateRemaining)
	}
	return t + " "
}

func fmtBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%d KB", n>>10)
	}
	return fmt.Sprintf("%d B", n)
}

func fmtDur(d time.Duration) string {
	if d < 10*time.Second {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return d.Round(time.Second).String()
}
