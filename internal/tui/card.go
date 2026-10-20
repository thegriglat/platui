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

// commentsEntry holds a task's loaded comments.
type commentsEntry struct {
	list []plane.Comment
	err  error
}

// cardView renders a card for both the preview and the full screen: scrolling, Markdown cache, comments.
type cardView struct {
	vp       viewport.Model
	md       map[string]string // glamour cache by (key, width)
	comments map[string]commentsEntry
	stats    map[string]statEntry
	shown    string // ID of the shown task (scrolls to the top on change)
}

func newCardView() *cardView {
	vp := viewport.New(0, 0)
	vp.KeyMap = viewport.KeyMap{
		PageDown: key.NewBinding(key.WithKeys("pgdown")), PageUp: key.NewBinding(key.WithKeys("pgup")),
		Down: key.NewBinding(key.WithKeys("down")), Up: key.NewBinding(key.WithKeys("up")),
	}
	return &cardView{vp: vp, md: map[string]string{}, comments: map[string]commentsEntry{}, stats: map[string]statEntry{}}
}

// scroll handles a scroll key; false means it is not a scroll key.
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

// renderMD converts Markdown to terminal text (glamour, word-wrapped to width; notty style without colors); cached by key.
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

// highlight highlights the runes of name at positions pos.
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

// header returns the task title and metadata lines.
func (c *cardView) header(s *session, it plane.WorkItem, width int) []string {
	key := itoaKey(s, it.Sequence)
	title := s.st.bold.Render(key) + " " + highlight(s, it.Name, s.hits[it.ID].Pos)
	var assignees []string
	for _, id := range it.AssigneeIDs {
		assignees = append(assignees, s.memberName(id))
	}
	meta := []string{s.stateName(it.StateID), s.st.priority(it.Priority).Render(orDash(it.Priority))}
	meta = append(meta, "assignee: "+orDash(strings.Join(assignees, ", ")))
	lines := []string{title, s.st.muted.Render(strings.Join(meta, " · "))}
	if ls := s.labelNames(it); len(ls) > 0 {
		lines = append(lines, s.st.muted.Render("labels: "+strings.Join(ls, ", ")))
	}
	if b, ok := s.branches[it.ID]; ok && b.Name != "" && (b.Local || b.Remote) {
		lines = append(lines, s.st.muted.Render("branch: "+b.Name))
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

// body returns the full card text: header, description, comments.
func (c *cardView) body(s *session, it plane.WorkItem, width int) string {
	lines := c.header(s, it, width)
	if strings.TrimSpace(it.DescHTML) == "" {
		lines = append(lines, s.st.muted.Render("(no description)"))
	} else {
		k := fmt.Sprintf("d|%s|%d|%s", it.ID, it.UpdatedAt.UnixNano(), it.DescHTML)
		lines = append(lines, c.renderMD(s, k, text.HTMLToMarkdown(it.DescHTML), width))
	}
	lines = append(lines, "", s.st.accent.Render("Comments"))
	e, ok := c.comments[it.ID]
	switch {
	case !ok:
		lines = append(lines, s.st.muted.Render("loading…"))
	case e.err != nil:
		lines = append(lines, s.st.err.Render("failed to load: "+e.err.Error()))
	case len(e.list) == 0:
		lines = append(lines, s.st.muted.Render("(no comments)"))
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

// view renders the card of task id in a width×height area; shows a hint if there is no task.
func (c *cardView) view(s *session, id string, width, height int) string {
	it, ok := s.itemByID(id)
	if !ok {
		return s.st.muted.Render("no task")
	}
	if c.shown != id {
		c.shown = id
		c.vp.GotoTop()
	}
	c.vp.Width, c.vp.Height = width, max(height, 1)
	c.vp.SetContent(c.body(s, it, width))
	return c.vp.View()
}

// onComments stores the comments from a commentsMsg.
func (c *cardView) onComments(m commentsMsg) {
	c.comments[m.itemID] = commentsEntry{list: m.comments, err: m.err}
}
