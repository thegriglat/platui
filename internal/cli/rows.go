package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/plane"
)

// row is an issue in ls/search/show output.
type row struct {
	Key        string   `json:"key"`
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	State      string   `json:"state"`
	StateGroup string   `json:"state_group"`
	Priority   string   `json:"priority"`
	Assignees  []string `json:"assignees"`
	Labels     []string `json:"labels"`
	URL        string   `json:"url"`
	UpdatedAt  string   `json:"updated_at"`
}

func itemKey(p plane.Project, it plane.WorkItem) string {
	return p.Identifier + "-" + strconv.Itoa(it.Sequence)
}

func buildRow(a *app.App, m app.Meta, it plane.WorkItem) row {
	r := row{
		Key: itemKey(m.Project, it), ID: it.ID, Name: it.Name, Priority: it.Priority,
		Assignees: []string{}, Labels: []string{}, URL: a.CopyItems(m.Project, it).URL,
	}
	if !it.UpdatedAt.IsZero() {
		r.UpdatedAt = it.UpdatedAt.UTC().Format(time.RFC3339)
	}
	for _, s := range m.States {
		if s.ID == it.StateID {
			r.State, r.StateGroup = s.Name, s.Group
		}
	}
	for _, id := range it.AssigneeIDs {
		r.Assignees = append(r.Assignees, nameOr(id, func() (string, bool) {
			for _, u := range m.Members {
				if u.ID == id {
					return u.Name(), true
				}
			}
			return "", false
		}))
	}
	for _, id := range it.LabelIDs {
		r.Labels = append(r.Labels, nameOr(id, func() (string, bool) {
			for _, l := range m.Labels {
				if l.ID == id {
					return l.Name, true
				}
			}
			return "", false
		}))
	}
	return r
}

func nameOr(id string, find func() (string, bool)) string {
	if n, ok := find(); ok {
		return n
	}
	return id
}

func writeJSON(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", b)
	return err
}

func prio(p string) string {
	if p == "" {
		return "none"
	}
	return p
}

// writeRows prints rows with aligned columns.
func writeRows(w io.Writer, rows []row) {
	var kw, sw, pw int
	for _, r := range rows {
		kw = max(kw, utf8.RuneCountInString(r.Key))
		sw = max(sw, utf8.RuneCountInString(r.State))
		pw = max(pw, utf8.RuneCountInString(prio(r.Priority)))
	}
	pad := func(s string, n int) string {
		return s + strings.Repeat(" ", max(n-utf8.RuneCountInString(s), 0))
	}
	for _, r := range rows {
		fmt.Fprintf(w, "%s  %s  %s  %s\n", pad(r.Key, kw), pad(r.State, sw), pad(prio(r.Priority), pw), r.Name)
	}
}

func sortBySeq(items []plane.WorkItem) {
	slices.SortStableFunc(items, func(a, b plane.WorkItem) int { return a.Sequence - b.Sequence })
}
