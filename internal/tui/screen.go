package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
)

// screen — экран в стеке модели.
type screen interface {
	Update(s *session, msg tea.Msg) (screen, tea.Cmd)
	View(s *session) string
	Title() string
	Scope() string       // "projects"|"main"|"kanban"|"item"|"dialog"|"palette"|"help"
	CapturesInput() bool // текстовый ввод: глобальные клавиши, кроме Ctrl+C, не перехватываются
}

// pane — панель основного экрана.
type pane interface {
	Update(s *session, msg tea.Msg) (pane, tea.Cmd)
	View(s *session, width, height int) string
}

// Точки подключения подзадач PT-5b…5f (присваиваются в init() их файлов).
var (
	newTable   func(s *session) pane   = newStubTable // PT-5b
	newPreview func(s *session) pane                  // PT-5c; nil — превью не показывается
	newKanban  func(s *session) screen                // PT-5f; nil — `v` сообщает «нет вида»
	// Дополнения PT-5a (не меняют контракт). Все nil-безопасны.
	newProjects   func(s *session) screen                           // PT-5b: экран проектов
	newItemScreen func(s *session, it plane.WorkItem) screen        // PT-5c: полная карточка
	pickState     func(s *session, onPick func(plane.State)) screen // PT-5c
	pickPriority  func(s *session, onPick func(string)) screen      // PT-5c
	pickLabel     func(s *session, onPick func(plane.Label)) screen // PT-5c
)

type pushScreenMsg struct{ screen screen }
type popScreenMsg struct{}

func pushScreen(sc screen) tea.Cmd { return func() tea.Msg { return pushScreenMsg{sc} } }
func popScreen() tea.Msg           { return popScreenMsg{} }

// openItem открывает полную карточку; без PT-5c — статус «карточка недоступна».
func openItem(s *session, it plane.WorkItem) tea.Cmd {
	if newItemScreen == nil {
		s.setStatus("карточка недоступна", true)
		return nil
	}
	s.hist.visit(it.ID)
	s.cursor, s.openItem = it.ID, it.ID
	return pushScreen(newItemScreen(s, it))
}

// openItemMsg — сообщение открытия карточки (для Sequence); nil, если карточка недоступна.
func openItemMsg(s *session, it plane.WorkItem) tea.Msg {
	if cmd := openItem(s, it); cmd != nil {
		return cmd()
	}
	return nil
}
