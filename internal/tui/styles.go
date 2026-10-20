package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/thegriglat/platui/internal/config"
)

// styles is a style set built from config.Color; with ColorsDisabled no colors are applied.
type styles struct {
	accent, muted, text, err, ok, warn, sel, border lipgloss.Style
	tabOn, tabOff, bold                             lipgloss.Style
	prio                                            map[string]lipgloss.Style
	plain                                           bool
}

func newStyles(cfg *config.Config) styles {
	if cfg.ColorsDisabled() {
		p := lipgloss.NewStyle()
		return styles{accent: p, muted: p, text: p, err: p, ok: p, warn: p, sel: p.Reverse(true), border: p,
			tabOn: p.Bold(true).Underline(true), tabOff: p, bold: p.Bold(true), prio: map[string]lipgloss.Style{}, plain: true}
	}
	fg := func(k string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Color(k))) }
	st := styles{accent: fg("accent"), muted: fg("muted"), text: fg("text"), err: fg("error"), ok: fg("success"),
		warn: fg("warning"), border: fg("border"), bold: lipgloss.NewStyle().Bold(true),
		sel:  lipgloss.NewStyle().Background(lipgloss.Color(cfg.Color("selection"))),
		prio: map[string]lipgloss.Style{}}
	st.tabOn = fg("accent").Bold(true).Underline(true)
	st.tabOff = fg("muted")
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
