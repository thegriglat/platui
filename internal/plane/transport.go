package plane

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// maxWait is the upper bound of the wait taken from headers.
const maxWait = 60 * time.Second

// retryDelay is the wait before a retry: Retry-After, X-RateLimit-Reset, otherwise 1s, 2s, 4s.
func retryDelay(resp *http.Response, attempt int) time.Duration {
	if v := strings.TrimSpace(resp.Header.Get("Retry-After")); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil && n >= 0 {
			return min(time.Duration(n*float64(time.Second)), maxWait)
		}
		if t, err := http.ParseTime(v); err == nil {
			return min(max(time.Until(t), 0), maxWait)
		}
	}
	if v := strings.TrimSpace(resp.Header.Get("X-RateLimit-Reset")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			d := time.Until(time.Unix(n, 0))
			if d < 0 {
				d = 0
			}
			return min(d, maxWait)
		}
	}
	return time.Second << uint(attempt)
}

// errorMessage is the error text from the body (detail, error, message) or the status.
func errorMessage(body []byte, status int) string {
	var m map[string]any
	if json.Unmarshal(body, &m) == nil {
		for _, k := range []string{"detail", "error", "message"} {
			if s, ok := m[k].(string); ok && s != "" {
				return s
			}
		}
	}
	t := http.StatusText(status)
	if t == "" {
		t = strconv.Itoa(status)
	}
	return t
}

// sleepCtx waits for d, interrupted by ctx.
func (c *Client) sleepCtx(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	done := make(chan struct{})
	go func() { c.sleep(d); close(done) }()
	select {
	case <-done:
		return ctx.Err()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// do performs a request with 429/503 retries and returns the 2xx body.
func (c *Client) do(ctx context.Context, method, u string, q url.Values, body any) ([]byte, error) {
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	var payload []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		payload = b
	}
	for attempt := 0; ; attempt++ {
		var rd io.Reader
		if payload != nil {
			rd = bytes.NewReader(payload)
		}
		req, err := http.NewRequestWithContext(ctx, method, u, rd)
		if err != nil {
			return nil, errors.New("plane: bad request URL")
		}
		req.Header.Set("X-API-Key", c.key)
		req.Header.Set("Accept", "application/json")
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := c.hc.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			var ue *url.Error
			if errors.As(err, &ue) {
				err = ue.Err // URL without query; the key is not passed in the URL
			}
			return nil, fmt.Errorf("plane: %s request failed: %w", method, err)
		}
		data, rerr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if rerr != nil {
			return nil, fmt.Errorf("plane: reading response: %w", rerr)
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return data, nil
		}
		ae := &APIError{Status: resp.StatusCode, Message: c.redact(errorMessage(data, resp.StatusCode))}
		if (resp.StatusCode == 429 || resp.StatusCode == 503) && attempt < c.retries {
			if err := c.sleepCtx(ctx, retryDelay(resp, attempt)); err != nil {
				return nil, err
			}
			continue
		}
		return nil, ae
	}
}

// redact strips the client key from text received from the server.
func (c *Client) redact(s string) string {
	if c.key == "" {
		return s
	}
	return strings.ReplaceAll(s, c.key, "***")
}
