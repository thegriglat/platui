package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// inputScreen — модальная строка ввода (PT-5d): новая задача, имя ветки.
type inputScreen struct {
	title, hint string
	text        []rune
	busy        bool
	onSubmit    func(c *inputScreen) tea.Cmd              // Enter
	onKey       func(c *inputScreen, key string) tea.Cmd  // дополнительные клавиши (nil — нет)
	onMsg       func(c *inputScreen, msg tea.Msg) tea.Cmd // прочие сообщения (nil — нет)
	extra       func() string                             // доп. строка под вводом

	// Подсказки под вводом (похожие задачи): при изменении текста sim пересчитывается;
	// ↓ переходит в список, Enter на строке вызывает onPick вместо onSubmit.
	similar func(text string) []string
	onPick  func(c *inputScreen, i int) tea.Cmd
	sim     []string
	simCur  int // -1 — фокус в строке ввода
}

func (c *inputScreen) refreshSim() {
	c.simCur = -1
	if c.similar != nil {
		c.sim = c.similar(string(c.text))
	}
}

func (c *inputScreen) Title() string       { return c.title }
func (c *inputScreen) Scope() string       { return "dialog" }
func (c *inputScreen) CapturesInput() bool { return true }

func (c *inputScreen) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		if c.onMsg != nil {
			return c, c.onMsg(c, msg)
		}
		return c, nil
	}
	name := keyName(k)
	if c.busy {
		return c, nil
	}
	if c.onKey != nil && strings.Contains(name, "+") {
		if cmd := c.onKey(c, name); cmd != nil {
			return c, cmd
		}
	}
	switch k.Type {
	case tea.KeyEsc:
		return c, popScreen
	case tea.KeyDown:
		if c.simCur+1 < len(c.sim) {
			c.simCur++
		}
	case tea.KeyUp:
		c.simCur = max(c.simCur-1, -1)
	case tea.KeyEnter:
		if c.simCur >= 0 && c.simCur < len(c.sim) && c.onPick != nil {
			return c, c.onPick(c, c.simCur)
		}
		return c, c.onSubmit(c)
	case tea.KeyBackspace:
		if len(c.text) > 0 {
			c.text = c.text[:len(c.text)-1]
		}
		c.refreshSim()
	case tea.KeySpace:
		c.text = append(c.text, ' ')
		c.refreshSim()
	case tea.KeyRunes:
		if !k.Alt {
			c.text = append(c.text, k.Runes...)
			c.refreshSim()
		}
	}
	return c, nil
}

func (c *inputScreen) View(s *session) string {
	rows := []string{s.st.accent.Bold(true).Render(" " + c.title), " > " + string(c.text) + "▏"}
	if c.extra != nil {
		if e := c.extra(); e != "" {
			rows = append(rows, s.st.muted.Render(" "+e))
		}
	}
	if len(c.sim) > 0 {
		rows = append(rows, s.st.muted.Render(" Похожие задачи (↓ — выбрать, Enter — открыть):"))
		for i, l := range c.sim {
			if i == c.simCur {
				rows = append(rows, s.st.sel.Render(clip("  > "+l, s.width)))
			} else {
				rows = append(rows, s.st.muted.Render(clip("    "+l, s.width)))
			}
		}
	}
	for len(rows) < s.height-2 {
		rows = append(rows, "")
	}
	rows = append(rows, s.st.muted.Render(" "+c.hint), s.statusView(s.width))
	return clip2(rows, s.height)
}
