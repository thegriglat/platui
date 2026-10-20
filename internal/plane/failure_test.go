package plane

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Аудит PT-45: недоступность API Plane на уровне клиента (httptest). Тесты GAP
// проверяют ожидаемое («как надо») и пропускаются, пока поведение не исправлено.

const flKey = "plane_api_secret"

func flGap(t *testing.T, ok bool, why string) {
	t.Helper()
	if !ok {
		t.Skip("GAP: " + why)
	}
}

func flNoKey(t *testing.T, err error) {
	t.Helper()
	if err != nil && strings.Contains(err.Error(), flKey) {
		t.Fatalf("ключ API попал в текст ошибки: %v", err)
	}
}

// flClient — клиент к srv с таймаутом to и фиксацией пауз повтора (без реального ожидания).
func flClient(base string, to time.Duration) (*Client, *[]time.Duration) {
	var sleeps []time.Duration
	return New(Options{BaseURL: base, APIKey: flKey, Workspace: ws,
		HTTPClient: &http.Client{Timeout: to},
		Sleep:      func(d time.Duration) { sleeps = append(sleeps, d) }}), &sleeps
}

func flServer(t *testing.T, h http.HandlerFunc) (*httptest.Server, *int32) {
	t.Helper()
	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&n, 1)
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, &n
}

// flCut отдаёт начало ответа и рвёт соединение (Hijack).
func flCut(w http.ResponseWriter, declared int, partial string) {
	conn, buf, err := w.(http.Hijacker).Hijack()
	if err != nil {
		return
	}
	fmt.Fprintf(buf, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s", declared, partial)
	buf.Flush()
	conn.Close()
}

func TestFailurePlaneNoNetworkDNS(t *testing.T) {
	c, _ := flClient("http://plane.invalid", 3*time.Second)
	_, err := c.Projects(context.Background())
	if err == nil || !strings.Contains(err.Error(), "request failed") {
		t.Fatalf("err = %v", err)
	}
	if errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrNotFound) {
		t.Fatalf("сетевая ошибка принята за ответ API: %v", err)
	}
	var ae *APIError
	if errors.As(err, &ae) {
		t.Fatalf("сетевая ошибка не должна быть APIError: %v", err)
	}
	flNoKey(t, err)
}

func TestFailurePlaneConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	base := srv.URL
	srv.Close() // порт закрыт
	c, sleeps := flClient(base, 3*time.Second)
	_, err := c.Me(context.Background())
	if err == nil || !strings.Contains(err.Error(), "request failed") {
		t.Fatalf("err = %v", err)
	}
	if len(*sleeps) != 0 {
		t.Fatalf("повтор на сетевой ошибке: %v", *sleeps) // как есть: повторов нет, backoff — уровнем выше (app.NextPoll)
	}
	flNoKey(t, err)
}

func TestFailurePlaneTimeout(t *testing.T) {
	release := make(chan struct{})
	srv, _ := flServer(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})
	t.Cleanup(func() { close(release) })
	c, _ := flClient(srv.URL, 50*time.Millisecond)
	start := time.Now()
	_, err := c.Projects(context.Background())
	if d := time.Since(start); err == nil || d > 3*time.Second {
		t.Fatalf("err=%v за %v", err, d)
	}
	var ne net.Error
	if !errors.As(err, &ne) || !ne.Timeout() {
		t.Fatalf("ошибка не определяется как таймаут: %v", err)
	}
	flNoKey(t, err)
}

func TestFailurePlaneCtxCancelDuringCall(t *testing.T) {
	release := make(chan struct{})
	srv, _ := flServer(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})
	t.Cleanup(func() { close(release) })
	c, _ := flClient(srv.URL, 10*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := c.Projects(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
}

func TestFailurePlaneMidBodyCut(t *testing.T) {
	srv, n := flServer(t, func(w http.ResponseWriter, r *http.Request) {
		flCut(w, 1000, `{"results":[{"id":"p1","identifier":"A"`)
	})
	c, _ := flClient(srv.URL, 3*time.Second)
	ps, err := c.Projects(context.Background())
	if err == nil || ps != nil || !strings.Contains(err.Error(), "reading response") {
		t.Fatalf("ps=%v err=%v", ps, err)
	}
	if *n != 1 {
		t.Fatalf("запросов %d", *n)
	}
	flNoKey(t, err)
}

func TestFailurePlaneMidPagination(t *testing.T) {
	for _, mode := range []string{"cut", "500", "429"} {
		t.Run(mode, func(t *testing.T) {
			srv, _ := flServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("cursor") == "" {
					fmt.Fprint(w, `{"next_cursor":"c2","next_page_results":true,"results":[{"id":"p1","identifier":"A","name":"Alpha"}]}`)
					return
				}
				switch mode {
				case "cut":
					flCut(w, 500, `{"results":[`)
				case "500":
					w.WriteHeader(500)
				case "429":
					w.WriteHeader(429)
				}
			})
			c, _ := flClient(srv.URL, 3*time.Second)
			ps, err := c.Projects(context.Background())
			if err == nil {
				t.Fatalf("частичный список выдан как полный: %v", ps)
			}
			if len(ps) != 0 {
				t.Fatalf("вместе с ошибкой возвращены данные: %v", ps)
			}
			flNoKey(t, err)
		})
	}
}

