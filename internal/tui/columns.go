package tui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/thegriglat/platui/internal/forge"
	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/search"
)

const (
	colMark   = 2
	colPrio   = 6
	colBranch = 7
	colMR     = 3
	colState  = 12
	colAssign = 9
	titleMin  = 20
)

// colLayout holds table column widths; 0 means the column is hidden.
type colLayout struct{ id, title, state, assign int }

// layoutCols: ID/priority/branch are fixed, the title takes the remainder;
// when space runs out, assignees are hidden first, then state.
func (s *session) layoutCols(width int) colLayout {
	idW := 4
	if s.project != nil {
		maxSeq := 0
		for _, it := range s.items {
			maxSeq = max(maxSeq, it.Sequence)
		}
		idW = max(idW, len([]rune(s.project.Identifier))+1+len(itoa(maxSeq)))
	}
	fixed := colMark + idW + 1 + colPrio + 1 + colBranch + 1 + colMR + 1
	rem := width - fixed - titleMin - 1
	l := colLayout{id: idW}
	switch {
	case rem >= colState+colAssign+2:
		l.state, l.assign = colState, colAssign
	case rem >= colState+1:
		l.state = colState
	}
	used := fixed + 1
	if l.state > 0 {
		used += l.state + 1
	}
	if l.assign > 0 {
		used += l.assign + 1
	}
	l.title = max(width-used, 1)
	return l
}

// cell truncates a cell to w columns and pads it with spaces.
func cell(s string, w int) string {
	s = ansi.Truncate(s, w, "…")
	return s + strings.Repeat(" ", max(0, w-lipgloss.Width(s)))
}

func prioLabel(p string) string {
	switch p {
	case "urgent", "high", "low":
		return p
	case "medium":
		return "med"
	}
	return "-"
}

// initials returns "AL" for "Anna Lee"; a single word gives its first two letters.
func initials(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	var out []rune
	switch {
	case len(words) >= 2:
		out = []rune{[]rune(words[0])[0], []rune(words[1])[0]}
	case len(words) == 1:
		out = []rune(words[0])
		out = out[:min(2, len(out))]
	}
	return strings.ToUpper(string(out))
}

func (s *session) assignees(it plane.WorkItem) string {
	var out []string
	for _, id := range it.AssigneeIDs {
		name := id
		for _, m := range s.meta.Members {
			if m.ID == id {
				name = m.Name()
			}
		}
		if id == s.me.ID && s.me.Name() != "" {
			name = s.me.Name()
		}
		out = append(out, initials(name))
	}
	return strings.Join(out, ",")
}

func (s *session) stateNameCol(id string) string {
	for _, st := range s.meta.States {
		if st.ID == id {
			return st.Name
		}
	}
	return ""
}

// branchMark is the branch indicator: ● local, ↑n ahead, ☁ remote.
func (s *session) branchMark(id string) string {
	b, ok := s.branches[id]
	if !ok {
		return ""
	}
	var parts []string
	if b.Local {
		parts = append(parts, "●")
	}
	if b.Ahead > 0 {
		parts = append(parts, "↑"+itoa(b.Ahead))
	}
	if b.Remote {
		parts = append(parts, "☁")
	}
	return strings.Join(parts, " ")
}

// mrMark is the MR indicator from app.CachedMRs (no glab call): ⇄ open, ✓ merged.
func (s *session) mrMark(id string) string {
	if s.deps.App == nil {
		return ""
	}
	mrs, _ := s.deps.App.CachedMRs(id)
	return mrSign(mrs)
}

func mrSign(mrs []forge.MR) string {
	merged := false
	for _, m := range mrs {
		switch m.State {
		case "opened":
			return "⇄"
		case "merged":
			merged = true
		}
	}
	if merged {
		return "✓"
	}
	return ""
}

// markTitle highlights the matched runes of the title (in brackets when colors are off).
func (s *session) markTitle(it plane.WorkItem) string {
	h, ok := s.hits[it.ID]
	if !ok || len(h.Pos) == 0 {
		return it.Name
	}
	return search.Highlight(it.Name, h.Pos, func(x string) string {
		if s.st.plain {
			return "[" + x + "]"
		}
		return s.st.accent.Bold(true).Underline(true).Render(x)
	})
}

func (s *session) headerRow(l colLayout) string {
	arrow := func(by int, up bool) string {
		if s.sortBy != by {
			return ""
		}
		if up {
			return "▲"
		}
		return "▼"
	}
	title := "Title"
	if a := arrow(sortUpdated, false); a != "" {
		title += " (updated " + a + ")"
	}
	parts := []string{cell("", colMark), cell("ID"+arrow(sortID, true), l.id), cell(title, l.title)}
	if l.state > 0 {
		parts = append(parts, cell("State"+arrow(sortState, true), l.state))
	}
	parts = append(parts, cell("Pri"+arrow(sortPriority, true), colPrio))
	if l.assign > 0 {
		parts = append(parts, cell("Asg", l.assign))
	}
	parts = append(parts, cell("Branch", colBranch), cell("MR", colMR))
	return s.st.muted.Bold(true).Render(strings.Join(parts, " "))
}

// itemRow renders a table row; sel marks the row under the cursor.
func (s *session) itemRow(r row, l colLayout, sel bool) string {
	it := r.it
	mark := "  "
	_, fresh := s.fresh[it.ID]
	switch {
	case s.marks[it.ID]:
		mark = "* "
	case fresh:
		mark = "+ "
	}
	if r.pinned {
		mark = mark[:1] + "☆"
	}
	idStyle := s.st.muted
	if fresh {
		idStyle = s.st.warn.Bold(true)
	}
	parts := []string{mark, idStyle.Render(cell(itoaKey(s, it.Sequence), l.id)), cell(r.treePrefix()+s.markTitle(it), l.title)}
	if l.state > 0 {
		parts = append(parts, s.st.muted.Render(cell(s.stateNameCol(it.StateID), l.state)))
	}
	parts = append(parts, s.st.priority(it.Priority).Render(cell(prioLabel(it.Priority), colPrio)))
	if l.assign > 0 {
		parts = append(parts, cell(s.assignees(it), l.assign))
	}
	parts = append(parts, cell(s.branchMark(it.ID), colBranch), cell(s.mrMark(it.ID), colMR))
	line := strings.Join(parts, " ")
	if sel {
		return s.st.sel.Render(line)
	}
	return line
}
