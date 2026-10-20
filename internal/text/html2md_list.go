package text

import (
	"strconv"
	"strings"
)

// renderList converts ul/ol (with nesting) to Markdown.
func renderList(n *node) string {
	start := 1
	if v, err := strconv.Atoi(n.attrs["start"]); err == nil && n.tag == "ol" {
		start = v
	}
	var items []*node
	loose := false
	for _, c := range n.children {
		if c.tag != "li" {
			continue
		}
		items = append(items, c)
		for _, cc := range c.children {
			if cc.tag == "p" {
				loose = true
			}
		}
	}
	sep, joiner := "\n", "\n"
	if loose {
		sep, joiner = "\n\n", "\n\n"
	}
	out := make([]string, 0, len(items))
	for i, li := range items {
		marker := "- "
		if n.tag == "ol" {
			marker = strconv.Itoa(start+i) + ". "
		}
		out = append(out, renderItem(li, marker, joiner))
	}
	return strings.Join(out, sep)
}

func renderItem(li *node, marker, joiner string) string {
	body := strings.Join(flow(li.children), joiner)
	if strings.TrimSpace(body) == "" {
		return strings.TrimRight(marker, " ")
	}
	lines := strings.Split(body, "\n")
	pad := strings.Repeat(" ", len(marker))
	for i, l := range lines {
		switch {
		case i == 0:
			lines[i] = marker + l
		case l != "":
			lines[i] = pad + l
		}
	}
	return strings.Join(lines, "\n")
}
