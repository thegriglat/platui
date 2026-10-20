package gitbranch

import "strings"

var translit = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "yo",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "shch",
	'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

// Slug converts a title to a branch-safe slug of at most max characters
// (max <= 0 means 40). An empty result is "task".
func Slug(title string, max int) string {
	if max <= 0 {
		max = 40
	}
	var b strings.Builder
	lastDash := true // suppress leading and repeated dashes
	emit := func(s string) {
		for _, c := range s {
			if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
				b.WriteRune(c)
				lastDash = false
			} else if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	for _, r := range strings.ToLower(title) {
		if t, ok := translit[r]; ok {
			if t == "" {
				continue
			}
			emit(t)
			continue
		}
		emit(string(r))
	}
	s := strings.Trim(b.String(), "-")
	if len(s) > max { // result is pure ASCII, so bytes are characters
		s = strings.TrimRight(s[:max], "-")
	}
	if s == "" {
		return "task"
	}
	return s
}
