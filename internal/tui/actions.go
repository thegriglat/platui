package tui

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// action is a user action bound to keys. Hint is the label in the hints line:
// "text" (the key is the first of Keys) or "KEYS|text" (keys given explicitly).
type action struct {
	ID     string
	Title  string
	Group  string
	Keys   []string
	Hint   string
	Scopes []string // empty: all scopes
	Bulk   bool     // acts on marked items
	Run    func(m *model) tea.Cmd
}

var groupOrder = []string{"navigation", "search", "task", "git", "views", "kanban"}

var registry []action

// registerAction is called from init(); a duplicate ID or key conflict panics.
func registerAction(a action) {
	if err := checkActions(append(slices.Clone(registry), a)); err != nil {
		panic(err)
	}
	registry = append(registry, a)
}

func scopesOverlap(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return true
	}
	return slices.ContainsFunc(a, func(s string) bool { return slices.Contains(b, s) })
}

// checkActions checks that IDs are unique and keys do not conflict in shared scopes
// (a Bulk/non-Bulk pair is allowed).
func checkActions(list []action) error {
	ids := map[string]bool{}
	for i, a := range list {
		if ids[a.ID] {
			return fmt.Errorf("action %q: duplicate ID", a.ID)
		}
		ids[a.ID] = true
		for _, b := range list[:i] {
			if a.Bulk != b.Bulk && !(a.Bulk && b.Bulk) && scopesOverlap(a.Scopes, b.Scopes) {
				continue
			}
			for _, k := range a.Keys {
				if slices.Contains(b.Keys, k) && scopesOverlap(a.Scopes, b.Scopes) {
					return fmt.Errorf("key %q: conflict between %s and %s", k, a.ID, b.ID)
				}
			}
		}
	}
	return nil
}

func (a action) in(scope string) bool { return len(a.Scopes) == 0 || slices.Contains(a.Scopes, scope) }

// actionsFor returns the scope's actions ordered by group, then by registration.
func actionsFor(scope string) []action {
	var out []action
	for _, a := range registry {
		if a.in(scope) {
			out = append(out, a)
		}
	}
	slices.SortStableFunc(out, func(x, y action) int {
		return slices.Index(groupOrder, x.Group) - slices.Index(groupOrder, y.Group)
	})
	return out
}

// findAction returns the action for a key; of a Bulk/non-Bulk pair it picks by whether items are marked.
func findAction(scope, key string, marked bool) (action, bool) {
	var found []action
	for _, a := range registry {
		if a.in(scope) && slices.Contains(a.Keys, key) {
			found = append(found, a)
		}
	}
	for _, a := range found {
		if a.Bulk == marked {
			return a, true
		}
	}
	if len(found) > 0 {
		return found[0], true
	}
	return action{}, false
}

// hintsLine renders the scope's hints line up to width; "? help" always comes last.
// Keys are highlighted, the line has the bar background.
func hintsLine(s *session, scope string, width int) string {
	type hint struct{ key, text string }
	var parts []hint
	var help *hint
	for _, a := range actionsFor(scope) {
		if a.Hint == "" {
			continue
		}
		h := hint{text: a.Hint}
		if k, t, ok := strings.Cut(a.Hint, "|"); ok {
			h = hint{k, t}
		} else if len(a.Keys) > 0 {
			h.key = keyLabel(a.Keys[0])
		}
		if a.ID == "view.help" {
			help = &h
		} else {
			parts = append(parts, h)
		}
	}
	plain := func(h hint) string { return strings.TrimSpace(h.key + " " + h.text) }
	const sep = "  "
	used := 1
	if help != nil {
		used += len([]rune(sep + plain(*help)))
	}
	var shown []hint
	for _, h := range parts {
		n := len([]rune(sep + plain(h)))
		if len(shown) == 0 {
			n -= len(sep)
		}
		if used+n > width {
			break
		}
		used += n
		shown = append(shown, h)
	}
	if help != nil {
		shown = append(shown, *help)
	}
	var b strings.Builder
	b.WriteString(s.st.hintText.Render(" "))
	for i, h := range shown {
		if i > 0 {
			b.WriteString(s.st.hintText.Render(sep))
		}
		if h.key != "" {
			b.WriteString(s.st.hintKey.Render(h.key))
			b.WriteString(s.st.hintText.Render(" "))
		}
		b.WriteString(s.st.hintText.Render(h.text))
	}
	line := ansi.Truncate(b.String(), width, "")
	return line + s.st.hintText.Render(strings.Repeat(" ", max(0, width-lipgloss.Width(line))))
}
