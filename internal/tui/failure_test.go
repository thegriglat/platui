package tui

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/cache"
	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/forge/forgetest"
	"github.com/thegriglat/platui/internal/plane"
	"github.com/thegriglat/platui/internal/plane/planetest"
	"github.com/thegriglat/platui/internal/text"
)

// Audit PT-45: Plane API unavailability at the TUI level. GAP tests check
// the expected behavior and are skipped until the behavior is fixed.

const flKey = "plane_api_secret"

var (
	flNet  = errors.New("plane: GET request failed: dial tcp: lookup plane.invalid: no such host")
	flAuth = &plane.APIError{Status: 401, Message: "Given API token is not valid"}
)

func flGap(t *testing.T, ok bool, why string) {
	t.Helper()
	if !ok {
		t.Fatal(why)
	}
}

func flFake() *planetest.Fake {
	api := planetest.NewFake()
	api.MeV = plane.Member{ID: "u-me", DisplayName: "Me"}
	api.ProjectsV = []plane.Project{{ID: "p1", Identifier: "PROJ", Name: "One"}}
	api.StatesV["p1"] = []plane.State{
		{ID: "s1", Name: "Backlog", Group: "backlog", Sequence: 1}, {ID: "s2", Name: "To Do", Group: "unstarted", Sequence: 2},
		{ID: "s3", Name: "In Progress", Group: "started", Sequence: 3}, {ID: "s4", Name: "Done", Group: "completed", Sequence: 4}}
	api.MembersV["p1"] = []plane.Member{{ID: "u-me", DisplayName: "Me"}}
	api.LabelsV["p1"] = []plane.Label{{ID: "l1", Name: "Bug"}}
	for i, st := range []string{"s1", "s2", "s2", "s3", "s4"} {
		api.ItemsV["p1"] = append(api.ItemsV["p1"], plane.WorkItem{ID: fmt.Sprintf("i%d", i+1), ProjectID: "p1", Sequence: i + 1,
			Name: fmt.Sprintf("Task number %d", i+1), StateID: st, Priority: "low", UpdatedAt: t0.Add(time.Duration(i) * time.Minute)})
	}
	return api
}

func (f *flEnv) setErr(method string, err error) {
	f.api.Mu.Lock()
	defer f.api.Mu.Unlock()
	if err == nil {
		delete(f.api.Err, method)
	} else {
		f.api.Err[method] = err
	}
}

func (f *flEnv) failAll(err error) {
	for _, m := range []string{"Me", "Projects", "States", "Members", "Labels", "WorkItems", "WorkItem", "WorkItemByKey",
		"Comments", "CreateWorkItem", "UpdateWorkItem", "CreateComment"} {
		f.setErr(m, err)
	}
}

type flEnv struct {
	api   *planetest.Fake
	store *cache.Mem
	cfg   *config.Config
}

// flModel builds a model over api and a shared store; interval 0 means no background polling.
func flModel(t *testing.T, api plane.API, store cache.Store, interval time.Duration, edit func(*config.Config)) *model {
	t.Helper()
	return flModelWith(t, api, store, interval, edit, feed, run)
}

// flRunLong is like run but waits up to 5 s (real network failures: DNS, client timeout).
func flRunLong(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	select {
	case msg := <-ch:
		if b, ok := msg.(tea.BatchMsg); ok {
			var out []tea.Msg
			for _, c := range b {
				out = append(out, flRunLong(t, c)...)
			}
			return out
		}
		if _, tick := msg.(previewTickMsg); tick || msg == nil {
			return nil // the periodic preview tick restarts itself
		}
		return []tea.Msg{msg}
	case <-time.After(5 * time.Second):
		t.Fatal("command did not finish within 5 s: hang")
		return nil
	}
}

func flFeedLong(t *testing.T, m *model, msg tea.Msg) {
	t.Helper()
	_, cmd := m.Update(msg)
	for _, r := range flRunLong(t, cmd) {
		flFeedLong(t, m, r)
	}
}

