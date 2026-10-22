package plane

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Request is a Plane API request in flight.
type Request struct {
	Start  time.Time
	Method string
	Path   string // relative to the workspace, ids shortened
	Page   int    // page number of a list read (1-based); 0 — not a list
}

// ListProgress is a paginated list read in progress.
type ListProgress struct {
	Resource    string    // last path segment: "work-items", "states", ...
	Start       time.Time // when the first page was requested
	Page, Pages int       // pages read / total (0 — unknown)
	Got, Count  int       // elements read / total (0 — unknown)
}

// MonitorSnapshot is a copy of the monitor state.
type MonitorSnapshot struct {
	Total, Errors int
	Bytes         int64
	RateRemaining int // X-RateLimit-Remaining of the last response; -1 — unknown
	InFlight      []Request
	Lists         []ListProgress
}

// Monitor records Plane API activity: counters, requests in flight and list progress. Safe for concurrent use; a nil *Monitor does nothing.
type Monitor struct {
	mu       sync.Mutex
	next     int
	total    int
	errs     int
	bytes    int64
	rate     int
	inflight map[int]Request
	lists    map[string]*ListProgress
}

// NewMonitor returns an empty monitor.
func NewMonitor() *Monitor {
	return &Monitor{rate: -1, inflight: map[int]Request{}, lists: map[string]*ListProgress{}}
}

func (m *Monitor) start(r Request) int {
	if m == nil {
		return 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.next++
	m.inflight[m.next] = r
	return m.next
}

func (m *Monitor) finish(id, status, bytes int, rate string, err error) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.inflight[id]; !ok {
		return
	}
	delete(m.inflight, id)
	m.total++
	m.bytes += int64(bytes)
	if err != nil || status >= 400 {
		m.errs++
	}
	if n, err := strconv.Atoi(strings.TrimSpace(rate)); err == nil {
		m.rate = n
	}
}

func (m *Monitor) listPage(key string, p ListProgress) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lists[key] = &p
}

func (m *Monitor) listDone(key string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.lists, key)
}

// Snapshot returns a copy of the current state.
func (m *Monitor) Snapshot() MonitorSnapshot {
	if m == nil {
		return MonitorSnapshot{RateRemaining: -1}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	s := MonitorSnapshot{Total: m.total, Errors: m.errs, Bytes: m.bytes, RateRemaining: m.rate}
	for _, r := range m.inflight {
		s.InFlight = append(s.InFlight, r)
	}
	for _, p := range m.lists {
		s.Lists = append(s.Lists, *p)
	}
	return s
}

var (
	uuidRe  = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	shortID = regexp.MustCompile(`^[0-9a-f]{8}$`)
)

// shortPath is u relative to the workspace API root, with ids cut to 8 characters.
func (c *Client) shortPath(u string) string {
	p := u
	if pu, err := url.Parse(u); err == nil {
		p = pu.Path
	}
	p = strings.TrimPrefix(p, strings.TrimPrefix(c.base, mustHost(c.base)))
	p = strings.TrimPrefix(p, "/api/v1")
	p = strings.TrimPrefix(p, "/workspaces/"+c.ws)
	return uuidRe.ReplaceAllStringFunc(p, func(s string) string { return s[:8] })
}

// resource is the last path segment that is not an id.
func resource(path string) string {
	segs := strings.Split(strings.Trim(path, "/"), "/")
	for i := len(segs) - 1; i >= 0; i-- {
		if s := segs[i]; s != "" && !shortID.MatchString(s) {
			return s
		}
	}
	return path
}

func mustHost(base string) string {
	if pu, err := url.Parse(base); err == nil && pu.Host != "" {
		return pu.Scheme + "://" + pu.Host
	}
	return ""
}
