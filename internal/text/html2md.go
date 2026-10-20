package text

import (
	"regexp"
	"strings"
)

// brMark is an internal marker for a <br> line break until the paragraph is finally assembled.
const brMark = "\x00"

var blockTags = map[string]bool{
	"p": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"ul": true, "ol": true, "blockquote": true, "pre": true, "hr": true, "table": true,
}

// HTMLToMarkdown converts a description's HTML to Markdown for editing in $EDITOR.
// Supported: p, br, h1–h6, strong/b, em/i, s/del, code, pre, a, ul/ol/li,
// blockquote, hr, as well as img and tables; other tags yield text only.
func HTMLToMarkdown(h string) string {
	root := parseHTML(h)
	return strings.TrimSpace(strings.Join(flow(root.children), "\n\n"))
}

// flow builds Markdown blocks from a sequence of nodes.
func flow(nodes []*node) []string {
	var blocks []string
	var cur strings.Builder
	flush := func() {
		if s := tidyInline(cur.String()); s != "" {
			blocks = append(blocks, escapeLineStarts(s))
		}
		cur.Reset()
	}
	for _, c := range nodes {
		switch {
		case blockTags[c.tag]:
			flush()
			if s := renderBlock(c); s != "" {
				blocks = append(blocks, s)
			}
		case c.tag != "" && hasBlock(c):
			flush()
			blocks = append(blocks, flow(c.children)...)
		default:
			cur.WriteString(inline(c))
		}
	}
	flush()
	return blocks
}

func hasBlock(n *node) bool {
	for _, c := range n.children {
		if blockTags[c.tag] || hasBlock(c) {
			return true
		}
	}
	return false
}

func renderBlock(n *node) string {
	switch n.tag {
	case "p":
		return strings.Join(flow(n.children), "\n\n")
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return strings.Repeat("#", int(n.tag[1]-'0')) + " " + oneLine(inlineAll(n.children))
	case "hr":
		return "---"
	case "pre":
		return renderPre(n)
	case "blockquote":
		return quote(strings.Join(flow(n.children), "\n\n"))
	case "ul", "ol":
		return renderList(n)
	case "table":
		return renderTable(n)
	}
	return ""
}

func quote(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight("> "+l, " ")
	}
	return strings.Join(lines, "\n")
}

func renderPre(n *node) string {
	lang := ""
	for _, c := range n.children {
		if c.tag == "code" {
			for _, cl := range strings.Fields(c.attrs["class"]) {
				if strings.HasPrefix(cl, "language-") {
					lang = strings.TrimPrefix(cl, "language-")
				}
			}
		}
	}
	body := strings.TrimSuffix(textContent(n), "\n")
	fence := strings.Repeat("`", max(3, longestRun(body, '`')+1))
	return fence + lang + "\n" + body + "\n" + fence
}

func longestRun(s string, ch byte) int {
	best, run := 0, 0
	for i := 0; i < len(s); i++ {
		if s[i] == ch {
			run++
			best = max(best, run)
		} else {
			run = 0
		}
	}
	return best
}

var (
	lineStartRe = regexp.MustCompile(`^(#{1,6}(\s|$)|[-+>]|=+\s*$)`)
	numListRe   = regexp.MustCompile(`^(\d+)([.)])(\s|$)`)
)

// escapeLineStarts escapes line starts that would otherwise become block markup.
func escapeLineStarts(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if lineStartRe.MatchString(l) {
			lines[i] = `\` + l
		} else if numListRe.MatchString(l) {
			lines[i] = numListRe.ReplaceAllString(l, `$1\$2$3`)
		}
	}
	return strings.Join(lines, "\n")
}
