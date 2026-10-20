package tui

import (
	"time"

	"errors"
	"fmt"
	"github.com/thegriglat/platui/internal/plane"
	"slices"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/app"
)

// handle processes load messages; others are ignored.
func (s *session) handle(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case bootMsg:
		return s.onBoot(msg)
	case openProjectMsg:
		return s.openProject(msg.project, msg.itemID)
	case metaMsg:
		if msg.err != nil {
			s.setStatus(s.errText(msg.err), true)
			return nil
		}
		if s.project != nil && msg.meta.Project.ID == s.project.ID {
			s.meta = msg.meta
			s.tabs = app.StateTabs(s.deps.Config, msg.meta.States)
			s.setTab(s.tab)
		}
	case itemsMsg:
		return s.onItems(msg)
	case branchesMsg:
		if msg.err == nil {
			s.branches = msg.m
		} else if !errors.Is(msg.err, app.ErrNoRepo) {
			s.setStatus(msg.err.Error(), true)
		}
	case itemMsg:
		if msg.err != nil {
			s.setStatus(msg.err.Error(), true)
		} else if i := slices.IndexFunc(s.items, func(x plane.WorkItem) bool { return x.ID == msg.item.ID }); i >= 0 {
			s.items = slices.Clone(s.items)
			s.items[i] = msg.item
			s.rebuildIndex()
		}
	case commentsMsg:
		if msg.err != nil {
			s.setStatus(msg.err.Error(), true)
		}
	case statusMsg:
		s.setStatus(msg.text, msg.isErr)
	case editFailMsg:
		s.onEditFail(msg)
	case pollMsg:
		if s.project == nil || s.polling {
			return nil
		}
		s.polling = true
		return loadItems(s, false)
	}
	return nil
}

func (s *session) onBoot(b bootMsg) tea.Cmd {
	s.me, s.projects = b.me, b.projects
	if b.offline {
		s.net.offline, s.net.offAt = true, b.offAt
		s.setStatus(s.net.offlineText(), false)
	}
	switch {
	case b.err != nil:
		s.setStatus(s.errText(b.err), true)
		return nil
	case b.warn != nil:
		s.setStatus(b.warn.Error(), true)
	}
	if b.project != nil {
		return s.openProject(*b.project, b.itemID)
	}
	if newProjects != nil {
		return pushScreen(newProjects(s))
	}
	if b.warn == nil {
		s.setStatus("no project selected", false)
	}
	return nil
}

func (s *session) openProject(p plane.Project, itemID string) tea.Cmd {
	s.resetProject(p)
	s.pending = itemID
	return tea.Batch(loadMeta(s), loadItems(s, true))
}

func (s *session) onItems(msg itemsMsg) tea.Cmd {
	if s.project == nil || msg.projectID != s.project.ID {
		return nil
	}
	if msg.fromCache && s.live {
		return nil
	}
	var cmds []tea.Cmd
	if !msg.fromCache {
		s.live, s.loading = true, false
		wasPoll := s.polling
		s.polling = false
		cmds = append(cmds, s.schedulePoll(msg.err, wasPoll))
		if msg.err != nil {
			s.setStatus(s.errText(msg.err), true)
			return tea.Batch(cmds...)
		}
		s.markFresh(msg.changes, wasPoll)
		back := s.restored()
		switch {
		case msg.reload:
			t := fmt.Sprintf("updated: %d tasks, %s", len(msg.items), s.deps.Now().Format("15:04"))
			if back {
				t = "connection restored; " + t
			}
			s.setStatus(t, false)
		case wasPoll && !msg.changes.Empty():
			t := fmt.Sprintf("changes: +%d new, %d updated, %d removed",
				len(msg.changes.New), len(msg.changes.Updated), len(msg.changes.Removed))
			if back {
				t = "connection restored; " + t
			}
			s.setStatus(t, false)
		case back:
			s.setStatus("connection restored", false)
		}
	}
	s.items = msg.items
	s.rebuildIndex()
	if s.pending != "" {
		if it, ok := s.itemByID(s.pending); ok {
			s.cursor, s.pending = it.ID, ""
			if i := slices.IndexFunc(s.tabList(), func(t tabInfo) bool { return t.state != nil && t.state.ID == it.StateID }); i >= 0 && !s.deps.Config.TabsAll() {
				s.tab = i
			}
			cmds = append(cmds, openItem(s, it))
		}
	}
	return tea.Batch(append(cmds, loadBranches(s))...)
}

func (s *session) markFresh(ch app.Changes, notify bool) {
	now := s.deps.Now()
	for id, t := range s.fresh {
		if now.Sub(t) > freshTTL {
			delete(s.fresh, id)
		}
	}
	for _, it := range slices.Concat(ch.New, ch.Updated) {
		s.fresh[it.ID] = now
	}
	if notify && s.deps.Config.NotifyEnabled() && (len(ch.New) > 0 || len(ch.Updated) > 0) {
		s.bell()
	}
}

// schedulePoll schedules the next poll; the chain is created once.
func (s *session) schedulePoll(err error, wasPoll bool) tea.Cmd {
	base := s.deps.Config.RefreshInterval()
	if base <= 0 || (s.pollArmed && !wasPoll) {
		return nil
	}
	s.pollArmed = true
	s.pollEvery = app.NextPoll(base, s.pollEvery, err)
	return tea.Tick(s.pollEvery, func(time.Time) tea.Msg { return pollMsg{} })
}
