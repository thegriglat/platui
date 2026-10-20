package text

import (
	"html"
	"strings"
)

// node — узел упрощённого дерева HTML: текст (tag == "") или элемент.
type node struct {
	tag      string
	text     string // для текстовых узлов — уже с раскодированными сущностями
	attrs    map[string]string
	children []*node
}

var voidTags = map[string]bool{"br": true, "hr": true, "img": true, "input": true, "meta": true, "link": true, "wbr": true}

// Теги, открытие которых неявно закрывает такой же открытый тег на вершине стека.
var autoClose = map[string]bool{"li": true, "p": true, "tr": true, "td": true, "th": true}

// parseHTML разбирает HTML токенайзером в дерево; ошибок не возвращает:
// неизвестное и «битое» разбирается best effort.
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

// closeTag снимает со стека всё до ближайшего открытого name; нет такого — игнор.
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

// parseOpenTag разбирает "<name attr=v ...>" с начала s; size — длина тега.
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

// textContent — весь текст поддерева без изменений.
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