func flModelWith(t *testing.T, api plane.API, store cache.Store, interval time.Duration, edit func(*config.Config),
	feedFn func(*testing.T, *model, tea.Msg), runFn func(*testing.T, tea.Cmd) []tea.Msg) *model {
	t.Helper()
	cfg := &config.Config{Workspace: "ws", DefaultProject: "PROJ", Refresh: config.Refresh{Interval: &interval},
		Tabs: config.Tabs{HideGroups: []string{"cancelled"}}}
	if edit != nil {
		edit(cfg)
	}
	a := app.New(app.Deps{API: api, Forge: &forgetest.Fake{}, Store: store, Config: cfg, Now: func() time.Time { return t0 }})
	m := newModel(Deps{App: a, Config: cfg, Now: func() time.Time { return t0 }, Start: StartAt{Project: "PROJ"}})
	m.s.bell = func() {}
	feedFn(t, m, tea.WindowSizeMsg{Width: 120, Height: 30})
	for _, r := range runFn(t, m.Init()) {
		feedFn(t, m, r)
	}
	return m
}

func flEnvNew(t *testing.T, interval time.Duration, edit func(*config.Config)) (*model, *flEnv) {
	t.Helper()
	e := &flEnv{api: flFake(), store: cache.NewMem(func() time.Time { return t0 })}
	m := flModel(t, e.api, e.store, interval, edit)
	if len(m.s.items) != 5 {
		t.Fatalf("warm-up: %d tasks, status %+v", len(m.s.items), m.s.status)
	}
	return m, e
}

// flFeedAll feeds a command result (unwrapping Batch) into the model.
func flFeedAll(t *testing.T, m *model, cmd tea.Cmd) {
	t.Helper()
	for _, r := range run(t, cmd) {
		feed(t, m, r)
	}
}

func flStatusClean(t *testing.T, m *model) {
	t.Helper()
	if strings.Contains(m.s.status.text, flKey) || strings.Contains(view(m), flKey) {
		t.Fatalf("API key on screen: %q", m.s.status.text)
	}
}

func TestFailureTUIBootNoNetwork(t *testing.T) {
	api := flFake()
	api.Err["Me"] = flNet
	m := flModel(t, api, cache.NewMem(nil), 0, nil)
	if !m.s.status.isErr || !strings.Contains(m.s.status.text, "no such host") || len(m.s.items) != 0 {
		t.Fatalf("status %+v, tasks %d", m.s.status, len(m.s.items))
	}
	if !strings.Contains(view(m), "no such host") {
		t.Fatalf("error not on screen:\n%s", view(m))
	}
	// actions do not crash without a project
	press(t, m, "R", "n", "esc", "s", "a", "e", "c", "v", "esc", "tab", "down", "enter", "esc")
}

func TestFailureTUIBootUnauthorized(t *testing.T) {
	api := flFake()
	api.Err["Me"] = flAuth
	m := flModel(t, api, cache.NewMem(nil), 0, nil)
	if !m.s.status.isErr || !strings.Contains(m.s.status.text, "401") || !strings.Contains(m.s.status.text, "not valid") {
		t.Fatalf("status %+v", m.s.status)
	}
	flStatusClean(t, m)
}

func TestFailureTUIBootProjectsDown(t *testing.T) {
	api := flFake()
	api.Err["Projects"] = &plane.APIError{Status: 503, Message: "Service Unavailable"}
	m := flModel(t, api, cache.NewMem(nil), 0, nil)
	if !m.s.status.isErr || !strings.Contains(m.s.status.text, "503") {
		t.Fatalf("status %+v", m.s.status)
	}
}

// The process starts with the API unavailable and a cache from a previous session on disk.
func TestFailureTUIOfflineStartFromCache(t *testing.T) {
	m1, e := flEnvNew(t, 0, nil)
	_ = m1
	e.failAll(flNet)
	m := flModel(t, e.api, e.store, 0, nil)
	if !m.s.status.isErr {
		t.Fatalf("error not shown: %+v", m.s.status)
	}
	flGap(t, len(m.s.items) == 5 && strings.Contains(m.s.status.text, "offline") && strings.Contains(m.s.status.text, t0.Format("15:04")),
		fmt.Sprintf("start with the API unavailable does not use the cache and sets no \"offline, data from HH:MM\" marker (tasks on screen: %d, status %q): bootstrap fails on Me, Projects/Meta do not read the disk; fix: app (Me/Projects/Meta cache) + tui/msgs.go bootstrap, status", len(m.s.items), m.s.status.text))
}

