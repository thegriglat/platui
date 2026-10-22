package tui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/thegriglat/platui/internal/config"
)

// styles is a style set built from config.Color; with ColorsDisabled no colors are applied.
type styles struct {
	accent, muted, text, err, ok, warn, sel, border lipgloss.Style
	tabOn, tabOff, bold, hintKey, hintText, bar     lipgloss.Style
	prio                                            map[string]lipgloss.Style
	plain                                           bool
}

func newStyles(cfg *config.Config) styles {
	if cfg.ColorsDisabled() {
		p := lipgloss.NewStyle()
		return styles{accent: p, muted: p, text: p, err: p, ok: p, warn: p, sel: p.Reverse(true), border: p,
			hintKey: p.Bold(true), hintText: p, bar: p,
			tabOn: p.Bold(true).Underline(true), tabOff: p, bold: p.Bold(true), prio: map[string]lipgloss.Style{}, plain: true}
	}
	fg := func(k string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Color(k))) }
	st := styles{accent: fg("accent"), muted: fg("muted"), text: fg("text"), err: fg("error"), ok: fg("success"),
		warn: fg("warning"), border: fg("border"), bold: lipgloss.NewStyle().Bold(true),
		sel:  lipgloss.NewStyle().Background(lipgloss.Color(cfg.Color("selection"))),
		prio: map[string]lipgloss.Style{}}
	st.tabOn = fg("accent").Bold(true).Underline(true)
	st.tabOff = fg("muted")
	st.bar = lipgloss.NewStyle().Background(lipgloss.Color(cfg.Color("bar")))
	st.hintKey = fg("accent").Bold(true).Inherit(st.bar)
	st.hintText = fg("muted").Inherit(st.bar)
	for _, p := range []string{"urgent", "high", "medium", "low"} {
		st.prio[p] = fg("priority_" + p)
	}
	return st
}

// priority returns the priority style (uncolored for an unknown one).
func (st styles) priority(p string) lipgloss.Style {
	if s, ok := st.prio[p]; ok {
		return s
	}
	return st.text
}

var sgr = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// paint applies the background of bg to a line that already has styled
// fragments: their resets would otherwise clear it after the first one.
func paint(bg lipgloss.Style, line string) string {
	pre, post, ok := strings.Cut(bg.Render("\u2063"), "\u2063")
	if !ok || pre == "" {
		return bg.Render(line)
	}
	return pre + sgr.ReplaceAllString(line, "${0}"+pre) + post
}
