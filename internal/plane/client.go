package plane

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client реализует API.
type Client struct {
	base, key, ws, web string
	hc                 *http.Client
	retries            int
	sleep              func(time.Duration)
}

// Options — параметры клиента.
// MaxRetries: <0 — без повторов, 0 — 3 повтора по умолчанию.
type Options struct {
	BaseURL, APIKey, Workspace, WebURL string
	HTTPClient                         *http.Client
	MaxRetries                         int
	Sleep                              func(time.Duration)
}

var _ API = (*Client)(nil)

// New создаёт клиент.
func New(o Options) *Client {
	c := &Client{
		base: strings.TrimRight(o.BaseURL, "/"), key: o.APIKey, ws: o.Workspace,
		web: strings.TrimRight(o.WebURL, "/"), hc: o.HTTPClient, retries: o.MaxRetries, sleep: o.Sleep,
	}
	if c.hc == nil {
		c.hc = &http.Client{Timeout: 30 * time.Second}
	}
	if c.retries == 0 {
		c.retries = 3
	}
	if c.sleep == nil {
		c.sleep = time.Sleep
	}
	return c
}

// root — URL вне workspace: <base_url>/api/v1/…/.
func (c *Client) root(parts ...string) string {
	p := c.base + "/api/v1/"
	for _, s := range parts {
		p += url.PathEscape(s) + "/"
	}
	return p
}

// path — URL под workspace.
func (c *Client) path(parts ...string) string {
	p := c.base + "/api/v1/workspaces/" + url.PathEscape(c.ws) + "/"
	for _, s := range parts {
		p += url.PathEscape(s) + "/"
	}
	return p
}
