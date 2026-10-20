package plane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// getList читает все страницы списка; elems — сырые элементы.
func (c *Client) getList(ctx context.Context, u string) ([]json.RawMessage, error) {
	var all []json.RawMessage
	cursor := ""
	for {
		q := url.Values{"per_page": {"100"}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		data, err := c.do(ctx, http.MethodGet, u, q, nil)
		if err != nil {
			return nil, err
		}
		items, next, more, err := splitPage(data)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if !more || next == "" || next == cursor {
			return all, nil
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
