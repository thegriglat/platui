package text

import (
	"regexp"
	"strings"
)

var (
	wsRe      = regexp.MustCompile(`[ \t\r\n\f]+`)
	nlSpaceRe = regexp.MustCompile(` *\n *`)
	brSpceRe  = regexp.MustCompile(`[ \n]*\x00[ \n]*`)
	mdEscape  = strings.NewReplacer(`\`, `\\`, "`", "\\`", "*", `\*`, "_", `\_`, "[", `\[`,
		"]", `\]`, "<", `\<`, ">", `\>`, "~", `\~`, "&", `\&`)
)

func inlineAll(nodes []*node) string {
	var b strings.Builder
	for _, c := range nodes {
		b.WriteString(inline(c))
	}
	return b.String()
}

// inline переводит узел в строчный Markdown (с brMark вместо переносов).
func inline(n *node) string {
	switch n.tag {
	case "":
		return mdEscape.Replace(wsRe.ReplaceAllStringFunc(n.text, func(w string) string {
			if strings.Contains(w, "\n") {
				return "\n" // мягкий перенос строки сохраняем
			}
			return " "
		}))
	case "br":
		return brMark
	case "strong", "b":
		return wrap("**", inlineAll(n.children))
	case "em", "i":
		return wrap("*", inlineAll(n.children))
	case "s", "del", "strike":
		return wrap("~~", inlineAll(n.children))
	case "code":
		return codeSpan(textContent(n))
	case "a":
		return link(n)
	case "img":
		return "![" + n.attrs["alt"] + "](" + linkDest(n.attrs["src"]) + ")"
	case "input":
		if n.attrs["type"] != "checkbox" {
			return ""
		}
		if _, ok := n.attrs["checked"]; ok {
			return "[x] "
		}
		return "[ ] "
	}
	return inlineAll(n.children)
}

// wrap оборачивает текст маркером, вынося пробелы по краям наружу.
func wrap(mark, inner string) string {
	core := strings.Trim(inner, " ")
	if core == "" {
		return inner
	}
	lead := inner[:len(inner)-len(strings.TrimLeft(inner, " "))]
	trail := inner[len(strings.TrimRight(inner, " ")):]
	return lead + mark + core + mark + trail
}

func codeSpan(s string) string {
	s = wsRe.ReplaceAllString(s, " ")
	if s == "" {
		return ""
	}
	fence := strings.Repeat("`", longestRun(s, '`')+1)
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		s = " " + s + " "
	}
	return fence + s + fence
}

func link(n *node) string {
	inner := inlineAll(n.children)
	href, ok := n.attrs["href"]
	if !ok || href == "" {
		return inner
	}
	if textContent(n) == href && !strings.ContainsAny(href, " <>") {
		return "<" + href + ">"
	}
	title := ""
	if t := n.attrs["title"]; t != "" {
		title = ` "` + strings.ReplaceAll(t, `"`, `\"`) + `"`
	}
	return "[" + inner + "](" + linkDest(href) + title + ")"
}

func linkDest(d string) string {
	if d == "" || strings.ContainsAny(d, " ()<>") {
		return "<" + strings.NewReplacer("<", "%3C", ">", "%3E", " ", "%20").Replace(d) + ">"
	}
	return d
}

// tidyInline превращает накопленный строчный текст в абзац: края обрезаны,
// метки переносов — в "\"+"\n".
func tidyInline(s string) string {
	s = strings.Trim(s, " \n"+brMark)
	s = brSpceRe.ReplaceAllString(s, "\\\n")
	return strings.TrimSpace(nlSpaceRe.ReplaceAllString(s, "\n"))
}

// oneLine — строчный текст в одну строку (заголовки, ячейки).
func oneLine(s string) string {
	return strings.TrimSpace(wsRe.ReplaceAllString(strings.ReplaceAll(s, brMark, " "), " "))
}