// Me/Projects/Meta are available, only tasks fail to respond: the cache is shown, the error goes to the status.
func TestFailureTUIItemsDownShowsCache(t *testing.T) {
	_, e := flEnvNew(t, 0, nil)
	e.setErr("WorkItems", flNet)
	m := flModel(t, e.api, e.store, 0, nil)
	if len(m.s.items) != 5 {
		t.Fatalf("cache not shown: %d tasks", len(m.s.items))
	}
	if !m.s.status.isErr || m.s.loading {
		t.Fatalf("status %+v loading=%v", m.s.status, m.s.loading)
	}
	if !strings.Contains(view(m), "Task number 1") {
		t.Fatalf("table is empty:\n%s", view(m))
	}
	flGap(t, strings.Contains(m.s.status.text, "offline"),
		fmt.Sprintf("no \"offline, data from HH:MM\" marker on a refresh error (status: %q); fix: tui/update.go onItems + app (list time)", m.s.status.text))
}

func TestFailureTUIReloadKeepsDataAndRecovers(t *testing.T) {
	for _, tc := range []struct {
		method string
		err    error
		want   string
	}{{"WorkItems", flNet, "no such host"}, {"States", flAuth, "401"}, {"Labels", &plane.APIError{Status: 500, Message: "Internal Server Error"}, "500"}} {
		t.Run(tc.method, func(t *testing.T) {
			m, e := flEnvNew(t, 0, nil)
			e.api.ItemsV["p1"] = e.api.ItemsV["p1"][:1] // the server has already "changed", but the refresh will be cut off
			e.setErr(tc.method, tc.err)
			press(t, m, "R")
			if len(m.s.items) != 5 || !m.s.status.isErr || !strings.Contains(m.s.status.text, tc.want) || m.s.loading {
				t.Fatalf("tasks %d, status %+v, loading=%v", len(m.s.items), m.s.status, m.s.loading)
			}
			if len(m.s.tabs) == 0 {
				t.Fatal("tabs vanished")
			}
			flStatusClean(t, m)
			e.setErr(tc.method, nil) // network is back
			press(t, m, "R")
			if len(m.s.items) != 1 || m.s.status.isErr || !strings.Contains(m.s.status.text, "updated: 1") {
				t.Fatalf("recovery: tasks %d, status %+v", len(m.s.items), m.s.status)
			}
		})
	}
}

func TestFailureTUIPollBackoffAndRecovery(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want []time.Duration
	}{
		{"net", flNet, []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 5 * time.Minute, 5 * time.Minute}},
		{"429", &plane.APIError{Status: 429, Message: "throttled"}, []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 10 * time.Minute, 10 * time.Minute}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, e := flEnvNew(t, 30*time.Second, nil)
			if m.s.pollEvery != 30*time.Second {
				t.Fatalf("base %v", m.s.pollEvery)
			}
			e.setErr("WorkItems", tc.err)
			for i, want := range tc.want {
				feed(t, m, pollMsg{})
				if m.s.pollEvery != want || !m.s.status.isErr || m.s.polling {
					t.Fatalf("step %d: interval %v (want %v), status %+v, polling=%v", i, m.s.pollEvery, want, m.s.status, m.s.polling)
				}
			}
			if len(m.s.items) != 5 {
				t.Fatal("data lost")
			}
			e.setErr("WorkItems", nil)
			feed(t, m, pollMsg{})
			if m.s.pollEvery != 30*time.Second {
				t.Fatalf("interval after recovery: %v", m.s.pollEvery)
			}
			flGap(t, !m.s.status.isErr,
				fmt.Sprintf("after the connection recovers the red error stays in the status line until the next message (%q); fix: tui/update.go onItems (clear / \"connection restored\" on a successful poll after an error)", m.s.status.text))
		})
	}
}