func TestFailurePlane5xx(t *testing.T) {
	for _, tc := range []struct {
		status  int
		retried bool
	}{{500, false}, {502, false}, {503, true}} {
		t.Run(fmt.Sprint(tc.status), func(t *testing.T) {
			srv, n := flServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(tc.status)
				fmt.Fprint(w, "<html><body>upstream failure page</body></html>")
			})
			c, sleeps := flClient(srv.URL, 3*time.Second)
			_, err := c.WorkItems(context.Background(), "p1")
			var ae *APIError
			if !errors.As(err, &ae) || ae.Status != tc.status {
				t.Fatalf("err = %v", err)
			}
			if strings.Contains(err.Error(), "<html>") {
				t.Fatalf("HTML-страница в тексте ошибки: %v", err)
			}
			want := int32(1)
			if tc.retried {
				want = 4 // 1 + 3 повтора с паузами 1s, 2s, 4s
			}
			if *n != want || (tc.retried && len(*sleeps) != 3) {
				t.Fatalf("запросов %d, пауз %v", *n, *sleeps)
			}
			flNoKey(t, err)
		})
	}
}

func TestFailurePlane429RetryAfter(t *testing.T) {
	srv, n := flServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(429)
		fmt.Fprint(w, `{"detail":"Request was throttled."}`)
	})
	c, sleeps := flClient(srv.URL, 3*time.Second)
	_, err := c.Projects(context.Background())
	if !errors.Is(err, ErrRateLimited) || *n != 4 {
		t.Fatalf("err=%v запросов %d", err, *n)
	}
	for _, d := range *sleeps {
		if d != 2*time.Second {
			t.Fatalf("паузы %v, ждали Retry-After 2s", *sleeps)
		}
	}
	if !strings.Contains(err.Error(), "throttled") {
		t.Fatalf("текст сервера потерян: %v", err)
	}
}

func TestFailurePlane429RecoversAndCaps(t *testing.T) {
	var calls int32
	srv, _ := flServer(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Retry-After", "3600")
			w.WriteHeader(429)
			return
		}
		fmt.Fprint(w, `[{"id":"p1","identifier":"A","name":"Alpha"}]`)
	})
	c, sleeps := flClient(srv.URL, 3*time.Second)
	ps, err := c.Projects(context.Background())
	if err != nil || len(ps) != 1 {
		t.Fatalf("восстановление после 429: %v %v", ps, err)
	}
	if len(*sleeps) != 1 || (*sleeps)[0] != maxWait {
		t.Fatalf("Retry-After не ограничен %v: %v", maxWait, *sleeps)
	}
}

func TestFailurePlane429CancelDuringWait(t *testing.T) {
	srv, _ := flServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(429) })
	block := make(chan struct{})
	t.Cleanup(func() { close(block) })
	c := New(Options{BaseURL: srv.URL, APIKey: flKey, Workspace: ws, Sleep: func(time.Duration) { <-block }})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := c.Projects(ctx)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 3*time.Second {
		t.Fatalf("err=%v за %v", err, time.Since(start))
	}
}

func TestFailurePlane429RetryAfterHTTPDate(t *testing.T) {
	srv, _ := flServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", time.Now().Add(20*time.Second).UTC().Format(http.TimeFormat))
		w.WriteHeader(429)
	})
	c, sleeps := flClient(srv.URL, 3*time.Second)
	_, _ = c.Projects(context.Background())
	flGap(t, len(*sleeps) > 0 && (*sleeps)[0] >= 10*time.Second,
		"Retry-After в формате HTTP-date (RFC 9110) не разбирается, пауза 1s/2s/4s вместо указанной; правка — transport.go retryDelay")
}

func TestFailurePlaneUnauthorized(t *testing.T) {
	for _, status := range []int{401, 403} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			srv, n := flServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				fmt.Fprint(w, `{"detail":"Given API token is not valid"}`)
			})
			c, _ := flClient(srv.URL, 3*time.Second)
			_, err := c.Me(context.Background())
			var ae *APIError
			if !errors.Is(err, ErrUnauthorized) || !errors.As(err, &ae) || ae.Status != status || *n != 1 {
				t.Fatalf("err=%v запросов %d", err, *n)
			}
			if !strings.Contains(err.Error(), "not valid") {
				t.Fatalf("нет понятного текста: %v", err)
			}
			flNoKey(t, err)
		})
	}
}

func TestFailurePlaneNotFound(t *testing.T) {
	srv, n := flServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		fmt.Fprint(w, `{"error":"Work item not found"}`)
	})
	c, _ := flClient(srv.URL, 3*time.Second)
	_, err := c.WorkItem(context.Background(), "p1", "nope")
	if !errors.Is(err, ErrNotFound) || errors.Is(err, ErrUnauthorized) || *n != 1 {
		t.Fatalf("err=%v запросов %d", err, *n)
	}
}

