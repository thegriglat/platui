package plane

import (
	"html"
	"regexp"
	"strings"
)

var (
	reBlockEnd   = regexp.MustCompile(`(?i)<\s*br\s*/?\s*>|</\s*(p|div|h[1-6]|li|ul|ol|blockquote|pre|tr)\s*>`)
	reLiStart    = regexp.MustCompile(`(?i)<\s*li(\s[^>]*)?>`)
	reTag        = regexp.MustCompile(`(?s)<[^>]*>`)
	reBlankLines = regexp.MustCompile(`\n{3,}`)
)

// HTMLToText: абзацы/<br>/<li> → переводы строк, сущности раскодированы, теги убраны.
func HTMLToText(s string) string {
	s = reLiStart.ReplaceAllString(s, "- ")
	s = reBlockEnd.ReplaceAllString(s, "\n")
	s = reTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	s = strings.Join(lines, "\n")
	s = reBlankLines.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
