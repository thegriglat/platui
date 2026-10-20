package plane

import (
	"encoding/json"
	"time"
)

// ---- parsing ----

// idOf takes a UUID string or the object's id field.
func idOf(r json.RawMessage) string {
	var s string
	if json.Unmarshal(r, &s) == nil {
		return s
	}
	var o struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(r, &o) == nil {
		return o.ID
	}
	return ""
}

func idList(r json.RawMessage) []string {
	var arr []json.RawMessage
	if len(r) == 0 || json.Unmarshal(r, &arr) != nil {
		return nil
	}
	var out []string
	for _, e := range arr {
		if id := idOf(e); id != "" {
			out = append(out, id)
		}
	}
	return out
}

func parseTime(s string) time.Time {
	for _, l := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999", "2006-01-02T15:04:05"} {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func str(r json.RawMessage) string {
	var s string
	_ = json.Unmarshal(r, &s)
	return s
}

func (c *Client) parseWorkItem(r json.RawMessage, projectID string) (WorkItem, error) {
	var w struct {
		ID          string          `json:"id"`
		Project     json.RawMessage `json:"project"`
		Sequence    int             `json:"sequence_id"`
		Name        string          `json:"name"`
		DescHTML    json.RawMessage `json:"description_html"`
		Desc        json.RawMessage `json:"description"`
		State       json.RawMessage `json:"state"`
		StateID     string          `json:"state_id"`
		Priority    *string         `json:"priority"`
		Assignees   json.RawMessage `json:"assignees"`
		AssigneeIDs json.RawMessage `json:"assignee_ids"`
		Labels      json.RawMessage `json:"labels"`
		LabelIDs    json.RawMessage `json:"label_ids"`
		Parent      json.RawMessage `json:"parent"`
		ParentIDRaw json.RawMessage `json:"parent_id"`
		CreatedAt   string          `json:"created_at"`
		UpdatedAt   string          `json:"updated_at"`
	}
	if err := json.Unmarshal(r, &w); err != nil {
		return WorkItem{}, err
	}
	if w.ID == "" {
		return WorkItem{}, &APIError{Status: 200, Message: "unexpected response"}
	}
	it := WorkItem{ID: w.ID, ProjectID: idOf(w.Project), Sequence: w.Sequence, Name: w.Name,
		StateID: w.StateID, CreatedAt: parseTime(w.CreatedAt), UpdatedAt: parseTime(w.UpdatedAt)}
	if it.ProjectID == "" {
		it.ProjectID = projectID
	}
	if it.StateID == "" {
		it.StateID = idOf(w.State)
	}
	if it.DescHTML = str(w.DescHTML); it.DescHTML == "" {
		it.DescHTML = str(w.Desc)
	}
	if w.Priority != nil && *w.Priority != "" {
		it.Priority = *w.Priority
	} else {
		it.Priority = "none"
	}
	it.AssigneeIDs = idList(w.AssigneeIDs)
	if it.AssigneeIDs == nil {
		it.AssigneeIDs = idList(w.Assignees)
	}
	it.LabelIDs = idList(w.LabelIDs)
	if it.LabelIDs == nil {
		it.LabelIDs = idList(w.Labels)
	}
	if it.ParentID = idOf(w.Parent); it.ParentID == "" {
		it.ParentID = idOf(w.ParentIDRaw)
	}
	return it, nil
}

// ---- API ----
