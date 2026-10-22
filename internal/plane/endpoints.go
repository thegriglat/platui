package plane

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type memberJSON struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
}

func (m memberJSON) toMember() Member {
	return Member{ID: m.ID, DisplayName: m.DisplayName, Email: m.Email, FirstName: m.FirstName, LastName: m.LastName}
}

func convMember(r json.RawMessage) (Member, error) {
	var m memberJSON
	if err := json.Unmarshal(r, &m); err != nil {
		return Member{}, err
	}
	// project-members may nest the user in the member field
	if m.ID == "" {
		var w struct {
			Member json.RawMessage `json:"member"`
		}
		if json.Unmarshal(r, &w) == nil && len(w.Member) > 0 {
			var in memberJSON
			if json.Unmarshal(w.Member, &in) == nil {
				return in.toMember(), nil
			}
			return Member{ID: idOf(w.Member)}, nil
		}
	}
	if m.ID == "" {
		return Member{}, &APIError{Status: 200, Message: "unexpected response"}
	}
	return m.toMember(), nil
}

// Me is an API method.
func (c *Client) Me(ctx context.Context) (Member, error) {
	data, err := c.do(ctx, http.MethodGet, c.root("users", "me"), nil, nil)
	if err != nil {
		return Member{}, err
	}
	m, err := convMember(data)
	if err != nil {
		return Member{}, fmt.Errorf("plane: bad response: %w", err)
	}
	return m, nil
}

// Projects is an API method.
func (c *Client) Projects(ctx context.Context) ([]Project, error) {
	return decodeList(c, ctx, c.path("projects"), func(r json.RawMessage) (Project, error) {
		var p struct {
			ID         string `json:"id"`
			Identifier string `json:"identifier"`
			Name       string `json:"name"`
		}
		err := json.Unmarshal(r, &p)
		return Project{ID: p.ID, Identifier: p.Identifier, Name: p.Name}, err
	})
}

// States is an API method.
func (c *Client) States(ctx context.Context, projectID string) ([]State, error) {
	return decodeList(c, ctx, c.path("projects", projectID, "states"), func(r json.RawMessage) (State, error) {
		return convState(r)
	})
}

// Members is an API method.
func (c *Client) Members(ctx context.Context, projectID string) ([]Member, error) {
	return decodeList(c, ctx, c.path("projects", projectID, "project-members"), convMember)
}

// Labels is an API method.
func (c *Client) Labels(ctx context.Context, projectID string) ([]Label, error) {
	return decodeList(c, ctx, c.path("projects", projectID, "labels"), func(r json.RawMessage) (Label, error) {
		var l struct{ ID, Name, Color string }
		err := json.Unmarshal(r, &l)
		return Label{ID: l.ID, Name: l.Name, Color: l.Color}, err
	})
}

// WorkItems is an API method.
func (c *Client) WorkItems(ctx context.Context, projectID string) ([]WorkItem, error) {
	return decodeList(c, ctx, c.path("projects", projectID, "work-items"), func(r json.RawMessage) (WorkItem, error) {
		return c.parseWorkItem(r, projectID)
	})
}

// WorkItemsUpdated returns the project tasks updated at or after since (newest first, at
// least those; pages are read until an older task) and total_count of the project tasks.
func (c *Client) WorkItemsUpdated(ctx context.Context, projectID string, since time.Time) ([]WorkItem, int, error) {
	older := func(page []json.RawMessage) bool {
		if len(page) == 0 {
			return true
		}
		var w struct {
			UpdatedAt string `json:"updated_at"`
		}
		return json.Unmarshal(page[len(page)-1], &w) == nil && parseTime(w.UpdatedAt).Before(since)
	}
	raw, total, err := c.getPages(ctx, c.path("projects", projectID, "work-items"), url.Values{"order_by": {"-updated_at"}}, older)
	if err != nil {
		return nil, 0, err
	}
	out := make([]WorkItem, 0, len(raw))
	for _, r := range raw {
		it, err := c.parseWorkItem(r, projectID)
		if err != nil {
			return nil, 0, fmt.Errorf("plane: bad item: %w", err)
		}
		if !it.UpdatedAt.Before(since) {
			out = append(out, it)
		}
	}
	return out, total, nil
}