func TestFailureTUIPollDoesNotPileUp(t *testing.T) {
	m, e := flEnvNew(t, 30*time.Second, nil)
	e.setErr("WorkItems", flNet)
	n := e.api.Calls
	before := len(n)
	feed(t, m, pollMsg{})
	// while a poll is in flight (polling), new pollMsg do not spawn requests
	m.s.polling = true
	for i := 0; i < 5; i++ {
		_, cmd := m.Update(pollMsg{})
		if cmd != nil {
			t.Fatal("pollMsg spawned a command during an active poll")
		}
	}
	e.api.Mu.Lock()
	got := len(e.api.Calls) - before
	e.api.Mu.Unlock()
	if got != 1 {
		t.Fatalf("requests %d", got)
	}
}

func TestFailureTUIEditsDoNotChangeItems(t *testing.T) {
	m, e := flEnvNew(t, 0, nil)
	m.s.me = plane.Member{ID: "u-me"}
	e.setErr("UpdateWorkItem", &plane.APIError{Status: 503, Message: "Service Unavailable"})
	snap := func(id string) plane.WorkItem { it, _ := m.s.itemByID(id); return it }
	before := snap("i2")

	// "assign to me"
	m.s.cursor = "i2"
	press(t, m, "a")
	// state and priority (the final commands of the list pickers)
	st, pr := "s4", "urgent"
	flFeedAll(t, m, patchWith(m.s, "i2", plane.WorkItemPatch{StateID: &st}, "state: Done"))
	flFeedAll(t, m, patchWith(m.s, "i2", plane.WorkItemPatch{Priority: &pr}, "priority: urgent"))
	flFeedAll(t, m, patchItem(m.s, "i2", plane.WorkItemPatch{StateID: &st}))
	after := snap("i2")
	if after.StateID != before.StateID || after.Priority != before.Priority || len(after.AssigneeIDs) != len(before.AssigneeIDs) {
		t.Fatalf("task changed: %+v", after)
	}
	if !m.s.status.isErr || !strings.Contains(m.s.status.text, "503") {
		t.Fatalf("status %+v", m.s.status)
	}
	flStatusClean(t, m)
	// success after recovery
	e.setErr("UpdateWorkItem", nil)
	press(t, m, "a")
	if it := snap("i2"); len(it.AssigneeIDs) != 1 || m.s.status.isErr {
		t.Fatalf("after recovery: %+v %+v", it, m.s.status)
	}
}

func TestFailureTUIKanbanMoveRollback(t *testing.T) {
	for _, err := range []error{flNet, flAuth} {
		m, e := flEnvNew(t, 0, nil)
		press(t, m, "v")
		e.setErr("UpdateWorkItem", err)
		special(t, m, tea.KeyShiftRight)
		if got := stateOf(m, "i1"); got != "s1" {
			t.Fatalf("not rolled back: %s", got)
		}
		if !m.s.status.isErr || !strings.Contains(m.s.status.text, err.Error()) {
			t.Fatalf("status %+v", m.s.status)
		}
		if k := m.top().(*kanban); k.sel != "i1" || k.col != 0 {
			t.Fatalf("selection col=%d sel=%q", k.col, k.sel)
		}
	}
}

func TestFailureTUICreateKeepsInput(t *testing.T) {
	m, e := flEnvNew(t, 0, nil)
	e.setErr("CreateWorkItem", flNet)
	press(t, m, "n", "Broken task !high")
	press(t, m, "enter")
	in, ok := m.top().(*inputScreen)
	if !ok || string(in.text) != "Broken task !high" || in.busy {
		t.Fatalf("input lost or hung: %T %+v", m.top(), in)
	}
	if !m.s.status.isErr || !strings.Contains(m.s.status.text, "no such host") || len(m.s.items) != 5 {
		t.Fatalf("status %+v, tasks %d", m.s.status, len(m.s.items))
	}
	e.setErr("CreateWorkItem", nil) // retry without retyping
	press(t, m, "enter")
	if _, ok := m.top().(*inputScreen); ok || m.s.status.isErr {
		t.Fatalf("retry failed: %T %+v", m.top(), m.s.status)
	}
}

