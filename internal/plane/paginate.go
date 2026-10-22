package plane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// getList reads all pages of a list; elems are the raw elements.
func (c *Client) getList(ctx context.Context, u string) ([]json.RawMessage, error) {
	all, _, err := c.getPages(ctx, u, nil, nil)
	return all, err
}

// getPages reads pages of a list with extra query parameters until stop(page) is true
// or the pages end; total is total_count of the first page (0 — absent).
func (c *Client) getPages(ctx context.Context, u string, extra url.Values, stop func([]json.RawMessage) bool) (all []json.RawMessage, total int, err error) {
	cursor := ""
	key := fmt.Sprintf("%p", &all) // unique per read: parallel reads of one list are separate
	defer c.mon.listDone(key)
	start := time.Now()
	for page := 1; ; page++ {
		q := url.Values{"per_page": {"100"}}
		for k, v := range extra {
			q[k] = v
		}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		data, err := c.doPage(ctx, http.MethodGet, u, q, nil, page)
		if err != nil {
			return nil, 0, err
		}
		items, next, more, err := splitPage(data)
		if err != nil {
			return nil, 0, err
		}
		all = append(all, items...)
		pages, count := pageTotals(data)
		if page == 1 {
			total = count
		}
		if stop != nil { // a partial read: the list totals are not its own
			pages, count = 0, 0
		}
		c.mon.listPage(key, ListProgress{Resource: resource(c.shortPath(u)), Start: start, Page: page, Pages: pages, Got: len(all), Count: count})
		if !more || next == "" || next == cursor || (stop != nil && stop(items)) {
			return all, total, nil
		}
		cursor = next
	}
}

func splitPage(data []byte) (items []json.RawMessage, next string, more bool, err error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		return nil, "", false, nil
	}
	if data[0] == '[' {
		items, err = flatten(data)
		return items, "", false, err
	}
	var env struct {
		Results    json.RawMessage `json:"results"`
		Next       string          `json:"next_cursor"`
		NextResult bool            `json:"next_page_results"`
	}
	if err = json.Unmarshal(data, &env); err != nil {
		return nil, "", false, fmt.Errorf("plane: bad list response: %w", err)
	}
	if env.Results == nil {
		return nil, "", false, &APIError{Status: 200, Message: "unexpected response"}
	}
	items, err = flatten(env.Results)
	return items, env.Next, env.NextResult, err
}

func flatten(data []byte) ([]json.RawMessage, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		return nil, nil
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("plane: bad list response: %w", err)
	}
	var out []json.RawMessage
	for _, e := range arr {
		t := bytes.TrimSpace(e)
		if len(t) > 0 && t[0] == '[' {
			sub, err := flatten(t)
			if err != nil {
				return nil, err
			}
			out = append(out, sub...)
		} else {
			out = append(out, e)
		}
	}
	return out, nil
}

func decodeList[T any](c *Client, ctx context.Context, u string, conv func(json.RawMessage) (T, error)) ([]T, error) {
	raw, err := c.getList(ctx, u)
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(raw))
	for _, r := range raw {
		v, err := conv(r)
		if err != nil {
			return nil, fmt.Errorf("plane: bad item: %w", err)
		}
		out = append(out, v)
	}
	return out, nil
}

// pageTotals reads total_pages and total_count of a paginated response (0 — absent).
func pageTotals(data []byte) (pages, count int) {
	var env struct {
		Pages int `json:"total_pages"`
		Count int `json:"total_count"`
	}
	if json.Unmarshal(data, &env) == nil {
		return env.Pages, env.Count
	}
	return 0, 0
}
