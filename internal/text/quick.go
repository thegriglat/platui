// Package text holds pure functions for working with issue text: quick
// create, Markdown↔HTML, editing in an external editor, and OSC 52.
package text

import (
	"errors"
	"fmt"
	"strings"
)

// ErrEmptyTitle means the quick-create line has no title.
var ErrEmptyTitle = errors.New("empty title")

// Quick is the result of parsing a quick-create line.
type Quick struct {
	Title     string   // remaining words joined by single spaces
	Priority  string   // "" or urgent|high|medium|low|none
	Assignees []string // without "@", in order of appearance, no duplicates
	Labels    []string // without "#", in order of appearance, no duplicates
}

var priorities = map[string]string{
	"urgent": "urgent", "u": "urgent",
	"high": "high", "h": "high",
	"medium": "medium", "m": "medium",
	"low": "low", "l": "low",
	"none": "none", "n": "none",
}

// ParseQuick parses a line like "fix login !high @me #bug".
func ParseQuick(s string) (Quick, error) {
	var q Quick
	var words []string
	for _, tok := range strings.Fields(s) {
		switch {
		case tok[0] == '\\' && len(tok) > 1 && strings.ContainsRune("!@#", rune(tok[1])):
			words = append(words, tok[1:])
		case tok[0] == '!' && len(tok) > 1:
			p, ok := priorities[strings.ToLower(tok[1:])]
			if !ok {
				return Quick{}, fmt.Errorf("unknown priority %q", tok)
			}
			if q.Priority != "" {
				return Quick{}, fmt.Errorf("second priority %q", tok)
			}
			q.Priority = p
		case tok[0] == '@' && len(tok) > 1:
			q.Assignees = addUnique(q.Assignees, tok[1:])
		case tok[0] == '#' && len(tok) > 1:
			q.Labels = addUnique(q.Labels, tok[1:])
		default:
			words = append(words, tok)
		}
	}
	q.Title = strings.Join(words, " ")
	if q.Title == "" {
		return Quick{}, ErrEmptyTitle
	}
	return q, nil
}

func addUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}