func TestFailurePlaneTLS(t *testing.T) {
	var hits int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { atomic.AddInt32(&hits, 1) }))
	defer srv.Close()
	c := New(Options{BaseURL: srv.URL, APIKey: flKey, Workspace: ws, HTTPClient: &http.Client{Timeout: 3 * time.Second}})
	_, err := c.Projects(context.Background())
	if err == nil || !strings.Contains(err.Error(), "certificate") {
		t.Fatalf("err = %v", err)
	}
	if hits != 0 {
		t.Fatal("запрос (с ключом) ушёл на сервер с недоверенным сертификатом")
	}
	flNoKey(t, err)
}

func TestFailurePlaneGarbageResponse(t *testing.T) {
	bodies := map[string]string{
		"html":      "<!doctype html><html><body>Login to your network</body></html>",
		"truncated": `{"results":[{"id":"p1","identifier":"A"`,
		"empty":     "",
	}
	calls := map[string]func(*Client) error{
		"Projects":  func(c *Client) error { _, err := c.Projects(context.Background()); return err },
		"WorkItems": func(c *Client) error { _, err := c.WorkItems(context.Background(), "p1"); return err },
		"Comments":  func(c *Client) error { _, err := c.Comments(context.Background(), "p1", "i1"); return err },
		"WorkItem":  func(c *Client) error { _, err := c.WorkItem(context.Background(), "p1", "i1"); return err },
		"Create": func(c *Client) error {
			_, err := c.CreateWorkItem(context.Background(), "p1", NewWorkItem{Name: "x"})
			return err
		},
		"CreateComm": func(c *Client) error {
			_, err := c.CreateComment(context.Background(), "p1", "i1", "<p>x</p>")
			return err
		},
	}
	for bn, body := range bodies {
		srv, _ := flServer(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) })
		c, _ := flClient(srv.URL, 3*time.Second)
		for cn, call := range calls {
			t.Run(bn+"/"+cn, func(t *testing.T) {
				err := call(c)
				if bn == "empty" && (cn == "Projects" || cn == "WorkItems" || cn == "Comments") {
					return // пустое тело списка = пустой список (как есть, см. api-failures.md)
				}
				if err == nil {
					t.Fatal("мусор принят как успех")
				}
				if strings.Contains(err.Error(), "Login to your network") {
					t.Fatalf("тело ответа в тексте ошибки: %v", err)
				}
				flNoKey(t, err)
			})
		}
	}
}

func TestFailurePlaneMeGarbageHasContext(t *testing.T) {
	srv, _ := flServer(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "<html>proxy</html>") })
	c, _ := flClient(srv.URL, 3*time.Second)
	_, err := c.Me(context.Background())
	if err == nil {
		t.Fatal("ожидалась ошибка")
	}
	flGap(t, strings.HasPrefix(err.Error(), "plane:"),
		"Me: ошибка разбора без префикса «plane:» и контекста («invalid character '<'…»); правка — endpoints.go Me, convMember")
}

func TestFailurePlaneListWithoutResults(t *testing.T) {
	srv, _ := flServer(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"detail":"maintenance"}`) })
	c, _ := flClient(srv.URL, 3*time.Second)
	items, err := c.WorkItems(context.Background(), "p1")
	flGap(t, err != nil,
		fmt.Sprintf("ответ 200 без поля results принят как пустой список (%d задач) и затрёт кэш; правка — paginate.go splitPage", len(items)))
}

func TestFailurePlaneEmptyObjectAsItem(t *testing.T) {
	srv, _ := flServer(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{}`) })
	c, _ := flClient(srv.URL, 3*time.Second)
	it, err := c.UpdateWorkItem(context.Background(), "p1", "i1", WorkItemPatch{})
	flGap(t, err != nil || it.ID != "",
		"ответ 200 «{}» на правку принят как задача без ID; правка — endpoints.go itemCall")
}

func TestFailurePlaneKeyNotInErrors(t *testing.T) {
	// Ключ уходит только в заголовке; ни одна ошибка клиента не содержит ни ключа, ни URL с query.
	srv, _ := flServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) })
	c, _ := flClient(srv.URL, 3*time.Second)
	_, err := c.Projects(context.Background())
	flNoKey(t, err)
	if err != nil && strings.Contains(err.Error(), srv.URL) {
		t.Fatalf("URL сервера в тексте ошибки: %v", err)
	}
}

func TestFailurePlaneServerEchoesKey(t *testing.T) {
	srv, _ := flServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		fmt.Fprintf(w, `{"detail":"Invalid token %s"}`, r.Header.Get("X-API-Key"))
	})
	c, _ := flClient(srv.URL, 3*time.Second)
	_, err := c.Me(context.Background())
	flGap(t, err != nil && !strings.Contains(err.Error(), flKey),
		"текст ошибки берётся из тела ответа как есть: сервер/прокси, эхом возвращающий ключ, раскроет его в статусной строке и stderr; правка — transport.go errorMessage (вырезать ключ клиента)")
}
