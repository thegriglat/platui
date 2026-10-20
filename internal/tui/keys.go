package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// keyName — нормализованное имя клавиши (tea.KeyMsg.String(); пробел — "space").
func keyName(m tea.KeyMsg) string {
	if s := m.String(); s != " " {
		return s
	}
	return "space"
}

var keyLabels = map[string]string{
	"up": "↑", "down": "↓", "left": "←", "right": "→", "pgup": "PgUp", "pgdown": "PgDn",
	"enter": "Enter", "esc": "Esc", "tab": "Tab", "home": "Home", "end": "End", "space": "Space",
}

// keyLabel — подпись клавиши для справки: "ctrl+left" → "Ctrl+←".
func keyLabel(k string) string {
	parts := strings.Split(k, "+")
	for i, p := range parts {
		switch {
		case keyLabels[p] != "":
			parts[i] = keyLabels[p]
		case len(parts) > 1 && i < len(parts)-1:
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		case p == "shift":
			parts[i] = "Shift"
		}
	}
	return strings.Join(parts, "+")
}

// keysLabel — подпись набора клавиш; alt+1…alt+9 сворачиваются в «Alt+1…9».
func keysLabel(keys []string) string {
	var out []string
	digits := 0
	for _, k := range keys {
		if len(k) == 5 && strings.HasPrefix(k, "alt+") && k[4] >= '1' && k[4] <= '9' {
			digits++
			continue
		}
		out = append(out, keyLabel(k))
	}
	if digits == 9 {
		out = append(out, "Alt+1…9")
	} else {
		for _, k := range keys {
			if len(k) == 5 && strings.HasPrefix(k, "alt+") && k[4] >= '1' && k[4] <= '9' {
				out = append(out, keyLabel(k))
			}
		}
	}
	return strings.Join(out, " ")
}
