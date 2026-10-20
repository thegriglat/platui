package app

import (
	"context"
	"slices"
	"strings"

	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/search"
)

var groupOrder = []string{"backlog", "unstarted", "started", "completed", "cancelled"}

func groupRank(g string) int {
	return slices.Index(groupOrder, strings.ToLower(g)) // неизвестная → -1, перед backlog
}

// StateTabs — состояния для табов (C1 v3).
func StateTabs(cfg *config.Config, states []plane.State) []plane.State {
	var out []plane.State
	if len(cfg.Tabs.States) > 0 {
		for _, want := range cfg.Tabs.States {
			for _, s := range states {
				if strings.EqualFold(strings.TrimSpace(want), s.Name) {
					out = append(out, s)
					break
				}
			}
		}
		return out
	}
	for _, s := range states {
		hidden := slices.ContainsFunc(cfg.Tabs.HideGroups, func(g string) bool { return strings.EqualFold(g, s.Group) })
		if !hidden {
			out = append(out, s)
		}
	}
	slices.SortStableFunc(out, func(x, y plane.State) int {
		if c := groupRank(x.Group) - groupRank(y.Group); c != 0 {
			return c
		}
		if x.Sequence != y.Sequence {
			if x.Sequence < y.Sequence {
				return -1
			}
			return 1
		}
		return strings.Compare(x.Name, y.Name)
	})
	return out
}

// SearchDocs готовит документы для локального поиска (C9 v4): только ID, ключ и заголовок.
func SearchDocs(m Meta, items []plane.WorkItem) []search.Doc {
	docs := make([]search.Doc, 0, len(items))
	for _, it := range items {
		docs = append(docs, search.Doc{ID: it.ID, Key: m.Project.Identifier + "-" + itoa(it.Sequence), Title: it.Name})
	}
	return docs
}

// ServerSearch — серверный поиск (до 50 результатов); projectID пусто — весь workspace.
func (a *App) ServerSearch(ctx context.Context, query, projectID string) ([]plane.SearchHit, error) {
	return a.d.API.Search(ctx, query, projectID, 50)
}

// Match проверяет задачу по сохранённому фильтру: критерии через И, значения внутри списка — ИЛИ.
// Поле Project фильтра вызывающий применяет сам (выбор проекта).
func Match(f config.Filter, it plane.WorkItem, m Meta, meID string) bool {
	if f.Project != "" && !strings.EqualFold(strings.TrimSpace(f.Project), m.Project.Identifier) {
		return false
	}
	if f.Mine && (meID == "" || !slices.Contains(it.AssigneeIDs, meID)) {
		return false
	}
	if len(f.StateGroups) > 0 || len(f.States) > 0 {
		var st *plane.State
		for i := range m.States {
			if m.States[i].ID == it.StateID {
				st = &m.States[i]
			}
		}
		if st == nil {
			return false
		}
		if len(f.StateGroups) > 0 && !containsFold(f.StateGroups, st.Group) {
			return false
		}
		if len(f.States) > 0 && !containsFold(f.States, st.Name) {
			return false
		}
	}
	if len(f.Priorities) > 0 {
		p := it.Priority
		if p == "" {
			p = "none"
		}
		if !containsFold(f.Priorities, p) {
			return false
		}
	}
	if len(f.Labels) > 0 && !slices.ContainsFunc(m.Labels, func(l plane.Label) bool {
		return slices.Contains(it.LabelIDs, l.ID) && containsFold(f.Labels, l.Name)
	}) {
		return false
	}
	if t := strings.TrimSpace(f.Text); t != "" && !strings.Contains(strings.ToLower(it.Name), strings.ToLower(t)) {
		return false
	}
	return true
}

func containsFold(list []string, s string) bool {
	return slices.ContainsFunc(list, func(x string) bool { return strings.EqualFold(strings.TrimSpace(x), s) })
}
