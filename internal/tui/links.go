package tui

import (
	"context"
	"slices"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/text"
)

func init() {
	registerAction(action{ID: "item.links", Title: "Links from description and comments", Group: "task",
		Keys: []string{"l"}, Hint: "links", Scopes: cardScopes,
		Run: withTarget(func(m *model, it plane.WorkItem) tea.Cmd {
			s := m.s
			a, pid := s.deps.App, s.project.ID
			return func() tea.Msg {
				links := text.ExtractLinks(it.DescHTML)
				cs, err := a.Comments(context.Background(), pid, it.ID)
				for _, c := range cs {
					links = append(links, text.ExtractLinks(c.HTML)...)
				}
				var uniq []string
				for _, l := range links {
					if !slices.Contains(uniq, l) {
						uniq = append(uniq, l)
					}
				}
				switch {
				case len(uniq) > 0:
					return pushScreenMsg{&linksScreen{links: uniq, list: cursorList{n: len(uniq), h: 12}}}
				case err != nil:
					return statusMsg{"no links (comments not loaded: " + err.Error() + ")", true}
				}
				return statusMsg{"no links", false}
			}
		})})
}

// linksScreen is the list of links; Enter opens the selected one in the browser, Esc closes.
type linksScreen struct {
	links []string
	list  cursorList
}

func (c *linksScreen) Title() string       { return "links" }
func (c *linksScreen) Scope() string       { return "dialog" }
func (c *linksScreen) CapturesInput() bool { return true }

func (c *linksScreen) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return c, nil
	}
	switch n := keyName(k); n {
	case "esc", "q":
		return c, popScreen
	case "enter":
		url := c.links[c.list.Pos()]
		return c, func() tea.Msg {
			if err := s.deps.OpenURL(url); err != nil {
				return statusMsg{"browser: " + err.Error(), true}
			}
			return statusMsg{"opened: " + url, false}
		}
	default:
		c.list.Key(n)
	}
	return c, nil
}

func (c *linksScreen) View(s *session) string {
	rows := []string{s.st.accent.Bold(true).Render(" Links (Enter — open in browser, Esc — close)")}
	c.list.SetHeight(max(s.height-3, 1))
	from, to := c.list.Range()
	for i := from; i < to; i++ {
		line := "  " + c.links[i]
		if i == c.list.Pos() {
			line = s.st.sel.Render(clip("> "+c.links[i], s.width))
		} else {
			line = clip(line, s.width)
		}
		rows = append(rows, line)
	}
	for len(rows) < s.height-1 {
		rows = append(rows, "")
	}
	return clip2(append(rows, s.statusView(s.width)), s.height)
}
