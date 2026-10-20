package app

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/text"
)

// CreateQuick создаёт задачу из строки "название !prio @user #label".
func (a *App) CreateQuick(ctx context.Context, projectID, line, descMarkdown string) (plane.WorkItem, error) {
	q, err := text.ParseQuick(line)
	if err != nil {
		return plane.WorkItem{}, err
	}
	m, err := a.Meta(ctx, projectID)
	if err != nil {
		return plane.WorkItem{}, err
	}
	n := plane.NewWorkItem{Name: q.Title, Priority: q.Priority}
	for _, who := range q.Assignees {
		var id string
		if strings.EqualFold(who, "me") {
			me, err := a.Me(ctx)
			if err != nil {
				return plane.WorkItem{}, err
			}
			id = me.ID
		} else if id, err = findMember(m.Members, who); err != nil {
			return plane.WorkItem{}, err
		}
		if !slices.Contains(n.AssigneeIDs, id) {
			n.AssigneeIDs = append(n.AssigneeIDs, id)
		}
	}
	for _, name := range q.Labels {
		i := slices.IndexFunc(m.Labels, func(l plane.Label) bool { return strings.EqualFold(l.Name, name) })
		if i < 0 {
			return plane.WorkItem{}, fmt.Errorf("%w: %s", ErrUnknownLabel, name)
		}
		if !slices.Contains(n.LabelIDs, m.Labels[i].ID) {
			n.LabelIDs = append(n.LabelIDs, m.Labels[i].ID)
		}
	}
	if strings.TrimSpace(descMarkdown) != "" {
		n.DescHTML = text.MarkdownToHTML(descMarkdown)
	}
	return a.d.API.CreateWorkItem(ctx, projectID, n)
}

// findMember: по DisplayName, затем по части Email до "@"; неоднозначно или нет — ErrUnknownMember.
func findMember(ms []plane.Member, name string) (string, error) {
	pick := func(match func(plane.Member) bool) (string, int) {
		var id string
		n := 0
		for _, m := range ms {
			if match(m) {
				id, n = m.ID, n+1
			}
		}
		return id, n
	}
	if id, n := pick(func(m plane.Member) bool { return strings.EqualFold(m.DisplayName, name) }); n == 1 {
		return id, nil
	} else if n > 1 {
		return "", fmt.Errorf("%w: %s (ambiguous)", ErrUnknownMember, name)
	}
	id, n := pick(func(m plane.Member) bool {
		local, _, _ := strings.Cut(m.Email, "@")
		return local != "" && strings.EqualFold(local, name)
	})
	if n != 1 {
		return "", fmt.Errorf("%w: %s", ErrUnknownMember, name)
	}
	return id, nil
}

// Comment добавляет комментарий из Markdown.
func (a *App) Comment(ctx context.Context, projectID, itemID, markdown string) (plane.Comment, error) {
	return a.d.API.CreateComment(ctx, projectID, itemID, text.MarkdownToHTML(markdown))
}

// SetDescription заменяет описание задачи (Markdown → HTML).
func (a *App) SetDescription(ctx context.Context, projectID, itemID, markdown string) (plane.WorkItem, error) {
	h := text.MarkdownToHTML(markdown)
	return a.Update(ctx, projectID, itemID, plane.WorkItemPatch{DescHTML: &h})
}

// bulk применяет fn к задачам последовательно; после отмены ctx остаток — ctx.Err().
func bulk[T any](ctx context.Context, xs []T, id func(T) string, fn func(T) error) map[string]error {
	res := make(map[string]error, len(xs))
	for _, x := range xs {
		if err := ctx.Err(); err != nil {
			res[id(x)] = err
			continue
		}
		res[id(x)] = fn(x)
	}
	return res
}

// BulkPatch применяет один патч к задачам.
func (a *App) BulkPatch(ctx context.Context, projectID string, ids []string, p plane.WorkItemPatch) map[string]error {
	return bulk(ctx, ids, func(s string) string { return s }, func(id string) error {
		_, err := a.Update(ctx, projectID, id, p)
		return err
	})
}

func (a *App) bulkList(ctx context.Context, projectID string, items []plane.WorkItem, get func(plane.WorkItem) []string,
	set func(*plane.WorkItemPatch, *[]string), val string, add bool) map[string]error {
	return bulk(ctx, items, func(it plane.WorkItem) string { return it.ID }, func(it plane.WorkItem) error {
		cur := get(it)
		has := slices.Contains(cur, val)
		if has == add {
			return nil // уже в нужном состоянии
		}
		var next []string
		if add {
			next = append(slices.Clone(cur), val)
		} else {
			next = slices.DeleteFunc(slices.Clone(cur), func(s string) bool { return s == val })
		}
		if next == nil {
			next = []string{}
		}
		var p plane.WorkItemPatch
		set(&p, &next)
		_, err := a.Update(ctx, projectID, it.ID, p)
		return err
	})
}

// BulkAssign добавляет или снимает исполнителя.
func (a *App) BulkAssign(ctx context.Context, projectID string, items []plane.WorkItem, memberID string, add bool) map[string]error {
	return a.bulkList(ctx, projectID, items, func(it plane.WorkItem) []string { return it.AssigneeIDs },
		func(p *plane.WorkItemPatch, l *[]string) { p.AssigneeIDs = l }, memberID, add)
}

// BulkLabel добавляет или снимает метку.
func (a *App) BulkLabel(ctx context.Context, projectID string, items []plane.WorkItem, labelID string, add bool) map[string]error {
	return a.bulkList(ctx, projectID, items, func(it plane.WorkItem) []string { return it.LabelIDs },
		func(p *plane.WorkItemPatch, l *[]string) { p.LabelIDs = l }, labelID, add)
}
