package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type statusLine struct {
	text  string
	isErr bool
}

func (s *session) setStatus(text string, isErr bool) { s.status = statusLine{text, isErr} }

func (s *session) statusView(width int) string {
	txt, st := s.status.text, s.st.ok
	if s.status.isErr {
		st = s.st.err
	}
	if s.net.offline {
		if !s.status.isErr {
			st = s.st.warn
		}
		switch {
		case txt == "":
			txt = s.net.offlineText()
		case !strings.Contains(txt, "offline"):
			txt = s.net.offlineText() + " | " + txt
		}
	}
	if s.deps.Monitor == nil {
		if s.loading {
			txt, st = "⟳ loading… "+txt, s.st.warn
		}
		return st.Render(clip(txt, width))
	}
	snap := s.deps.Monitor.Snapshot()
	if act := s.apiActivity(snap); act != "" {
		txt, st = strings.TrimSpace(act+"  "+txt), s.st.warn
	} else if s.loading {
		txt, st = "⟳ loading… "+txt, s.st.warn
	}
	right := apiCounters(snap)
	if lipgloss.Width(right)+10 > width {
		return st.Render(clip(txt, width))
	}
	left := clip(txt, width-lipgloss.Width(right)-1)
	gap := strings.Repeat(" ", max(1, width-lipgloss.Width(left)-lipgloss.Width(right)))
	return st.Render(left) + gap + s.st.muted.Render(right)
}
