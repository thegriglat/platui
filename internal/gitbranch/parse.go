package gitbranch

import (
	"regexp"
	"strconv"
	"strings"
)

var fallbackRe = regexp.MustCompile(`(?:^|[/_-])([A-Z][A-Z0-9]*)-([0-9]+)(?:$|[/_-])`)

// ParseBranch extracts the task identifier and sequence from a branch name:
// first by the branch template, then by a generic ID-N pattern. The
// identifier is upper-cased; Title is empty.
func ParseBranch(template, name string) (Item, bool) {
	if it, ok := parseByTemplate(template, name); ok {
		return it, true
	}
	m := fallbackRe.FindStringSubmatch(name)
	if m == nil {
		return Item{}, false
	}
	n, err := strconv.Atoi(m[2])
	if err != nil {
		return Item{}, false
	}
	return Item{Identifier: m[1], Sequence: n}, true
}

func parseByTemplate(template, name string) (Item, bool) {
	if !strings.Contains(template, "{identifier}") || !strings.Contains(template, "{sequence}") {
		return Item{}, false
	}
	var kinds []string
	var sb strings.Builder
	sb.WriteString("^")
	rest := template
	for {
		loc := placeholderRe.FindStringIndex(rest)
		if loc == nil {
			sb.WriteString(regexp.QuoteMeta(rest))
			break
		}
		sb.WriteString(regexp.QuoteMeta(rest[:loc[0]]))
		ph := rest[loc[0]:loc[1]]
		switch ph {
		case "{identifier}":
			sb.WriteString(`([A-Za-z][A-Za-z0-9]*)`)
		case "{sequence}":
			sb.WriteString(`([0-9]+)`)
		case "{slug}":
			sb.WriteString(`([a-z0-9-]*)`)
		default:
			return Item{}, false
		}
		kinds = append(kinds, ph)
		rest = rest[loc[1]:]
	}
	sb.WriteString("$")
	re, err := regexp.Compile(sb.String())
	if err != nil {
		return Item{}, false
	}
	m := re.FindStringSubmatch(name)
	if m == nil {
		return Item{}, false
	}
	var it Item
	var haveID, haveSeq bool
	for i, k := range kinds {
		switch {
		case k == "{identifier}" && !haveID:
			it.Identifier, haveID = strings.ToUpper(m[i+1]), true
		case k == "{sequence}" && !haveSeq:
			n, err := strconv.Atoi(m[i+1])
			if err != nil {
				return Item{}, false
			}
			it.Sequence, haveSeq = n, true
		}
	}
	return it, true
}
