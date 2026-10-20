package plane

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func convState(r json.RawMessage) (State, error) {
	var s struct {
		ID, Name, Group, Color string
		Sequence               json.RawMessage `json:"sequence"`
		Default                bool            `json:"default"`
		IsDefault              bool            `json:"is_default"`
	}
	if err := json.Unmarshal(r, &s); err != nil {
		return State{}, err
	}
	st := State{ID: s.ID, Name: s.Name, Group: s.Group, Color: s.Color, Default: s.Default || s.IsDefault}
	var f float64
	if json.Unmarshal(s.Sequence, &f) == nil {
		st.Sequence = f
	} else if t := str(s.Sequence); t != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
			st.Sequence = f
		}
	}
	return st, nil
}

// WorkItemByKey — метод API: задача по PROJ-N.
func (c *Client) WorkItemByKey(ctx context.Context, identifier string, seq int) (WorkItem, error) {
	key := strings.ToUpper(identifier) + "-" + strconv.Itoa(seq)
	data, err := c.do(ctx, http.MethodGet, c.path("work-items", key), nil, nil)
	if err != nil {
		return WorkItem{}, err
	}
	it, err := c.parseWorkItem(data, "")
	if err != nil {
		return WorkItem{}, fmt.Errorf("plane: bad work item: %w", err)
	}
	return it, nil
}

// Search — метод API: серверный поиск, limit<=0 → 50.
func (c *Client) Search(ctx context.Context, query, projectID string, limit int) ([]SearchHit, error) {
	if limit <= 0 {
		limit = 50
	}
	q := url.Values{"search": {query}, "limit": {strconv.Itoa(limit)}}
	if projectID != "" {
		q.Set("project_id", projectID)
	}
	data, err := c.do(ctx, http.MethodGet, c.path("work-items", "search"), q, nil)
	if err != nil {
		return nil, err
	}
	var env struct {
		Issues json.RawMessage `json:"issues"`
	}
	raw := data
	if t := strings.TrimSpace(string(data)); strings.HasPrefix(t, "{") {
		if err := json.Unmarshal(data, &env); err != nil {
			return nil, fmt.Errorf("plane: bad search response: %w", err)
		}
		raw = env.Issues
	}
	items, err := flatten(raw)
	if err != nil {
		return nil, err
	}
	out := make([]SearchHit, 0, len(items))
	for _, r := range items {
		var h struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			Sequence   int    `json:"sequence_id"`
			ProjectID  string `json:"project_id"`
			Identifier string `json:"project__identifier"`
		}
		if err := json.Unmarshal(r, &h); err != nil {
			return nil, fmt.Errorf("plane: bad search hit: %w", err)
		}
		out = append(out, SearchHit{ID: h.ID, Name: h.Name, Sequence: h.Sequence, ProjectID: h.ProjectID, ProjectIdentifier: h.Identifier})
	}
	return out, nil
}