func (c *Client) itemCall(ctx context.Context, method, projectID, id string, body any) (WorkItem, error) {
	u := c.path("projects", projectID, "work-items")
	if id != "" {
		u = c.path("projects", projectID, "work-items", id)
	}
	data, err := c.do(ctx, method, u, nil, body)
	if err != nil {
		return WorkItem{}, err
	}
	it, err := c.parseWorkItem(data, projectID)
	if err != nil {
		return WorkItem{}, fmt.Errorf("plane: bad work item: %w", err)
	}
	return it, nil
}

// WorkItem is an API method.
func (c *Client) WorkItem(ctx context.Context, projectID, id string) (WorkItem, error) {
	return c.itemCall(ctx, http.MethodGet, projectID, id, nil)
}

// CreateWorkItem is an API method; empty fields are not sent.
func (c *Client) CreateWorkItem(ctx context.Context, projectID string, n NewWorkItem) (WorkItem, error) {
	body := map[string]any{"name": n.Name, "description_html": n.DescHTML}
	if n.StateID != "" {
		body["state"] = n.StateID
	}
	if n.Priority != "" {
		body["priority"] = n.Priority
	}
	if len(n.AssigneeIDs) > 0 {
		body["assignees"] = n.AssigneeIDs
	}
	if len(n.LabelIDs) > 0 {
		body["labels"] = n.LabelIDs
	}
	if n.ParentID != "" {
		body["parent"] = n.ParentID
	}
	return c.itemCall(ctx, http.MethodPost, projectID, "", body)
}

// UpdateWorkItem is an API method; only non-nil fields are sent.
func (c *Client) UpdateWorkItem(ctx context.Context, projectID, id string, p WorkItemPatch) (WorkItem, error) {
	body := map[string]any{}
	for k, v := range map[string]*string{"state": p.StateID, "priority": p.Priority, "name": p.Name, "description_html": p.DescHTML} {
		if v != nil {
			body[k] = *v
		}
	}
	for k, v := range map[string]*[]string{"assignees": p.AssigneeIDs, "labels": p.LabelIDs} {
		if v != nil {
			ids := *v
			if ids == nil {
				ids = []string{}
			}
			body[k] = ids
		}
	}
	return c.itemCall(ctx, http.MethodPatch, projectID, id, body)
}

// Comments is an API method.
func (c *Client) Comments(ctx context.Context, projectID, id string) ([]Comment, error) {
	return decodeList(c, ctx, c.commentsURL(projectID, id), convComment)
}

// CreateComment is an API method.
func (c *Client) CreateComment(ctx context.Context, projectID, id, html string) (Comment, error) {
	data, err := c.do(ctx, http.MethodPost, c.commentsURL(projectID, id), nil, map[string]any{"comment_html": html})
	if err != nil {
		return Comment{}, err
	}
	cm, err := convComment(data)
	if err != nil {
		return Comment{}, fmt.Errorf("plane: bad comment: %w", err)
	}
	return cm, nil
}

func (c *Client) commentsURL(projectID, id string) string {
	return c.path("projects", projectID, "work-items", id, "comments")
}

// convComment parses a comment: the author is actor, otherwise created_by.
func convComment(r json.RawMessage) (Comment, error) {
	var cm struct {
		ID          string          `json:"id"`
		Actor       json.RawMessage `json:"actor"`
		CreatedBy   json.RawMessage `json:"created_by"`
		CommentHTML string          `json:"comment_html"`
		CreatedAt   string          `json:"created_at"`
	}
	if err := json.Unmarshal(r, &cm); err != nil {
		return Comment{}, err
	}
	author := idOf(cm.Actor)
	if author == "" {
		author = idOf(cm.CreatedBy)
	}
	return Comment{ID: cm.ID, Author: author, HTML: cm.CommentHTML, CreatedAt: parseTime(cm.CreatedAt)}, nil
}
