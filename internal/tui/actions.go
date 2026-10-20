package tui

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// action — действие с привязкой клавиш. Hint — подпись в строке подсказок:
// «текст» (клавиша — первая из Keys) или «КЛАВИШИ|текст» (клавиши заданы явно).
type action struct {
	ID     string
	Title  string
	Group  string
	Keys   []string
	Hint   string
	Scopes []string // пусто — везде
	Bulk   bool     // действие над отмеченными
	Run    func(m *model) tea.Cmd
}

var groupOrder = []string{"навигация", "поиск", "задача", "git", "виды", "канбан"}

var registry []action

// registerAction вызывается из init(); дубль ID или конфликт клавиш — panic.
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

// checkActions проверяет уникальность ID и отсутствие конфликтов клавиш в общих областях
// (пара Bulk/не-Bulk допустима).
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

// actionsFor — действия области в порядке групп, затем регистрации.
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

// findAction — действие по клавише; из пары Bulk/не-Bulk выбирается по наличию отметок.
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

// hintsLine — строка подсказок области до ширины width; «? справка» всегда последняя.
func hintsLine(s *session, scope string, width int) string {
	var parts []string
	help := ""
	for _, a := range actionsFor(scope) {
		if a.Hint == "" {
			continue
		}
		text := a.Hint
		if k, t, ok := strings.Cut(a.Hint, "|"); ok {
			text = k + " " + t
		} else if len(a.Keys) > 0 {
			text = keyLabel(a.Keys[0]) + " " + a.Hint
		}
		if a.ID == "view.help" {
			help = text
		} else {
			parts = append(parts, text)
		}
	}
	sep := "  "
	out := ""
	for _, p := range parts {
		next := out + sep + p
		if help != "" {
			next += sep + help
		}
		if len([]rune(next)) > width {
			break
		}
		out += sep + p
	}
	if help != "" {
		out += sep + help
	}
	return s.st.muted.Render(clip(strings.TrimPrefix(out, sep), width))
}
