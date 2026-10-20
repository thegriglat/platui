package tui

import (
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// helpScreen — оверлей справки: привязки области из реестра действий по группам.
type helpScreen struct {
	under string
	vp    viewport.Model
}

func newHelp(under string) *helpScreen {
	vp := viewport.New(0, 0)
	vp.KeyMap = viewport.KeyMap{
		PageDown: key.NewBinding(key.WithKeys("pgdown")), PageUp: key.NewBinding(key.WithKeys("pgup")),
		Down: key.NewBinding(key.WithKeys("down")), Up: key.NewBinding(key.WithKeys("up")),
	}
	return &helpScreen{under: under, vp: vp}
}

func (h *helpScreen) Title() string       { return "справка" }
func (h *helpScreen) Scope() string       { return "help" }
func (h *helpScreen) CapturesInput() bool { return false }

func (h *helpScreen) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch keyName(k) {
		case "?", "h":
			return h, popScreen
		case "home":
			h.vp.GotoTop()
			return h, nil
		case "end":
			h.vp.GotoBottom()
			return h, nil
		}
	}
	var cmd tea.Cmd
	h.vp, cmd = h.vp.Update(msg)
	return h, cmd
}

func (h *helpScreen) content(s *session) string {
	var b strings.Builder
	group := ""
	for _, a := range actionsFor(h.under) {
		if len(a.Keys) == 0 {
			continue
		}
		if a.Group != group {
			group = a.Group
			b.WriteString("\n" + s.st.accent.Bold(true).Render(strings.ToUpper(group[:2])+group[2:]) + "\n")
		}
		keys := keysLabel(a.Keys)
		pad := strings.Repeat(" ", max(1, 22-lipglossWidth(keys)))
		b.WriteString("  " + s.st.bold.Render(keys) + pad + a.Title + "\n")
	}
	if cmds := s.deps.Config.Commands; len(cmds) > 0 && slices.Contains(cardScopes, h.under) {
		b.WriteString("\n" + s.st.accent.Bold(true).Render("Свои") + "\n")
		for _, c := range cmds {
			keys := keysLabel(nonEmpty(c.Key))
			pad := strings.Repeat(" ", max(1, 22-lipglossWidth(keys)))
			b.WriteString("  " + s.st.bold.Render(keys) + pad + c.Name + "\n")
		}
	}
	return b.String()
}

func (h *helpScreen) View(s *session) string {
	h.vp.Width, h.vp.Height = s.width, max(s.height-2, 1)
	h.vp.SetContent(h.content(s))
	head := s.st.accent.Bold(true).Render(" Справка — привязки клавиш")
	foot := s.st.muted.Render(" ↑↓ прокрутка   Esc, q, ?, h закрыть")
	return head + "\n" + h.vp.View() + "\n" + foot
}
