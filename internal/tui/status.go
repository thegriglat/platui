package tui

import "strings"

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
	if s.loading {
		txt, st = "⟳ loading… "+txt, s.st.warn
	}
	return st.Render(clip(txt, width))
}
