package app

import (
	"context"
	"fmt"
	"html"

	"github.com/thegriglat/platui/internal/forge"
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
)

// ReviewResult — итог «готово к ревью».
type ReviewResult struct {
	MRURL, State string
	Warnings     []string
}

// Review отправляет ветку, создаёт MR (GitLab), комментирует задачу и переводит её на ревью.
func (a *App) Review(ctx context.Context, p plane.Project, it plane.WorkItem) (ReviewResult, error) {
	var res ReviewResult
	g, err := a.git()
	if err != nil {
		return res, err
	}
	branch, err := g.Current()
	if err != nil {
		return res, err
	}
	if err := g.Push(a.remote(), branch); err != nil {
		return res, err
	}
	a.createMR(ctx, g, branch, p, it, &res)
	m, err := a.Meta(ctx, p.ID)
	if err != nil {
		return res, err
	}
	name := a.cfg().Workflow.ReviewState
	if name == "" {
		name = "In Review"
	}
	st, ok := findState(m.States, name)
	if !ok {
		res.Warnings = append(res.Warnings, fmt.Sprintf("review state %q not found", name))
		return res, nil
	}
	if st.ID == it.StateID {
		return res, nil
	}
	if _, err := a.Update(ctx, p.ID, it.ID, plane.WorkItemPatch{StateID: &st.ID}); err != nil {
		return res, err
	}
	res.State = st.Name
	return res, nil
}

func (a *App) createMR(ctx context.Context, g gitbranch.Git, branch string, p plane.Project, it plane.WorkItem, res *ReviewResult) {
	warn := func(format string, args ...any) { res.Warnings = append(res.Warnings, fmt.Sprintf(format, args...)) }
	dir, _ := g.Toplevel()
	title, err := gitbranch.CommitMessage(a.commitTemplate(), gitItem(p, it))
	if err != nil {
		warn("MR title: %v", err)
		return
	}
	url, err := a.d.Forge.CreateMR(ctx, forge.MRRequest{
		Dir: dir, Head: branch, Target: a.cfg().GitLab.TargetBranch, Title: title,
		Body: a.d.API.WebURL(p, it), Draft: a.cfg().GitLab.Draft,
	})
	if err != nil {
		warn("MR: %v", err)
		return
	}
	if url == "" {
		return
	}
	res.MRURL = url
	h := html.EscapeString(url)
	if _, err := a.d.API.CreateComment(ctx, p.ID, it.ID, `<p>MR: <a href="`+h+`">`+h+`</a></p>`); err != nil {
		warn("MR comment: %v", err)
	}
}
