package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
)

type itemsMsg struct {
	projectID         string
	items             []plane.WorkItem
	changes           app.Changes
	fromCache, reload bool
	err               error
}
type metaMsg struct {
	meta app.Meta
	err  error
}
type itemMsg struct {
	item plane.WorkItem
	err  error
}
type commentsMsg struct {
	itemID   string
	comments []plane.Comment
	err      error
}
type branchesMsg struct {
	m   map[string]gitbranch.BranchStatus
	err error
}
type statusMsg struct {
	text  string
	isErr bool
}
type pollMsg struct{}

// bootMsg — итог старта: пользователь, проекты, выбранный проект и задача.
type bootMsg struct {
	me       plane.Member
	projects []plane.Project
	project  *plane.Project
	itemID   string
	warn     error
	err      error
	offline  bool // данные из кэша (app.Offline)
	offAt    time.Time
}

// openProjectMsg — переключение проекта (itemID — задача для курсора).
type openProjectMsg struct {
	project plane.Project
	itemID  string
}

func errStatus(err error) tea.Msg { return statusMsg{err.Error(), true} }

func bootstrap(s *session) tea.Cmd {
	a, cfg, start := s.deps.App, s.deps.Config, s.deps.Start
	return func() tea.Msg {
		ctx := context.Background()
		me, err := a.Me(ctx)
		if err != nil {
			return bootMsg{err: err}
		}
		ps, err := a.Projects(ctx)
		if err != nil {
			return bootMsg{me: me, err: err}
		}
		b := bootMsg{me: me, projects: ps}
		var p *plane.Project
		if start.Item != "" {
			def := start.Project
			if def == "" {
				def = cfg.DefaultProject
			}
			if id, seq, ok := app.ParseRef(start.Item, def); ok {
				pr, it, err := a.Find(ctx, id, seq)
				switch {
				case err != nil:
					b.warn = err
				case start.Project == "" || strings.EqualFold(pr.Identifier, start.Project):
					p, b.itemID = &pr, it.ID
				}
			}
		} else if pr, it, err := a.CurrentItem(ctx); err == nil &&
			(start.Project == "" || strings.EqualFold(pr.Identifier, start.Project)) {
			p, b.itemID = &pr, it.ID
		}
		if p == nil {
			ident := start.Project
			if ident == "" {
				ident = cfg.DefaultProject
			}
			if ident != "" {
				if pr, err := a.Project(ctx, ident); err != nil {
					b.warn = err
				} else {
					p = &pr
				}
			}
		}
		b.project = p
		b.offline, b.offAt = a.Offline()
		return b
	}
}

func loadMeta(s *session) tea.Cmd {
	if s.project == nil {
		return nil
	}
	a, pid := s.deps.App, s.project.ID
	return func() tea.Msg {
		m, err := a.Meta(context.Background(), pid)
		return metaMsg{meta: m, err: err}
	}
}

// loadItems: useCache — сначала кэш (fromCache), затем обновление из API.
func loadItems(s *session, useCache bool) tea.Cmd {
	if s.project == nil {
		return nil
	}
	a, pid := s.deps.App, s.project.ID
	refresh := func() tea.Msg {
		its, ch, err := a.RefreshItems(context.Background(), pid)
		return itemsMsg{projectID: pid, items: its, changes: ch, err: err}
	}
	if !useCache {
		return refresh
	}
	cached := func() tea.Msg {
		its, _, ok := a.CachedItems(pid)
		if !ok {
			return nil
		}
		return itemsMsg{projectID: pid, items: its, fromCache: true}
	}
	return tea.Batch(cached, refresh)
}

// reload — app.Reload в обход кэша: metaMsg + itemsMsg{reload:true}.
func reload(s *session) tea.Cmd {
	if s.project == nil {
		return nil
	}
	a, pid := s.deps.App, s.project.ID
	return func() tea.Msg {
		m, its, ch, err := a.Reload(context.Background(), pid)
		if err != nil {
			return itemsMsg{projectID: pid, reload: true, err: err}
		}
		return tea.BatchMsg{
			func() tea.Msg { return metaMsg{meta: m} },
			func() tea.Msg { return itemsMsg{projectID: pid, items: its, changes: ch, reload: true} },
		}
	}
}

func loadBranches(s *session) tea.Cmd {
	if s.project == nil {
		return nil
	}
	a, p, items := s.deps.App, *s.project, s.items
	return func() tea.Msg {
		m, err := a.BranchStatuses(p, items)
		return branchesMsg{m: m, err: err}
	}
}

// loadItem перечитывает задачу (app.Item) → itemMsg.
func loadItem(s *session, pid, id string) tea.Cmd {
	a := s.deps.App
	return func() tea.Msg {
		it, err := a.Item(context.Background(), pid, id)
		return itemMsg{item: it, err: err}
	}
}

// loadComments загружает комментарии (app.Comments) → commentsMsg.
func loadComments(s *session, pid, id string) tea.Cmd {
	a := s.deps.App
	return func() tea.Msg {
		cs, err := a.Comments(context.Background(), pid, id)
		return commentsMsg{itemID: id, comments: cs, err: err}
	}
}

// patchItem обновляет задачу (app.Update) → itemMsg с новой задачей; ошибка → statusMsg.
func patchItem(s *session, id string, p plane.WorkItemPatch) tea.Cmd {
	if s.project == nil {
		return nil
	}
	a, pid := s.deps.App, s.project.ID
	return func() tea.Msg {
		it, err := a.Update(context.Background(), pid, id, p)
		if err != nil {
			return statusMsg{err.Error(), true}
		}
		return itemMsg{item: it}
	}
}
