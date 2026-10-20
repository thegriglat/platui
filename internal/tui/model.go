package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/config"
)

// model — корневая модель: стек экранов, глобальные клавиши, маршрутизация сообщений.
type model struct {
	s       *session
	stack   []screen
	lastKey tea.KeyMsg
}

// New создаёт корневую модель.
func New(d Deps) tea.Model { return newModel(d) }

func newModel(d Deps) *model {
	s := newSession(d.withDefaults())
	return &model{s: s, stack: []screen{newMainScreen(s)}}
}

// Run запускает программу в альтернативном экране.
func Run(d Deps) error {
	m := newModel(d)
	defer m.s.cancel() // выход из программы отменяет выполняющиеся команды
	_, err := tea.NewProgram(m, programOptions(m.s.deps.Config)...).Run()
	return err
}

// programOptions — опции программы: альтернативный экран, мышь — при ui.mouse (по умолчанию да).
func programOptions(cfg *config.Config) []tea.ProgramOption {
	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if cfg.MouseEnabled() {
		opts = append(opts, tea.WithMouseCellMotion())
	}
	return opts
}

func (m *model) top() screen { return m.stack[len(m.stack)-1] }

func (m *model) Init() tea.Cmd { return bootstrap(m.s) }

func (m *model) View() string {
	if m.s.width == 0 {
		return ""
	}
	return m.top().View(m.s)
}

func (m *model) pop() {
	if len(m.stack) > 1 {
		if m.top().Scope() == "item" {
			m.s.openItem = ""
		}
		m.stack = m.stack[:len(m.stack)-1]
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.s.width, m.s.height = msg.Width, msg.Height
		if _, r := m.s.paneWidths(); r == 0 {
			m.s.focus = 0
		}
		return m, nil
	case tea.KeyMsg:
		return m, m.key(msg)
	case tea.MouseMsg:
		// мышь — только верхнему экрану; диалоги и оверлеи её игнорируют
		if ms, ok := m.top().(*mainScreen); ok {
			return m, ms.mouse(m.s, msg)
		}
		return m, nil
	case pushScreenMsg:
		m.stack = append(m.stack, msg.screen)
		return m, nil
	case popScreenMsg:
		m.pop()
		return m, nil
	}
	if _, ok := msg.(openProjectMsg); ok {
		// смена проекта закрывает карточку прежнего проекта
		for len(m.stack) > 1 && m.top().Scope() == "item" {
			m.pop()
		}
	}
	cmds := []tea.Cmd{m.s.handle(msg)}
	for i := range m.stack {
		var c tea.Cmd
		m.stack[i], c = m.stack[i].Update(m.s, msg)
		cmds = append(cmds, c)
	}
	return m, tea.Batch(cmds...)
}

// key: Ctrl+C всегда выходит; экран с вводом получает клавиши сам; иначе — действие
// по реестру для области экрана, иначе клавишу получает экран.
func (m *model) key(k tea.KeyMsg) tea.Cmd {
	name := keyName(k)
	if name == "ctrl+c" {
		return tea.Quit
	}
	top := m.top()
	if !top.CapturesInput() {
		if a, ok := findAction(top.Scope(), name, len(m.s.marks) > 0); ok && a.Run != nil {
			m.lastKey = k
			return a.Run(m)
		}
		if c, ok := m.s.customFor(top.Scope(), name); ok {
			return runCustom(m.s, top.Scope(), c)
		}
	}
	var cmd tea.Cmd
	m.stack[len(m.stack)-1], cmd = top.Update(m.s, k)
	return cmd
}

// forward передаёт последнюю клавишу текущему экрану.
func (m *model) forward() tea.Cmd {
	var cmd tea.Cmd
	m.stack[len(m.stack)-1], cmd = m.top().Update(m.s, m.lastKey)
	return cmd
}
