package app

import (
	"strings"

	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
)

// Prompt — строка для приглашения оболочки/tmux: только кэш и git, без сети.
func (a *App) Prompt() string {
	if a.d.Git == nil {
		return ""
	}
	br, err := a.d.Git.Current()
	if err != nil {
		return ""
	}
	ref, ok := gitbranch.ParseBranch(a.branchTemplate(), br)
	if !ok {
		return ""
	}
	id := ref.Identifier + "-" + itoa(ref.Sequence)
	if st := a.cachedStateName(ref); st != "" {
		return id + " " + st
	}
	return id
}

func (a *App) cachedStateName(ref gitbranch.Item) string {
	a.mu.Lock()
	ps := a.projects
	a.mu.Unlock()
	if ps == nil {
		if _, ok := a.load(a.key("projects"), &ps); !ok {
			return ""
		}
	}
	var pid string
	for _, p := range ps {
		if strings.EqualFold(p.Identifier, ref.Identifier) {
			pid = p.ID
		}
	}
	if pid == "" {
		return ""
	}
	its, _, ok := a.CachedItems(pid)
	if !ok {
		return ""
	}
	a.mu.Lock()
	m, ok := a.meta[pid]
	a.mu.Unlock()
	if !ok {
		if _, ok = a.load(a.key("meta", pid), &m); !ok {
			return ""
		}
	}
	for _, it := range its {
		if it.Sequence == ref.Sequence {
			return stateName(m.States, it)
		}
	}
	return ""
}

func stateName(states []plane.State, it plane.WorkItem) string {
	for _, s := range states {
		if s.ID == it.StateID {
			return s.Name
		}
	}
	return ""
}
