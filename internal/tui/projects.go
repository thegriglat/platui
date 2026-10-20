package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/plane"
)

func init() { newProjects = func(s *session) screen { return newProjectsScreen(s) } }

// projectsScreen — экран выбора проекта (P).
type projectsScreen struct{ list cursorList }

func newProjectsScreen(s *session) *projectsScreen {
	p := &projectsScreen{}
	p.list.n = len(s.projects)
	for i, pr := range s.projects {
		if s.project != nil && pr.ID == s.project.ID {
			p.list.pos = i
		}
	}
	return p
}

func (p *projectsScreen) Title() string       { return "проекты" }
func (p *projectsScreen) Scope() string       { return "projects" }
func (p *projectsScreen) CapturesInput() bool { return false }

func (p *projectsScreen) Update(s *session, msg tea.Msg) (screen, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		p.list.SetLen(len(s.projects))
		return p, nil
	}
	name := keyName(k)
	if name == "enter" && len(s.projects) > 0 {
		pr := s.projects[p.list.Pos()]
		if s.project != nil && s.project.ID == pr.ID {
			return p, popScreen
		}
		return p, tea.Sequence(popScreen, func() tea.Msg { return openProjectMsg{project: pr} })
	}
	p.list.Key(name)
	return p, nil
}

func (p *projectsScreen) View(s *session) string {
	p.list.SetLen(len(s.projects))
	p.list.SetHeight(max(s.height-4, 1))
	lines := []string{s.st.accent.Bold(true).Render(" Проекты"), ""}
	from, to := p.list.Range()
	for i := from; i < to; i++ {
		lines = append(lines, p.row(s, s.projects[i], i == p.list.Pos()))
	}
	if len(s.projects) == 0 {
		lines = append(lines, s.st.muted.Render("  проектов нет"))
	}
	for len(lines) < s.height-2 {
		lines = append(lines, "")
	}
	lines = append(lines, hintsLine(s, "projects", s.width), s.statusView(s.width))
	return strings.Join(lines[:min(len(lines), s.height)], "\n")
}

func (p *projectsScreen) row(s *session, pr plane.Project, sel bool) string {
	cur := "  "
	if s.project != nil && s.project.ID == pr.ID {
		cur = "✓ "
	}
	line := clip(cur+cell(pr.Identifier, 10)+" "+pr.Name, s.width)
	if sel {
		return s.st.sel.Render(line)
	}
	return line
}
