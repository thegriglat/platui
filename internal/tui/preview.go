package tui

import (
	"strings"

	"github.com/thegriglat/platui/internal/plane"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// previewTick — период проверки «какая карточка нужна» (отложенная загрузка комментариев).
const previewTick = 250 * time.Millisecond

type previewTickMsg struct{}

func tickPreview() tea.Cmd {
	return tea.Tick(previewTick, func(time.Time) tea.Msg { return previewTickMsg{} })
}

func init() {
	newPreview = func(*session) pane { return &preview{card: newCardView()} }
	newItemScreen = func(s *session, it plane.WorkItem) screen { return &itemScreen{card: newCardView(), id: it.ID} }
}

// preview — панель превью карточки под курсором; она же загружает комментарии
// для курсора и открытой карточки (после того как выбор не менялся один тик).
type preview struct {
	card      *cardView
	armed     bool
	prev      string // выбор на прошлом тике
	requested string // для кого уже запрошены комментарии
	openSeen  string // открытая карточка, замеченная на прошлом тике
}

func wantedItem(s *session) string {
	if s.openItem != "" {
		return s.openItem
	}
	return s.cursor
}

func (p *preview) Update(s *session, msg tea.Msg) (pane, tea.Cmd) {
	var cmds []tea.Cmd
	if !p.armed {
		p.armed = true
		cmds = append(cmds, tickPreview())
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		p.card.scroll(keyName(msg))
	case commentsMsg:
		p.card.onComments(msg)
	case diffStatMsg:
		p.card.onStat(msg)
	case itemsMsg:
		if msg.reload && !msg.fromCache {
			p.requested = ""
		}
	case editDoneMsg:
		cmds = append(cmds, finishEdit(s, msg))
	case previewTickMsg:
		cmds = append(cmds, tickPreview(), p.load(s))
	}
	return p, tea.Batch(cmds...)
}

// load запрашивает комментарии (и свежую задачу для открытой карточки), когда выбор стабилен.
func (p *preview) load(s *session) tea.Cmd {
	if s.openItem != p.openSeen {
		p.openSeen, p.requested = s.openItem, ""
	}
	id := wantedItem(s)
	stable := id == p.prev || id == s.openItem
	p.prev = id
	if id == "" || s.project == nil || !stable {
		return nil
	}
	cmds := []tea.Cmd{p.card.wantStat(s, id)}
	if id == p.requested {
		return tea.Batch(cmds...)
	}
	p.requested = id
	cmds = append(cmds, loadComments(s, s.project.ID, id))
	if id == s.openItem {
		cmds = append(cmds, loadItem(s, s.project.ID, id))
	}
	return tea.Batch(cmds...)
}

func (p *preview) View(s *session, width, height int) string {
	if len(s.visible()) == 0 && s.cursor == "" {
		return s.st.muted.Render("нет задачи")
	}
	return p.card.view(s, s.cursor, width, height)
}

// itemScreen — полная карточка на весь экран.
type itemScreen struct {
	card *cardView
	id   string
}

func (c *itemScreen) Title() string       { return "карточка" }
func (c *itemScreen) Scope() string       { return "item" }
func (c *itemScreen) CapturesInput() bool { return false }

func (c *itemScreen) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		c.card.scroll(keyName(msg))
	case commentsMsg:
		c.card.onComments(msg)
	case diffStatMsg:
		c.card.onStat(msg)
	case previewTickMsg:
		return c, c.card.wantStat(s, c.id)
	}
	return c, nil
}

func (c *itemScreen) View(s *session) string {
	head := s.st.accent.Bold(true).Render(" Карточка")
	h := max(s.height-3, 1)
	body := fitBlock(c.card.view(s, c.id, s.width, h), s.width, h)
	out := append([]string{head}, body...)
	out = append(out, hintsLine(s, "item", s.width), s.statusView(s.width))
	return strings.Join(out, "\n")
}
