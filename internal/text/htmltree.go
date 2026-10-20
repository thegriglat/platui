package text

import (
	"html"
	"strings"
)

// node is a node of the simplified HTML tree: text (tag == "") or an element.
type node struct {
	tag      string
	text     string // for text nodes — entities already decoded
	attrs    map[string]string
	children []*node
}

var voidTags = map[string]bool{"br": true, "hr": true, "img": true, "input": true, "meta": true, "link": true, "wbr": true}

// Tags whose opening implicitly closes the same tag open on top of the stack.
var autoClose = map[string]bool{"li": true, "p": true, "tr": true, "td": true, "th": true}

// parseHTML parses HTML into a tree with a tokenizer; it returns no errors:
// unknown and malformed input is parsed best effort.
func parseHTML(s string) *node {
	root := &node{tag: "#root"}
	stack := []*node{root}
	top := func() *node { return stack[len(stack)-1] }
	add := func(n *node) { top().children = append(top().children, n) }
	text := func(t string) {
		if t != "" {
			add(&node{text: html.UnescapeString(t)})
		}
	}
	for i := 0; i < len(s); {
		j := strings.IndexByte(s[i:], '<')
		if j < 0 {
			text(s[i:])
			break
		}
		text(s[i : i+j])
		i += j
		rest := s[i:]
		switch {
		case strings.HasPrefix(rest, "<!--"):
			end := strings.Index(rest, "-->")
			if end < 0 {
				return root
			}
			i += end + 3
		case len(rest) > 1 && (rest[1] == '!' || rest[1] == '?'):
			end := strings.IndexByte(rest, '>')
			if end < 0 {
				return root
			}
			i += end + 1
		case len(rest) > 2 && rest[1] == '/' && isAlpha(rest[2]):
			end := strings.IndexByte(rest, '>')
			if end < 0 {
				return root
			}
			name := strings.ToLower(strings.TrimSpace(rest[2:end]))
			stack = closeTag(stack, name)
			i += end + 1
		case len(rest) > 1 && isAlpha(rest[1]):
			n, selfClose, size := parseOpenTag(rest)
			if autoClose[n.tag] && top().tag == n.tag {
				stack = stack[:len(stack)-1]
			}
			add(n)
			if !voidTags[n.tag] && !selfClose {
				stack = append(stack, n)
			}
			i += size
		default:
			text("<")
			i++
		}
	}
	return root
}

// closeTag pops the stack down to the nearest open name; if there is none, it is ignored.
func closeTag(stack []*node, name string) []*node {
	for k := len(stack) - 1; k > 0; k-- {
		if stack[k].tag == name {
			return stack[:k]
		}
	}
	return stack
}

func isAlpha(c byte) bool { return c|0x20 >= 'a' && c|0x20 <= 'z' }

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' }

// parseOpenTag parses "<name attr=v ...>" at the start of s; size is the tag length.
func parseOpenTag(s string) (n *node, selfClose bool, size int) {
	i := 1
	for i < len(s) && !isSpace(s[i]) && s[i] != '>' && s[i] != '/' {
		i++
	}
	n = &node{tag: strings.ToLower(s[1:i]), attrs: map[string]string{}}
	for i < len(s) {
		for i < len(s) && isSpace(s[i]) {
			i++
		}
		if i >= len(s) {
			break
		}
		if s[i] == '>' {
			return n, selfClose, i + 1
		}
		if s[i] == '/' {
			selfClose = true
			i++
			continue
		}
		selfClose = false
		k := i
		for i < len(s) && !isSpace(s[i]) && s[i] != '=' && s[i] != '>' && s[i] != '/' {
			i++
		}
		key := strings.ToLower(s[k:i])
		for i < len(s) && isSpace(s[i]) {
			i++
		}
		val := ""
		if i < len(s) && s[i] == '=' {
			i++
			for i < len(s) && isSpace(s[i]) {
				i++
			}
			val, i = parseAttrValue(s, i)
		}
		n.attrs[key] = html.UnescapeString(val)
	}
	return n, selfClose, len(s)
}

func parseAttrValue(s string, i int) (string, int) {
	if i < len(s) && (s[i] == '"' || s[i] == '\'') {
		end := strings.IndexByte(s[i+1:], s[i])
		if end < 0 {
			return s[i+1:], len(s)
		}
		return s[i+1 : i+1+end], i + end + 2
	}
	k := i
	for i < len(s) && !isSpace(s[i]) && s[i] != '>' {
		i++
	}
	return s[k:i], i
}

// textContent is the whole text of the subtree, unchanged.
func textContent(n *node) string {
	if n.tag == "" {
		return n.text
	}
	var b strings.Builder
	for _, c := range n.children {
		b.WriteString(textContent(c))
	}
	return b.String()
}
