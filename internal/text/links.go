package text

import (
	"strings"

	"mvdan.cc/xurls/v2"
)

var urlRe = xurls.Strict()

// ExtractLinks возвращает http/https-ссылки из HTML: href у <a> и голые URL
// в тексте; без дублей, в порядке появления.
func ExtractLinks(html string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(u string) {
		u = strings.TrimSpace(u)
		low := strings.ToLower(u)
		if (strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://")) && !seen[u] {
			seen[u] = true
			out = append(out, u)
		}
	}
	var walk func(n *node)
	walk = func(n *node) {
		if n.tag == "" {
			for _, u := range urlRe.FindAllString(n.text, -1) {
				add(u)
			}
			return
		}
		if n.tag == "a" {
			add(n.attrs["href"])
		}
		for _, c := range n.children {
			walk(c)
		}
	}
	walk(parseHTML(html))
	return out
}
