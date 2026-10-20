package text

import "strings"

// renderTable переводит таблицу GFM; первая строка — заголовок.
func renderTable(n *node) string {
	var rows [][]*node
	collectRows(n, &rows)
	if len(rows) == 0 {
		return ""
	}
	cols := 0
	for _, r := range rows {
		cols = max(cols, len(r))
	}
	line := func(r []*node) string {
		cells := make([]string, cols)
		for i := range cells {
			if i < len(r) {
				cells[i] = strings.ReplaceAll(oneLine(inlineAll(r[i].children)), "|", `\|`)
			}
		}
		return "| " + strings.Join(cells, " | ") + " |"
	}
	delim := make([]string, cols)
	for i := range delim {
		delim[i] = "---"
		if i < len(rows[0]) {
			delim[i] = alignDelim(rows[0][i])
		}
	}
	out := []string{line(rows[0]), "| " + strings.Join(delim, " | ") + " |"}
	for _, r := range rows[1:] {
		out = append(out, line(r))
	}
	return strings.Join(out, "\n")
}

func collectRows(n *node, rows *[][]*node) {
	for _, c := range n.children {
		switch c.tag {
		case "tr":
			var cells []*node
			for _, cc := range c.children {
				if cc.tag == "td" || cc.tag == "th" {
					cells = append(cells, cc)
				}
			}
			*rows = append(*rows, cells)
		case "thead", "tbody", "tfoot":
			collectRows(c, rows)
		}
	}
}

func alignDelim(cell *node) string {
	a := cell.attrs["align"]
	if st := cell.attrs["style"]; a == "" && strings.Contains(st, "text-align") {
		for _, v := range []string{"left", "right", "center"} {
			if strings.Contains(st, "text-align:"+v) || strings.Contains(st, "text-align: "+v) {
				a = v
			}
		}
	}
	switch a {
	case "left":
		return ":---"
	case "right":
		return "---:"
	case "center":
		return ":---:"
	}
	return "---"
}
