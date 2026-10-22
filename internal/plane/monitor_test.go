package plane

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMonitorList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "57")
		if r.URL.Query().Get("cursor") == "" {
			fmt.Fprint(w, `{"results":[{"id":"a"}],"next_cursor":"c2","next_page_results":true,"total_pages":2,"total_count":2}`)
			return
		}
		fmt.Fprint(w, `{"results":[{"id":"b"}],"next_cursor":"c3","next_page_results":false,"total_pages":2,"total_count":2}`)
	}))
	defer srv.Close()
	m := NewMonitor()
	c := New(Options{BaseURL: srv.URL, APIKey: "k", Workspace: "ws", Monitor: m})
	if _, err := c.getList(context.Background(), srv.URL+"/api/v1/workspaces/ws/projects/0f1e2d3c-4b5a-6978-8796-a5b4c3d2e1f0/work-items/"); err != nil {
		t.Fatal(err)
	}
	s := m.Snapshot()
	if s.Total != 2 || s.Errors != 0 || s.RateRemaining != 57 || len(s.InFlight) != 0 || len(s.Lists) != 0 {
		t.Fatalf("%+v", s)
	}
	if p := c.shortPath(srv.URL + "/api/v1/workspaces/ws/projects/0f1e2d3c-4b5a-6978-8796-a5b4c3d2e1f0/work-items/"); p != "/projects/0f1e2d3c/work-items/" {
		t.Fatal(p)
	}
	if got := resource("/projects/0f1e2d3c/work-items/"); got != "work-items" {
		t.Fatal(got)
	}
	if got := resource("/projects/"); got != "projects" {
		t.Fatal(got)
	}
}

func TestMonitorNil(t *testing.T) {
	var m *Monitor
	m.finish(m.start(Request{}), 200, 1, "", nil)
	if s := m.Snapshot(); s.Total != 0 || s.RateRemaining != -1 {
		t.Fatalf("%+v", s)
	}
}

func TestWorkItemsUpdatedStopsAtOlder(t *testing.T) {
	var pages int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages++
		if r.URL.Query().Get("order_by") != "-updated_at" {
			t.Errorf("order_by %q", r.URL.Query().Get("order_by"))
		}
		fmt.Fprint(w, `{"results":[{"id":"a","updated_at":"2026-10-22T12:00:00Z"},{"id":"b","updated_at":"2026-10-20T12:00:00Z"}],
			"next_cursor":"c2","next_page_results":true,"total_pages":38,"total_count":3788}`)
	}))
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, APIKey: "k", Workspace: "ws"})
	its, total, err := c.WorkItemsUpdated(context.Background(), "p", time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC))
	if err != nil || pages != 1 || total != 3788 || len(its) != 1 || its[0].ID != "a" {
		t.Fatal(err, pages, total, its)
	}
}
