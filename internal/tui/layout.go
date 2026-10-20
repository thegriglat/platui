package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const wideWidth = 100 // two panes from this width

func itoa(n int) string          { return strconv.Itoa(n) }
func lipglossWidth(s string) int { return lipgloss.Width(s) }

// clip truncates the string to w columns (with "…").
func clip(s string, w int) string {
	if w <= 0 {
		return ""
	}
	return ansi.Truncate(s, w, "…")
}

// fitBlock fits the block to exactly w×h: truncates and pads with spaces/empty lines.
func fitBlock(s string, w, h int) []string {
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for i, l := range lines {
		l = clip(l, w)
		lines[i] = l + strings.Repeat(" ", max(0, w-lipgloss.Width(l)))
	}
	for len(lines) < h {
		lines = append(lines, strings.Repeat(" ", w))
	}
	return lines
}

// paneWidths returns pane widths: preview on the right (ui.preview_width, % of width), separator, remainder on the left;
// right == 0 — single pane; left == 0 — preview full screen (z).
func (s *session) paneWidths() (left, right int) {
	if newPreview == nil {
		return s.width, 0
	}
	p := s.panel
	if p.zoom {
		return 0, s.width
	}
	if s.width < wideWidth {
		return s.width, 0
	}
	left = s.width * (100 - p.pct) / 100
	return left, s.width - left - 1
}

// searchRow is the search line; empty means it is not shown.
func (s *session) searchRow() string {
	switch {
	case s.searchLine != "":
		return s.searchLine
	case s.query != "":
		return "/ " + s.query + "▏  found " + itoa(len(s.visible()))
	}
	return ""
}

// frame assembles the screen: tabs, search, panes, hints, status.
func (s *session) frame(scope string, table, preview func(w, h int) string) string {
	rows := []string{tabsLine(s, s.width)}
	if r := s.searchRow(); r != "" {
		rows = append(rows, s.st.accent.Render(clip(r, s.width)))
	}
	h := max(s.height-len(rows)-2, 1)
	lw, rw := s.paneWidths()
	var body []string
	switch {
	case lw == 0 && preview != nil:
		body = fitBlock(preview(rw, h), rw, h)
	case rw == 0 || preview == nil:
		body = fitBlock(table(lw, h), lw, h)
	default:
		l, r := fitBlock(table(lw, h), lw, h), fitBlock(preview(rw, h), rw, h)
		sep := s.st.border.Render("│")
		for i := range l {
			body = append(body, l[i]+sep+r[i])
		}
	}
	rows = append(rows, body...)
	rows = append(rows, hintsLine(s, scope, s.width), s.statusView(s.width))
	return strings.Join(rows, "\n")
}
