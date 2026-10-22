package tui

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/cli/browser"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/search"
	"github.com/thegriglat/platui/internal/text"
)

// Deps are the TUI dependencies (C4 v3).
type Deps struct {
	App       *app.App
	Config    *config.Config
	OpenURL   func(url string) error  // nil → browser.OpenURL
	Clipboard func(text string) error // nil → OSC52 to os.Stderr
	Now       func() time.Time        // nil → time.Now
	Start     StartAt
	Monitor   *plane.Monitor // nil — no API activity in the status line and no API log
}

// StartAt is where to start.
type StartAt struct {
	Project string // identifier; empty — Config.DefaultProject, otherwise the projects screen
	Item    string // "PROJ-42"; empty — from the current branch
}

func (d Deps) withDefaults() Deps {
	if d.Config == nil {
		d.Config = &config.Config{}
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.OpenURL == nil {
		d.OpenURL = func(u string) error {
			browser.Stdout, browser.Stderr = io.Discard, io.Discard
			return browser.OpenURL(u)
		}
	}
	if d.Clipboard == nil {
		d.Clipboard = func(t string) error {
			_, err := io.WriteString(os.Stderr, text.OSC52(t, os.Getenv("TMUX") != ""))
			return err
		}
	}
	return d
}

// Sort orders (sortBy).
const (
	sortAPI = iota
	sortPriority
	sortState
	sortUpdated
	sortID
)

// freshTTL is how long a new/changed task counts as fresh.
const freshTTL = 10 * time.Minute

// session is the shared program state.
type session struct {
	deps          Deps
	st            styles
	width, height int
	me            plane.Member
	projects      []plane.Project
	project       *plane.Project
	meta          app.Meta
	items         []plane.WorkItem
	tabs          []plane.State
	tab           int
	mine          bool
	filter        int
	sortBy        int
	index         *search.Index
	query         string
	hits          map[string]search.Hit
	marks         map[string]bool
	branches      map[string]gitbranch.BranchStatus
	fresh         map[string]time.Time
	focus         int
	cursor        string
	loading       bool
	status        statusLine

	// PT-5a internals
	searching  bool   // a query is being typed (PT-5b): the main screen captures keys
	searchLine string // ready search line from PT-5b; empty — "/ query found N"
	live       bool   // project data was received from the API
	pending    string // ID of the task to select after loading
	openItem   string // ID of the task in the open card (set by openItem, cleared by pop)
	pollArmed  bool
	polling    bool
	pollEvery  time.Duration
	bell       func()
	net        netState // connection and drafts (offline.go)

	panel panelState // pane layout: preview width, zoom (panels.go)
	hist  history    // history of opened cards (history.go)

	ctx    context.Context // cancelled on program exit (own commands)
	cancel context.CancelFunc
}

func newSession(d Deps) *session {
	ctx, cancel := context.WithCancel(context.Background())
	return &session{deps: d, st: newStyles(d.Config), ctx: ctx, cancel: cancel,
		panel: panelState{pct: clampPct(d.Config.PreviewPercent())}, marks: map[string]bool{}, fresh: map[string]time.Time{},
		branches: map[string]gitbranch.BranchStatus{}, bell: func() { _, _ = io.WriteString(os.Stderr, "\a") }}
}

// resetProject switches the project: data is reset.
func (s *session) resetProject(p plane.Project) {
	s.project = &p
	s.meta, s.items, s.tabs, s.tab = app.Meta{Project: p}, nil, nil, 0
	s.index, s.query, s.hits, s.filter, s.cursor = nil, "", nil, 0, ""
	s.marks, s.fresh, s.branches = map[string]bool{}, map[string]time.Time{}, map[string]gitbranch.BranchStatus{}
	s.live, s.loading, s.searching, s.searchLine, s.polling, s.openItem = false, true, false, "", false, ""
}

// rebuildIndex rebuilds the search index (by titles) and the matches.
func (s *session) rebuildIndex() {
	if s.project == nil {
		return
	}
	s.index = search.New(app.SearchDocs(app.Meta{Project: *s.project}, s.items))
	s.setQuery(s.query)
}

// setQuery sets the query (at most search.MaxQuery runes); a key query does not change the filter.
func (s *session) setQuery(q string) {
	if r := []rune(q); len(r) > search.MaxQuery {
		q = string(r[:search.MaxQuery])
	}
	s.query, s.hits = q, nil
	if q == "" || s.index == nil {
		return
	}
	if _, _, ok := search.ParseKey(q); ok {
		return
	}
	s.hits = map[string]search.Hit{}
	for _, h := range s.index.Search(q) {
		s.hits[h.ID] = h
	}
}

// itemByID returns a project task by ID.
func (s *session) itemByID(id string) (plane.WorkItem, bool) {
	for _, it := range s.items {
		if it.ID == id {
			return it, true
		}
	}
	return plane.WorkItem{}, false
}
