package plane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const ws = "acme"
const pfx = "/api/v1/workspaces/acme/"

func newTest(t *testing.T, h http.HandlerFunc) (*Client, *[]time.Duration) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	var sleeps []time.Duration
	c := New(Options{BaseURL: srv.URL + "/", APIKey: "plane_api_secret", Workspace: ws,
		Sleep: func(d time.Duration) { sleeps = append(sleeps, d) }})
	return c, &sleeps
}

func TestContractPlane(t *testing.T) {
	var pages int32
	c, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "plane_api_secret" {
			w.WriteHeader(401)
			return
		}
		if r.URL.Path != pfx+"projects/" {
			w.WriteHeader(404)
			return
		}
		atomic.AddInt32(&pages, 1)
		if r.URL.Query().Get("per_page") != "100" {
			t.Errorf("per_page = %q", r.URL.Query().Get("per_page"))
		}
		switch r.URL.Query().Get("cursor") {
		case "":
			fmt.Fprint(w, `{"next_cursor":"100:1:0","next_page_results":true,"results":[{"id":"p1","identifier":"A","name":"Alpha"}]}`)
		case "100:1:0":
			fmt.Fprint(w, `{"next_cursor":"100:2:0","next_page_results":true,"results":[{"id":"p2","identifier":"B","name":"Beta"}]}`)
		case "100:2:0":
			fmt.Fprint(w, `{"next_page_results":false,"results":[{"id":"p3","identifier":"C","name":"Gamma"}]}`)
		}
	})
	ps, err := c.Projects(context.Background())
	if err != nil || len(ps) != 3 || ps[2].Identifier != "C" || pages != 3 {
		t.Fatalf("projects=%v err=%v pages=%d", ps, err, pages)
	}

	// 429 -> повтор
	var n int32
	c, sleeps := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&n, 1) == 1 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(429)
			return
		}
		fmt.Fprint(w, `{"id":"u1","display_name":"Ann"}`)
	})
	m, err := c.Me(context.Background())
	if err != nil || m.ID != "u1" || n != 2 || len(*sleeps) != 1 || (*sleeps)[0] != 2*time.Second {
		t.Fatalf("m=%v err=%v n=%d sleeps=%v", m, err, n, *sleeps)
	}

	// 404
	c, _ = newTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		fmt.Fprint(w, `{"detail":"Issue not found"}`)
	})
	_, err = c.WorkItem(context.Background(), "p", "x")
	var ae *APIError
	if !errors.Is(err, ErrNotFound) || !errors.As(err, &ae) || ae.Message != "Issue not found" {
		t.Fatalf("err=%v", err)
	}

	// поля WorkItem
	c, _ = newTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != pfx+"projects/p1/work-items/i1/" {
			t.Errorf("path %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"id":"i1","project":"p1","sequence_id":12,"name":"Fix","description_html":"<p>hi</p>",
		"state":{"id":"s1","name":"Todo"},"priority":null,"assignees":["u1",{"id":"u2"}],"labels":[{"id":"l1"}],
		"created_at":"2026-01-02T03:04:05.123456Z","updated_at":"2026-01-03T00:00:00Z","unknown":1}`)
	})
	it, err := c.WorkItem(context.Background(), "p1", "i1")
	if err != nil {
		t.Fatal(err)
	}
	if it.ID != "i1" || it.ProjectID != "p1" || it.Sequence != 12 || it.Name != "Fix" || it.DescHTML != "<p>hi</p>" ||
		it.StateID != "s1" || it.Priority != "none" || strings.Join(it.AssigneeIDs, ",") != "u1,u2" ||
		strings.Join(it.LabelIDs, ",") != "l1" || it.CreatedAt.Year() != 2026 || it.UpdatedAt.Day() != 3 {
		t.Fatalf("%+v", it)
	}
}

func TestContractPlaneErrors(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		is     error
		msg    string
	}{
		{401, `{"error":"bad key"}`, ErrUnauthorized, "bad key"},
		{403, `{"message":"nope"}`, ErrUnauthorized, "nope"},
		{500, `oops`, nil, "Internal Server Error"},
	} {
		c, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
			io.WriteString(w, tc.body)
		})
		_, err := c.Me(context.Background())
		var ae *APIError
		if !errors.As(err, &ae) || ae.Message != tc.msg || ae.Status != tc.status {
			t.Fatalf("%d: %v", tc.status, err)
		}
		if tc.is != nil && !errors.Is(err, tc.is) {
			t.Fatalf("%d: not %v", tc.status, tc.is)
		}
		if strings.Contains(err.Error(), "secret") {
			t.Fatal("key leaked")
		}
	}
}

func TestContractPlaneRetryExhausted(t *testing.T) {
	var n int32
	c, sleeps := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&n, 1)
		w.WriteHeader(429)
	})
	_, err := c.Me(context.Background())
	if !errors.Is(err, ErrRateLimited) || n != 4 {
		t.Fatalf("err=%v n=%d", err, n)
	}
	want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
	for i, d := range want {
		if (*sleeps)[i] != d {
			t.Fatalf("sleeps=%v", *sleeps)
		}
	}
	// 503 исчерпан -> APIError, не ErrRateLimited
	c, _ = newTest(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) })
	_, err = c.Me(context.Background())
	var ae *APIError
	if !errors.As(err, &ae) || ae.Status != 503 || errors.Is(err, ErrRateLimited) {
		t.Fatalf("err=%v", err)
	}
}

func TestContractPlaneRateLimitReset(t *testing.T) {
	c, sleeps := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Reset", fmt.Sprint(time.Now().Add(time.Hour).Unix()))
		w.WriteHeader(429)
	})
	c.retries = 1
	c.Me(context.Background())
	if len(*sleeps) != 1 || (*sleeps)[0] != 60*time.Second {
		t.Fatalf("%v", *sleeps)
	}
}

func TestContractPlaneContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(429) }))
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, Workspace: ws, Sleep: func(time.Duration) { cancel(); time.Sleep(50 * time.Millisecond) }})
	_, err := c.Me(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestContractPlaneListsAndTolerantParsing(t *testing.T) {
	c, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimPrefix(r.URL.Path, pfx) {
		case "projects/p/states/":
			fmt.Fprint(w, `{"results":[{"id":"s1","name":"Todo","group":"unstarted","color":"#fff"},{"id":"s2","name":"X","group":"weird"}]}`)
		case "projects/p/project-members/":
			fmt.Fprint(w, `[{"id":"u1","display_name":"","first_name":"A","last_name":"B","email":"a@b"},{"member":{"id":"u2","email":"c@d"}}]`)
		case "projects/p/labels/":
			fmt.Fprint(w, `[[{"id":"l1","name":"bug","color":"red"}]]`)
		case "projects/p/work-items/":
			fmt.Fprint(w, `{"results":[{"id":"i1","state_id":"s9","description":"d","assignee_ids":["u1"],"label_ids":["l1"],"priority":"high"}]}`)
		case "projects/p/work-items/i1/comments/":
			fmt.Fprint(w, `{"results":[{"id":"c1","actor":"u1","comment_html":"<p>x</p>","created_at":"2026-02-01T00:00:00Z"},{"id":"c2","created_by":"u2"}]}`)
		}
	})
	ctx := context.Background()
	ss, err := c.States(ctx, "p")
	if err != nil || len(ss) != 2 || ss[1].Group != "weird" {
		t.Fatal(ss, err)
	}
	ms, err := c.Members(ctx, "p")
	if err != nil || ms[0].Name() != "A B" || ms[1].ID != "u2" || ms[1].Name() != "c@d" {
		t.Fatal(ms, err)
	}
	ls, err := c.Labels(ctx, "p")
	if err != nil || len(ls) != 1 || ls[0].Name != "bug" {
		t.Fatal(ls, err)
	}
	ws, err := c.WorkItems(ctx, "p")
	if err != nil || ws[0].StateID != "s9" || ws[0].ProjectID != "p" || ws[0].DescHTML != "d" ||
		ws[0].AssigneeIDs[0] != "u1" || ws[0].LabelIDs[0] != "l1" || ws[0].Priority != "high" {
		t.Fatalf("%+v %v", ws, err)
	}
	cs, err := c.Comments(ctx, "p", "i1")
	if err != nil || cs[0].Author != "u1" || cs[1].Author != "u2" || cs[0].CreatedAt.IsZero() {
		t.Fatal(cs, err)
	}
	if (Member{DisplayName: "dn", Email: "e"}).Name() != "dn" {
		t.Fatal("Name")
	}
}

func TestContractPlaneCreateAndPatch(t *testing.T) {
	var got map[string]any
	var method string
	c, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		got = nil
		json.NewDecoder(r.Body).Decode(&got)
		fmt.Fprint(w, `{"id":"i1","sequence_id":1}`)
	})
	ctx := context.Background()
	if _, err := c.CreateWorkItem(ctx, "p", NewWorkItem{Name: "n", DescHTML: "<p>d</p>"}); err != nil {
		t.Fatal(err)
	}
	if method != "POST" || got["name"] != "n" || got["description_html"] != "<p>d</p>" || len(got) != 2 {
		t.Fatalf("%s %v", method, got)
	}
	st := "s1"
	if _, err := c.UpdateWorkItem(ctx, "p", "i1", WorkItemPatch{StateID: &st, AssigneeIDs: &[]string{"u1"}}); err != nil {
		t.Fatal(err)
	}
	if method != "PATCH" || got["state"] != "s1" || len(got) != 2 || got["assignees"].([]any)[0] != "u1" {
		t.Fatalf("%s %v", method, got)
	}
}

func TestContractPlaneWebURL(t *testing.T) {
	p, it := Project{Identifier: "PT"}, WorkItem{Sequence: 7}
	for _, tc := range []struct {
		o    Options
		want string
	}{
		{Options{BaseURL: "https://api.plane.so", Workspace: "w"}, "https://app.plane.so/w/browse/PT-7/"},
		{Options{BaseURL: "https://plane.x.io/", Workspace: "w"}, "https://plane.x.io/w/browse/PT-7/"},
		{Options{BaseURL: "https://api.plane.so", Workspace: "w", WebURL: "https://web.x/"}, "https://web.x/w/browse/PT-7/"},
	} {
		if got := New(tc.o).WebURL(p, it); got != tc.want {
			t.Errorf("%s != %s", got, tc.want)
		}
	}
}

func TestContractPlaneHTMLToText(t *testing.T) {
	in := "<p>Hello &amp; <b>bye</b></p><p>Line1<br/>Line2</p><ul><li>a</li><li>b &lt;c&gt;</li></ul>"
	got := HTMLToText(in)
	for _, s := range []string{"Hello & bye", "Line1\nLine2", "- a\n- b <c>"} {
		if !strings.Contains(got, s) {
			t.Errorf("missing %q in %q", s, got)
		}
	}
	if HTMLToText("") != "" {
		t.Fatal("empty")
	}
}

func TestContractPlaneMe(t *testing.T) {
	var path string
	c, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		fmt.Fprint(w, `{"id":"u1","display_name":"Ann"}`)
	})
	m, err := c.Me(context.Background())
	if err != nil || m.ID != "u1" || path != "/api/v1/users/me/" {
		t.Fatalf("m=%v err=%v path=%s", m, err, path)
	}
}

func TestContractPlaneRetryAfterCap(t *testing.T) {
	var n int32
	c, sleeps := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&n, 1) == 1 {
			w.Header().Set("Retry-After", "120")
			w.WriteHeader(503)
			return
		}
		fmt.Fprint(w, `{"id":"u1"}`)
	})
	if _, err := c.Me(context.Background()); err != nil || len(*sleeps) != 1 || (*sleeps)[0] != 60*time.Second {
		t.Fatalf("err=%v sleeps=%v", err, *sleeps)
	}
}

func TestContractPlaneCreateComment(t *testing.T) {
	var got map[string]any
	var path, method string
	c, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		path, method = r.URL.Path, r.Method
		json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(201)
		fmt.Fprint(w, `{"id":"c1","actor":"u1","comment_html":"<p>x</p>","created_at":"2026-02-01T00:00:00Z"}`)
	})
	cm, err := c.CreateComment(context.Background(), "p", "i1", "<p>x</p>")
	if err != nil || method != "POST" || path != pfx+"projects/p/work-items/i1/comments/" ||
		len(got) != 1 || got["comment_html"] != "<p>x</p>" || cm.ID != "c1" || cm.Author != "u1" || cm.HTML != "<p>x</p>" {
		t.Fatalf("cm=%+v err=%v %s %s %v", cm, err, method, path, got)
	}
	c, _ = newTest(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(400); fmt.Fprint(w, `{"error":"bad"}`) })
	var ae *APIError
	if _, err := c.CreateComment(context.Background(), "p", "i1", "x"); !errors.As(err, &ae) || ae.Status != 400 {
		t.Fatal(err)
	}
}

func TestContractPlaneCreateFields(t *testing.T) {
	var got map[string]any
	c, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		got = nil
		json.NewDecoder(r.Body).Decode(&got)
		fmt.Fprint(w, `{"id":"i1"}`)
	})
	n := NewWorkItem{Name: "n", StateID: "s", Priority: "high", AssigneeIDs: []string{"u"}, LabelIDs: []string{"l"}}
	if _, err := c.CreateWorkItem(context.Background(), "p", n); err != nil {
		t.Fatal(err)
	}
	if len(got) != 6 || got["state"] != "s" || got["priority"] != "high" || got["assignees"].([]any)[0] != "u" || got["labels"].([]any)[0] != "l" {
		t.Fatalf("%v", got)
	}
	if _, err := c.CreateWorkItem(context.Background(), "p", NewWorkItem{Name: "n"}); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"state", "priority", "assignees", "labels"} {
		if _, ok := got[k]; ok {
			t.Fatalf("%s sent: %v", k, got)
		}
	}
}

func TestContractPlanePatchFields(t *testing.T) {
	var got map[string]any
	c, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		got = nil
		json.NewDecoder(r.Body).Decode(&got)
		fmt.Fprint(w, `{"id":"i1"}`)
	})
	nm, d := "N", "<p>d</p>"
	p := WorkItemPatch{Name: &nm, DescHTML: &d, LabelIDs: &[]string{"l1"}}
	if _, err := c.UpdateWorkItem(context.Background(), "p", "i1", p); err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got["name"] != "N" || got["description_html"] != "<p>d</p>" || got["labels"].([]any)[0] != "l1" {
		t.Fatalf("%v", got)
	}
	if _, err := c.UpdateWorkItem(context.Background(), "p", "i1", WorkItemPatch{}); err != nil || len(got) != 0 {
		t.Fatalf("%v %v", got, err)
	}
}

func TestContractPlaneV5(t *testing.T) {
	var last *http.Request
	c, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		last = r
		switch r.URL.Path {
		case pfx + "projects/p/states/":
			fmt.Fprint(w, `[{"id":"s1","name":"A","group":"started","sequence":15000,"default":true},{"id":"s2","name":"B","sequence":"65535.5","is_default":true},{"id":"s3","sequence":"x"},{"id":"s4"}]`)
		case pfx + "work-items/PROJ-7/":
			fmt.Fprint(w, `{"id":"i7","name":"seven","sequence_id":7,"priority":"high"}`)
		case pfx + "work-items/search/":
			if r.URL.Query().Get("limit") == "5" {
				fmt.Fprint(w, `[{"id":"i1","name":"n","sequence_id":1,"project_id":"p","project__identifier":"PROJ"}]`)
				return
			}
			fmt.Fprint(w, `{"issues":[{"id":"i1","name":"n","sequence_id":3,"project_id":"p","project__identifier":"PROJ"}]}`)
		default:
			w.WriteHeader(404)
		}
	})
	ctx := context.Background()
	sts, err := c.States(ctx, "p")
	if err != nil || len(sts) != 4 || sts[0].Sequence != 15000 || !sts[0].Default || sts[1].Sequence != 65535.5 ||
		!sts[1].Default || sts[2].Sequence != 0 || sts[3].Default {
		t.Fatalf("%+v %v", sts, err)
	}
	it, err := c.WorkItemByKey(ctx, "proj", 7)
	if err != nil || it.ID != "i7" || it.ProjectID != "" || it.Sequence != 7 {
		t.Fatalf("%+v %v", it, err)
	}
	if _, err := c.WorkItemByKey(ctx, "NO", 1); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	h, err := c.Search(ctx, "foo bar", "", 0)
	q := last.URL.Query()
	if err != nil || len(h) != 1 || h[0].Sequence != 3 || h[0].ProjectIdentifier != "PROJ" || h[0].ProjectID != "p" ||
		q.Get("search") != "foo bar" || q.Get("limit") != "50" || q.Has("project_id") {
		t.Fatalf("%+v %v %v", h, err, q)
	}
	h, err = c.Search(ctx, "x", "p", 5)
	q = last.URL.Query()
	if err != nil || len(h) != 1 || q.Get("project_id") != "p" || q.Get("limit") != "5" {
		t.Fatalf("%+v %v %v", h, err, q)
	}
	if _, err := c.Search(ctx, "x", "", 5); err != nil {
		t.Fatal(err)
	}
}

func TestContractPlaneV6(t *testing.T) {
	var body map[string]any
	c, _ := newTest(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			body = map[string]any{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			fmt.Fprint(w, `{"id":"n","name":"x","parent":"par"}`)
		default:
			fmt.Fprint(w, `[{"id":"a","parent":"p1"},{"id":"b","parent":{"id":"p2"}},{"id":"c","parent_id":"p3"},{"id":"d","parent":null},{"id":"e"}]`)
		}
	})
	ctx := context.Background()
	items, err := c.WorkItems(ctx, "p")
	if err != nil || len(items) != 5 {
		t.Fatal(items, err)
	}
	for i, want := range []string{"p1", "p2", "p3", "", ""} {
		if items[i].ParentID != want {
			t.Fatalf("%d: %q", i, items[i].ParentID)
		}
	}
	it, err := c.CreateWorkItem(ctx, "p", NewWorkItem{Name: "x", ParentID: "par"})
	if err != nil || body["parent"] != "par" || it.ParentID != "par" {
		t.Fatalf("%+v %v %v", it, body, err)
	}
	if _, err := c.CreateWorkItem(ctx, "p", NewWorkItem{Name: "x"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["parent"]; ok {
		t.Fatal("parent sent")
	}
}