// Text from $EDITOR must not be lost on a submit error.
func TestFailureTUIEditorTextSurvivesAPIError(t *testing.T) {
	for _, tc := range []struct{ key, method string }{{"c", "CreateComment"}, {"e", "UpdateWorkItem"}} {
		t.Run(tc.key, func(t *testing.T) {
			syncExec(t)
			var path string
			inner := runEditor
			runEditor = func(sess *text.EditSession, editor string, fn tea.ExecCallback) tea.Cmd {
				path = sess.Path
				return inner(sess, editor, fn)
			}
			m, e := flEnvNew(t, 0, func(c *config.Config) { c.Editor = `printf 'precious text\n' >` })
			t.Cleanup(func() {
				if path != "" {
					os.Remove(path)
				}
			})
			e.setErr(tc.method, flNet)
			m.s.cursor = "i1"
			press(t, m, tc.key)
			if !m.s.status.isErr || !strings.Contains(m.s.status.text, "no such host") {
				t.Fatalf("error not shown: %+v", m.s.status)
			}
			b, err := os.ReadFile(path)
			flGap(t, err == nil && strings.Contains(string(b), "precious text"),
				"on a comment/description submit error the editor temp file is removed (finishEdit: defer sess.Close()), losing the typed text; fix: tui/edit.go (keep the file on an API error, show the path or save a draft)")
		})
	}
}

func TestFailureTUINoNetworkViewStaysAlive(t *testing.T) {
	m, e := flEnvNew(t, 0, nil)
	e.failAll(flNet)
	// every action that goes to the network must return control and show an error
	m.s.cursor = "i1"
	for _, keys := range [][]string{{"R"}, {"a"}, {"enter"}, {"esc"}, {"c"}, {"v"}, {"esc"}} {
		press(t, m, keys...)
	}
	if v := view(m); !strings.Contains(v, "Task number") || m.s.loading {
		t.Fatalf("screen broken (loading=%v):\n%s", m.s.loading, v)
	}
}

// ---- end to end: a real plane.Client against httptest ----

func flReal(t *testing.T, base string, timeout time.Duration) *model {
	t.Helper()
	api := plane.New(plane.Options{BaseURL: base, APIKey: flKey, Workspace: "ws",
		HTTPClient: &http.Client{Timeout: timeout}, Sleep: func(time.Duration) {}})
	return flModelWith(t, api, cache.NewMem(nil), 0, nil, flFeedLong, flRunLong)
}

func TestFailureTUIE2EErrorsReadable(t *testing.T) {
	hang := make(chan struct{})
	var hits int32
	mk := func(h http.HandlerFunc) string {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { atomic.AddInt32(&hits, 1); h(w, r) }))
		t.Cleanup(srv.Close)
		return srv.URL
	}
	closed := httptest.NewServer(http.NotFoundHandler())
	closedURL := closed.URL
	closed.Close()
	tlsSrv := httptest.NewTLSServer(http.NotFoundHandler())
	defer tlsSrv.Close()
	t.Cleanup(func() { close(hang) })
	cases := map[string]struct{ base, want string }{
		"401": {mk(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
			fmt.Fprint(w, `{"detail":"Given API token is not valid"}`)
		}), "401"},
		"403":     {mk(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403) }), "403"},
		"500":     {mk(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }), "500"},
		"html":    {mk(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "<html>captive portal</html>") }), "invalid character"},
		"refused": {closedURL, "request failed"},
		"tls":     {tlsSrv.URL, "certificate"},
		"dns":     {"http://plane.invalid", "request failed"},
		"timeout": {mk(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-hang:
			case <-r.Context().Done():
			}
		}), "Timeout"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			to := 3 * time.Second
			if name == "timeout" {
				to = 100 * time.Millisecond
			}
			m := flReal(t, tc.base, to)
			if d := time.Since(start); d > 5*time.Second {
				t.Fatalf("hang %v", d)
			}
			if !m.s.status.isErr || !strings.Contains(m.s.status.text, tc.want) {
				t.Fatalf("status %+v, want %q", m.s.status, tc.want)
			}
			if strings.Contains(m.s.status.text, "<html>") || strings.Contains(m.s.status.text, tc.base) {
				t.Fatalf("response body or address in the status: %q", m.s.status.text)
			}
			flStatusClean(t, m)
		})
	}
}
