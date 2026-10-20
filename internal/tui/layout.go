package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const wideWidth = 100 // от этой ширины — две панели

func itoa(n int) string          { return strconv.Itoa(n) }
func lipglossWidth(s string) int { return lipgloss.Width(s) }

// clip обрезает строку до w колонок (с «…»).
func clip(s string, w int) string {
	if w <= 0 {
		return ""
	}
	return ansi.Truncate(s, w, "…")
}

// fitBlock приводит блок к ровно w×h: обрезает и дополняет пробелами/пустыми строками.
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

// paneWidths — ширина панелей: справа превью (ui.preview_width, % ширины), разделитель, слева остаток;
// right == 0 — одна панель; left == 0 — превью на весь экран (z).
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

// searchRow — строка поиска; пусто — не показывается.
func (s *session) searchRow() string {
	switch {
	case s.searchLine != "":
		return s.searchLine
	case s.query != "":
		return "/ " + s.query + "▏  найдено " + itoa(len(s.visible()))
	}
	return ""
}

// frame собирает экран: табы, поиск, панели, подсказки, статус.
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
