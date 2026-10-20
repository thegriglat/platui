package cli

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/plane"
)

// Audit PT-45: Plane API unavailability in the CLI. GAP tests check the expected
// ("as it should be") behavior and are skipped until it is fixed.

const flKey = "plane_api_secret"

func flGap(t *testing.T, ok bool, why string) {
	t.Helper()
	if !ok {
		t.Fatal(why)
	}
}

type flResult struct {
	code        int
	out, errOut string
}

// flRunReal runs platui with a real plane.Client against base.
func flRunReal(t *testing.T, ctx context.Context, base string, timeout time.Duration, args ...string) flResult {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("api_key: "+flKey+"\nworkspace: ws\ndefault_project: PROJ\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	env := Env{Stdout: &out, Stderr: &errOut, Getenv: func(string) string { return "" },
		Getwd: func() (string, error) { return "/src/repo", nil },
		Build: func(cfg *config.Config, _ string) (*app.App, error) {
			api := plane.New(plane.Options{BaseURL: base, APIKey: cfg.Key(), Workspace: cfg.Workspace,
				HTTPClient: &http.Client{Timeout: timeout}, Sleep: func(time.Duration) {}})
			return app.New(app.Deps{API: api, Config: cfg}), nil
		}}
	code := Run(ctx, append([]string{"--config", path}, args...), env)
	return flResult{code, out.String(), errOut.String()}
}

func flServer(t *testing.T, h http.HandlerFunc) string {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv.URL
}

func flClosedURL() string {
	srv := httptest.NewServer(http.NotFoundHandler())
	u := srv.URL
	srv.Close()
	return u
}

// flOK serves a sample project; page2 handles the second page of issues.
func flOK(page2 func(http.ResponseWriter)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case strings.HasSuffix(p, "/users/me/"):
			fmt.Fprint(w, `{"id":"u1","display_name":"Me"}`)
		case strings.HasSuffix(p, "/workspaces/ws/projects/"):
			fmt.Fprint(w, `[{"id":"p1","identifier":"PROJ","name":"One"}]`)
		case strings.HasSuffix(p, "/states/"):
			fmt.Fprint(w, `[{"id":"s1","name":"Todo","group":"unstarted","sequence":1}]`)
		case strings.HasSuffix(p, "/project-members/"), strings.HasSuffix(p, "/labels/"):
			fmt.Fprint(w, `[]`)
		case strings.HasSuffix(p, "/work-items/"):
			if r.URL.Query().Get("cursor") == "" {
				fmt.Fprint(w, `{"next_cursor":"c2","next_page_results":true,"results":[{"id":"i1","sequence_id":1,"name":"One","state":"s1"}]}`)
				return
			}
			page2(w)
		default:
			w.WriteHeader(404)
		}
	}
}

func TestFailureCLIFakeErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method string
		err    error
		args   []string
		code   int
	}{
		{"ls/Projects", "Projects", &plane.APIError{Status: 503, Message: "Service Unavailable"}, []string{"ls"}, 4},
		{"ls/States", "States", &plane.APIError{Status: 500, Message: "boom"}, []string{"ls"}, 4},
		{"ls/WorkItems", "WorkItems", flNetErr, []string{"ls"}, 4},
		{"show/WorkItemByKey", "WorkItemByKey", flNetErr, []string{"show", "PROJ-99"}, 4},
		{"search/WorkItems", "WorkItems", flNetErr, []string{"search", "fix"}, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFx(t)
			f.api.Err[tc.method] = tc.err
			code := f.run(tc.args...)
			if code != tc.code || f.out.Len() != 0 || !strings.HasPrefix(f.err.String(), "platui: ") || strings.Count(f.err.String(), "\n") != 1 {
				t.Fatalf("code %d, stdout %q, stderr %q", code, f.out.String(), f.err.String())
			}
		})
	}
}

var flNetErr = &url.Error{Op: "Get", URL: "http://plane.invalid", Err: fmt.Errorf("dial tcp: lookup plane.invalid: no such host")}

func TestFailureCLIShowNotFound(t *testing.T) {
	f := newFx(t)
	f.api.Err["WorkItemByKey"] = &plane.APIError{Status: 404, Message: "not found"}
	code := f.run("show", "PROJ-99")
	if code != 1 || !strings.Contains(f.err.String(), "not found") || f.out.Len() != 0 {
		t.Fatalf("code %d, stderr %q", code, f.err.String())
	}
}

