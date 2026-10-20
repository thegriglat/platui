package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
)

// confirmScreen — диалог y/N: подтверждает только y/Y, всё остальное отменяет.
type confirmScreen struct {
	question string
	yes      func() tea.Cmd // вызывается синхронно в Update при подтверждении
	no       func() tea.Cmd // не nil — вызывается вместо «отменено» при отказе
}

// confirm открывает диалог подтверждения деструктивного действия; onYes выполняется только после y/Y.
func confirm(s *session, question string, onYes tea.Cmd) tea.Cmd {
	return confirmWith(question, func() tea.Cmd { return onYes })
}

// confirmWith — как confirm, но действие вычисляется при подтверждении (с побочными эффектами в сессии).
func confirmWith(question string, yes func() tea.Cmd) tea.Cmd {
	return pushScreen(&confirmScreen{question: question, yes: yes})
}

func (c *confirmScreen) Title() string       { return "подтверждение" }
func (c *confirmScreen) Scope() string       { return "dialog" }
func (c *confirmScreen) CapturesInput() bool { return true }

func (c *confirmScreen) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return c, nil
	}
	if n := keyName(k); n == "y" || n == "Y" {
		return c, tea.Sequence(popScreen, c.yes())
	}
	if c.no != nil {
		return c, tea.Sequence(popScreen, c.no())
	}
	return c, tea.Sequence(popScreen, func() tea.Msg { return statusMsg{"отменено", false} })
}

func (c *confirmScreen) View(s *session) string {
	head, rest, _ := strings.Cut(c.question, "\n") // первая строка — вопрос, остальные — детали
	rows := []string{s.st.warn.Bold(true).Render(" " + head)}
	for _, l := range strings.Split(rest, "\n") {
		if l != "" {
			rows = append(rows, "  "+l)
		}
	}
	rows = append(rows, "", s.st.muted.Render(" y — да   любая другая клавиша — нет (по умолчанию)"))
	for len(rows) < s.height-1 {
		rows = append(rows, "")
	}
	return strings.Join(append(rows, s.statusView(s.width)), "\n")
}

// destructiveState — переход в completed/cancelled требует подтверждения.
func destructiveState(st plane.State) bool { return st.Group == "completed" || st.Group == "cancelled" }
