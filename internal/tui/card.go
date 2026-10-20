package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/text"
)

// commentsEntry — загруженные комментарии задачи.
type commentsEntry struct {
	list []plane.Comment
	err  error
}

// cardView — общий рендер карточки для превью и полного экрана: прокрутка, кэш Markdown, комментарии.
type cardView struct {
	vp       viewport.Model
	md       map[string]string // кэш glamour по (ключ, ширина)
	comments map[string]commentsEntry
	stats    map[string]statEntry
	shown    string // ID показанной задачи (при смене — прокрутка в начало)
}

func newCardView() *cardView {
	vp := viewport.New(0, 0)
	vp.KeyMap = viewport.KeyMap{
		PageDown: key.NewBinding(key.WithKeys("pgdown")), PageUp: key.NewBinding(key.WithKeys("pgup")),
		Down: key.NewBinding(key.WithKeys("down")), Up: key.NewBinding(key.WithKeys("up")),
	}
	return &cardView{vp: vp, md: map[string]string{}, comments: map[string]commentsEntry{}, stats: map[string]statEntry{}}
}

// scroll обрабатывает клавишу прокрутки; false — не прокрутка.
func (c *cardView) scroll(k string) bool {
	switch k {
	case "up":
		c.vp.ScrollUp(1)
	case "down":
		c.vp.ScrollDown(1)
	case "pgup":
		c.vp.PageUp()
	case "pgdown":
		c.vp.PageDown()
	case "home":
		c.vp.GotoTop()
	case "end":
		c.vp.GotoBottom()
	default:
		return false
	}
	return true
}

// renderMD — Markdown → текст терминала (glamour, WordWrap по ширине; без цветов — notty); кэш по key.
func (c *cardView) renderMD(s *session, key, md string, width int) string {
	ck := fmt.Sprintf("%s|%d|%t", key, width, s.st.plain)
	if v, ok := c.md[ck]; ok {
		return v
	}
	style := "dark"
	if s.st.plain {
		style = "notty"
	}
	out := md
	if r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(max(width-2, 10))); err == nil {
		if o, err := r.Render(md); err == nil {
			out = o
		}
	}
	out = strings.Trim(out, "\n")
	if len(c.md) > 200 {
		c.md = map[string]string{}
	}
	c.md[ck] = out
	return out
}

// highlight подсвечивает руны name на позициях pos.
func highlight(s *session, name string, pos []int) string {
	if len(pos) == 0 {
		return name
	}
	var b strings.Builder
	for i, r := range []rune(name) {
		if slices.Contains(pos, i) {
			b.WriteString(s.st.accent.Bold(true).Underline(true).Render(string(r)))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *session) stateName(id string) string {
	for _, st := range s.meta.States {
		if st.ID == id {
			return st.Name
		}
	}
	return id
}

func (s *session) memberName(id string) string {
	for _, m := range s.meta.Members {
		if m.ID == id {
			return m.Name()
		}
	}
	return id
}

func (s *session) labelNames(it plane.WorkItem) []string {
	var out []string
	for _, id := range it.LabelIDs {
		name := id
		for _, l := range s.meta.Labels {
			if l.ID == id {
				name = l.Name
			}
		}
		out = append(out, name)
	}
	return out
}

// header — заголовок и метаданные задачи.
func (c *cardView) header(s *session, it plane.WorkItem, width int) []string {
	key := itoaKey(s, it.Sequence)
	title := s.st.bold.Render(key) + " " + highlight(s, it.Name, s.hits[it.ID].Pos)
	var assignees []string
	for _, id := range it.AssigneeIDs {
		assignees = append(assignees, s.memberName(id))
	}
	meta := []string{s.stateName(it.StateID), s.st.priority(it.Priority).Render(orDash(it.Priority))}
	meta = append(meta, "исп.: "+orDash(strings.Join(assignees, ", ")))
	lines := []string{title, s.st.muted.Render(strings.Join(meta, " · "))}
	if ls := s.labelNames(it); len(ls) > 0 {
		lines = append(lines, s.st.muted.Render("метки: "+strings.Join(ls, ", ")))
	}
	if b, ok := s.branches[it.ID]; ok && b.Name != "" && (b.Local || b.Remote) {
		lines = append(lines, s.st.muted.Render("ветка: "+b.Name))
	}
	if l := c.statLine(it); l != "" {
		lines = append(lines, s.st.muted.Render(l))
	}
	if t, ok := s.timerOf(it.ID); ok {
		lines = append(lines, s.st.warn.Render("⏱ "+app.FormatDuration(s.deps.Now().Sub(t))))
	}
	return append(lines, s.st.border.Render(strings.Repeat("─", max(width, 1))))
}

func orDash(v string) string {
	if v == "" {
		return "—"
	}
	return v
}

// body — полный текст карточки: заголовок, описание, комментарии.
func (c *cardView) body(s *session, it plane.WorkItem, width int) string {
	lines := c.header(s, it, width)
	if strings.TrimSpace(it.DescHTML) == "" {
		lines = append(lines, s.st.muted.Render("(нет описания)"))
	} else {
		k := fmt.Sprintf("d|%s|%d|%s", it.ID, it.UpdatedAt.UnixNano(), it.DescHTML)
		lines = append(lines, c.renderMD(s, k, text.HTMLToMarkdown(it.DescHTML), width))
	}
	lines = append(lines, "", s.st.accent.Render("Комментарии"))
	e, ok := c.comments[it.ID]
	switch {
	case !ok:
		lines = append(lines, s.st.muted.Render("загрузка…"))
	case e.err != nil:
		lines = append(lines, s.st.err.Render("не загружены: "+e.err.Error()))
	case len(e.list) == 0:
		lines = append(lines, s.st.muted.Render("(нет комментариев)"))
	}
	for _, cm := range e.list {
		author := cm.Author
		for _, m := range s.meta.Members {
			if m.ID == cm.Author {
				author = m.Name()
			}
		}
		lines = append(lines, "", s.st.bold.Render(author)+s.st.muted.Render(" · "+cm.CreatedAt.Format("2006-01-02 15:04")),
			c.renderMD(s, "c|"+cm.ID, text.HTMLToMarkdown(cm.HTML), width))
	}
	return strings.Join(lines, "\n")
}

// view рисует карточку задачи id в области width×height; нет задачи — подсказка.
func (c *cardView) view(s *session, id string, width, height int) string {
	it, ok := s.itemByID(id)
	if !ok {
		return s.st.muted.Render("нет задачи")
	}
	if c.shown != id {
		c.shown = id
		c.vp.GotoTop()
	}
	c.vp.Width, c.vp.Height = width, max(height, 1)
	c.vp.SetContent(c.body(s, it, width))
	return c.vp.View()
}

// onComments запоминает комментарии из commentsMsg.
func (c *cardView) onComments(m commentsMsg) {
	c.comments[m.itemID] = commentsEntry{list: m.comments, err: m.err}
}