func TestFailureCLIRealScenarios(t *testing.T) {
	hang := make(chan struct{})
	t.Cleanup(func() { close(hang) })
	tlsSrv := httptest.NewTLSServer(http.NotFoundHandler())
	defer tlsSrv.Close()
	cases := map[string]struct{ base, want string }{
		"401": {flServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
			fmt.Fprint(w, `{"detail":"Given API token is not valid"}`)
		}), "not valid"},
		"403":     {flServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(403) }), "403"},
		"404":     {flServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }), "404"},
		"500":     {flServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }), "500"},
		"html":    {flServer(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "<html>portal</html>") }), "invalid character"},
		"refused": {flClosedURL(), "request failed"},
		"dns":     {"http://plane.invalid", "request failed"},
		"tls":     {tlsSrv.URL, "certificate"},
		"timeout": {flServer(t, func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-hang:
			case <-r.Context().Done():
			}
		}), "Timeout"},
		"cut-page2": {flServer(t, flOK(func(w http.ResponseWriter) {
			conn, buf, err := w.(http.Hijacker).Hijack()
			if err == nil {
				fmt.Fprint(buf, "HTTP/1.1 200 OK\r\nContent-Length: 900\r\n\r\n{\"results\":[{\"id\":\"i2\"")
				buf.Flush()
				conn.Close()
			}
		})), "reading response"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			to := 3 * time.Second
			if name == "timeout" {
				to = 100 * time.Millisecond
			}
			start := time.Now()
			r := flRunReal(t, context.Background(), tc.base, to, "ls")
			if time.Since(start) > 10*time.Second {
				t.Fatalf("hang: %v", time.Since(start))
			}
			wantCode := map[string]int{"401": 3, "403": 3, "500": 4, "refused": 4, "dns": 4, "timeout": 4, "tls": 4, "cut-page2": 4}[name]
			if wantCode == 0 {
				wantCode = 1
			}
			if r.code != wantCode || r.out != "" || !strings.HasPrefix(r.errOut, "platui: ") || !strings.Contains(r.errOut, tc.want) {
				t.Fatalf("code %d, stdout %q, stderr %q (want %q)", r.code, r.out, r.errOut, tc.want)
			}
			if strings.Contains(r.errOut, flKey) || strings.Contains(r.errOut, "<html>") || strings.Contains(r.errOut, tc.base) {
				t.Fatalf("key, URL or response body in stderr: %q", r.errOut)
			}
		})
	}
}

func TestFailureCLICtrlC(t *testing.T) {
	hang := make(chan struct{})
	t.Cleanup(func() { close(hang) })
	base := flServer(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-hang:
		case <-r.Context().Done():
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	r := flRunReal(t, ctx, base, 30*time.Second, "ls")
	if r.code == 0 || time.Since(start) > 5*time.Second || r.out != "" {
		t.Fatalf("code %d in %v, stdout %q", r.code, time.Since(start), r.out)
	}
}

// Expectation: 401/403 (wrong key — fix the config) is distinguishable by exit code from a network error (retry is possible).
func TestFailureCLIExitCodesDistinguishAuthFromNetwork(t *testing.T) {
	auth := flRunReal(t, context.Background(), flServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) }), 3*time.Second, "ls")
	net := flRunReal(t, context.Background(), flClosedURL(), 3*time.Second, "ls")
	if auth.code == 0 || net.code == 0 {
		t.Fatalf("codes %d %d", auth.code, net.code)
	}
	flGap(t, auth.code != net.code,
		fmt.Sprintf("Run returns 1 for any error (401 and network: %d/%d), so a script cannot tell a wrong key from an unreachable network; fix — cli.go Run (e.g. 3 for auth, 4 for network/timeout via errors.Is plane.ErrUnauthorized / net.Error), contract C10", auth.code, net.code))
}

func TestFailureCLIUnauthorizedHint(t *testing.T) {
	r := flRunReal(t, context.Background(), flServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) }), 3*time.Second, "ls")
	flGap(t, strings.Contains(strings.ToLower(r.errOut), "api_key") || strings.Contains(strings.ToLower(r.errOut), "api key"),
		fmt.Sprintf("on 401/403 there is no hint to check api_key/api_key_env in the config (stderr: %q); fix — cli.go Run", r.errOut))
}

// Is the HTTP timeout configurable? Currently 30s is hardcoded in plane.New and main.go does not pass an HTTPClient.
func TestFailureCLITimeoutsConfigurable(t *testing.T) {
	var find func(rt reflect.Type, depth int) bool
	find = func(rt reflect.Type, depth int) bool {
		for rt.Kind() == reflect.Ptr {
			rt = rt.Elem()
		}
		if rt.Kind() != reflect.Struct || depth > 3 {
			return false
		}
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if strings.Contains(strings.ToLower(f.Tag.Get("yaml")), "timeout") || find(f.Type, depth+1) {
				return true
			}
		}
		return false
	}
	flGap(t, find(reflect.TypeOf(config.Config{}), 0),
		"the config has no Plane request timeout (or retry count): 30s and 3 retries are hardcoded in plane.New, main.go does not pass an HTTPClient; fix — config (http.timeout, http.retries) + main.go build, contracts C1/C0")
}
