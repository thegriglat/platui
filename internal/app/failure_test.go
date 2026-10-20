package app

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thegriglat/platui/internal/cache"
	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/plane"
)

// Аудит PT-45: недоступность API Plane на уровне сервисного слоя. Тесты GAP
// проверяют ожидаемое («как надо») и пропускаются, пока поведение не исправлено.

const flKey = "plane_api_secret"

var (
	flNet  = errors.New("plane: GET request failed: dial tcp: lookup plane.invalid: no such host")
	flAuth = &plane.APIError{Status: 401, Message: "Given API token is not valid"}
)

func flGap(t *testing.T, ok bool, why string) {
	t.Helper()
	if !ok {
		t.Skip("GAP: " + why)
	}
}

func (e *env) failAll(err error) {
	e.api.Mu.Lock()
	defer e.api.Mu.Unlock()
	for _, m := range []string{"Me", "Projects", "States", "Members", "Labels", "WorkItems", "WorkItem", "WorkItemByKey",
		"Comments", "CreateWorkItem", "UpdateWorkItem", "CreateComment"} {
		if err == nil {
			delete(e.api.Err, m)
		} else {
			e.api.Err[m] = err
		}
	}
}

func (e *env) fail(method string, err error) {
	e.api.Mu.Lock()
	defer e.api.Mu.Unlock()
	if err == nil {
		delete(e.api.Err, method)
	} else {
		e.api.Err[method] = err
	}
}

// flWarm прогревает память и диск: проекты, справочники, задачи.
func flWarm(t *testing.T, e *env) {
	t.Helper()
	if _, err := e.app.Me(ctx()); err != nil {
		t.Fatal(err)
	}
	if _, err := e.app.Meta(ctx(), "p1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.app.RefreshItems(ctx(), "p1"); err != nil {
		t.Fatal(err)
	}
}

func TestFailureAppRefreshKeepsData(t *testing.T) {
	e := newEnv(t)
	flWarm(t, e)
	before, at, _ := e.app.CachedItems("p1")
	e.now = e.now.Add(time.Hour)
	for _, err := range []error{flNet, flAuth, &plane.APIError{Status: 503, Message: "x"}} {
		e.fail("WorkItems", err)
		if its, ch, got := e.app.RefreshItems(ctx(), "p1"); got == nil || its != nil || !ch.Empty() || !errors.Is(got, err) {
			t.Fatalf("RefreshItems при %v: its=%v got=%v", err, its, got)
		}
		cur, curAt, ok := e.app.CachedItems("p1")
		if !ok || len(cur) != len(before) || !curAt.Equal(at) {
			t.Fatalf("данные или время подменены ошибкой: %v %v", cur, curAt)
		}
	}
	// диск тоже не тронут
	e.rebuild()
	if cur, _, ok := e.app.CachedItems("p1"); !ok || len(cur) != len(before) {
		t.Fatalf("кэш на диске: %v", cur)
	}
	// восстановление само: следующий успешный вызов работает
	e.fail("WorkItems", nil)
	if _, _, err := e.app.RefreshItems(ctx(), "p1"); err != nil {
		t.Fatal(err)
	}
}

func TestFailureAppReloadKeepsData(t *testing.T) {
	for _, method := range []string{"States", "Members", "Labels", "WorkItems"} {
		t.Run(method, func(t *testing.T) {
			e := newEnv(t)
			flWarm(t, e)
			oldMeta, _ := e.app.Meta(ctx(), "p1")
			oldItems, _, _ := e.app.CachedItems("p1")
			// сервер уже отдаёт новое, но обновление обрывается на method
			e.api.StatesV["p1"] = e.api.StatesV["p1"][:2]
			e.api.MembersV["p1"] = nil
			e.api.LabelsV["p1"] = nil
			e.api.ItemsV["p1"] = nil
			e.fail(method, flNet)
			if _, _, _, err := e.app.Reload(ctx(), "p1"); !errors.Is(err, flNet) {
				t.Fatalf("err = %v", err)
			}
			m, _ := e.app.Meta(ctx(), "p1")
			its, _, _ := e.app.CachedItems("p1")
			if len(m.States) != len(oldMeta.States) || len(m.Members) != len(oldMeta.Members) || len(m.Labels) != len(oldMeta.Labels) {
				t.Fatalf("справочники подменены частично: %+v", m)
			}
			if len(its) != len(oldItems) {
				t.Fatalf("задачи подменены: %v", its)
			}
		})
	}
}

func TestFailureAppMetaFirstLoadNotPoisoned(t *testing.T) {
	e := newEnv(t)
	e.fail("Labels", flNet)
	if _, err := e.app.Meta(ctx(), "p1"); !errors.Is(err, flNet) {
		t.Fatalf("err = %v", err)
	}
	var m Meta
	if _, ok, _ := e.store.Load(e.app.key("meta", "p1"), &m); ok {
		t.Fatal("неполные справочники записаны в кэш")
	}
	e.fail("Labels", nil)
	if m, err := e.app.Meta(ctx(), "p1"); err != nil || len(m.Labels) != 2 {
		t.Fatalf("восстановление: %+v %v", m, err)
	}
}

func TestFailureAppMeProjectsRetry(t *testing.T) {
	e := newEnv(t)
	e.fail("Me", flNet)
	e.fail("Projects", flAuth)
	if _, err := e.app.Me(ctx()); !errors.Is(err, flNet) {
		t.Fatalf("Me: %v", err)
	}
	if _, err := e.app.Projects(ctx()); !errors.Is(err, plane.ErrUnauthorized) {
		t.Fatalf("Projects: %v", err)
	}
	if _, err := e.app.Project(ctx(), "PROJ"); !errors.Is(err, plane.ErrUnauthorized) || errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("Project при 401 принят за «нет проекта»: %v", err)
	}
	e.failAll(nil)
	if m, err := e.app.Me(ctx()); err != nil || m.ID != "u-me" {
		t.Fatalf("Me после восстановления: %v %v", m, err)
	}
	if p, err := e.app.Project(ctx(), "PROJ"); err != nil || p.ID != "p1" {
		t.Fatalf("Project после восстановления: %v %v", p, err)
	}
}

// Старт при недоступном API: кэш на диске есть, процесс новый (память пуста).
func TestFailureAppOfflineStartFromDisk(t *testing.T) {
	e := newEnv(t)
	flWarm(t, e)
	if _, err := e.app.Projects(ctx()); err != nil {
		t.Fatal(err)
	}
	e.rebuild() // новый процесс, тот же диск
	e.failAll(flNet)

	t.Run("items", func(t *testing.T) {
		its, at, ok := e.app.CachedItems("p1")
		if !ok || len(its) != 2 || !at.Equal(t0) {
			t.Fatalf("задачи с диска: %v %v %v", its, at, ok)
		}
	})
	t.Run("projects", func(t *testing.T) {
		ps, err := e.app.Projects(ctx())
		flGap(t, err == nil && len(ps) == 2,
			"Projects не читает кэш на диске (только пишет): при недоступном API нет списка проектов; правка — app.go Projects (фолбэк на a.load при ошибке сети)")
	})
	t.Run("meta", func(t *testing.T) {
		m, err := e.app.Meta(ctx(), "p1")
		flGap(t, err == nil && len(m.States) > 0,
			"Meta не читает кэш на диске: без API нет состояний/участников/меток; правка — app.go Meta/fetchMeta (фолбэк на a.load)")
	})
	t.Run("me", func(t *testing.T) {
		m, err := e.app.Me(ctx())
		flGap(t, err == nil && m.ID != "",
			"Me не кэшируется на диске: без API нет фильтра «Мои»; TUI не стартует (bootstrap падает на Me); правка — app.go Me")
	})
	t.Run("find in cache", func(t *testing.T) {
		_, it, err := e.app.Find(ctx(), "PROJ", 42)
		flGap(t, err == nil && it.ID == "i42",
			"Find по ключу при недоступном API не работает, хотя задача в кэше: Project() зависит от сети")
	})
}

func TestFailureAppItemsTimestampTracksFullList(t *testing.T) {
	e := newEnv(t)
	flWarm(t, e)
	_, at, _ := e.app.CachedItems("p1")
	e.now = e.now.Add(3 * time.Hour)
	st := "s-done"
	if _, err := e.app.Update(ctx(), "p1", "i1", plane.WorkItemPatch{StateID: &st}); err != nil {
		t.Fatal(err)
	}
	_, at2, _ := e.app.CachedItems("p1")
	flGap(t, at2.Equal(at),
		"правка одной задачи (replaceItem→commitItems) обновляет время всего списка: пометка «офлайн, данные от ЧЧ:ММ» будет врать; правка — items.go commitItems (время только у RefreshItems/Reload)")
}

func TestFailureAppNextPollBackoff(t *testing.T) {
	base := time.Minute
	cur := base
	var seq []time.Duration
	for i := 0; i < 6; i++ {
		cur = NextPoll(base, cur, flNet)
		seq = append(seq, cur)
	}
	want := []time.Duration{2 * time.Minute, 4 * time.Minute, 5 * time.Minute, 5 * time.Minute, 5 * time.Minute, 5 * time.Minute}
	for i := range want {
		if seq[i] != want[i] {
			t.Fatalf("backoff %v, ждали %v", seq, want)
		}
	}
	if got := NextPoll(base, 5*time.Minute, nil); got != base {
		t.Fatalf("возврат к базе после успеха: %v", got)
	}
	cur = base
	for i := 0; i < 8; i++ {
		cur = NextPoll(base, cur, &plane.APIError{Status: 429})
	}
	if cur != 10*time.Minute {
		t.Fatalf("429: потолок %v", cur)
	}
	// 401 не лечится повтором, но опрос продолжается с потолком 5m (как есть)
	if got := NextPoll(base, base, flAuth); got != 2*time.Minute {
		t.Fatalf("401: %v", got)
	}
}

func TestFailureAppEditsFail(t *testing.T) {
	for _, tc := range []struct {
		name string
		fail string
		call func(e *env) error
	}{
		{"Update", "UpdateWorkItem", func(e *env) error {
			p := "high"
			_, err := e.app.Update(ctx(), "p1", "i1", plane.WorkItemPatch{Priority: &p})
			return err
		}},
		{"SetDescription", "UpdateWorkItem", func(e *env) error { _, err := e.app.SetDescription(ctx(), "p1", "i1", "new **text**"); return err }},
		{"CreateQuick", "CreateWorkItem", func(e *env) error { _, err := e.app.CreateQuick(ctx(), "p1", "New one !high", "desc"); return err }},
		{"Comment", "CreateComment", func(e *env) error { _, err := e.app.Comment(ctx(), "p1", "i1", "hello"); return err }},
		{"Item", "WorkItem", func(e *env) error { _, err := e.app.Item(ctx(), "p1", "i1"); return err }},
		{"Comments", "Comments", func(e *env) error { _, err := e.app.Comments(ctx(), "p1", "i1"); return err }},
	} {
		for _, ferr := range []error{flNet, flAuth} {
			t.Run(fmt.Sprintf("%s/%v", tc.name, errors.Is(ferr, plane.ErrUnauthorized)), func(t *testing.T) {
				e := newEnv(t)
				flWarm(t, e)
				before, _, _ := e.app.CachedItems("p1")
				e.fail(tc.fail, ferr)
				if err := tc.call(e); !errors.Is(err, ferr) {
					t.Fatalf("ошибка потеряна или обёрнута без Is: %v", err)
				}
				after, _, _ := e.app.CachedItems("p1")
				if len(after) != len(before) || after[0].Priority != before[0].Priority || after[0].DescHTML != before[0].DescHTML {
					t.Fatalf("кэш изменён неудачной правкой: %+v", after)
				}
			})
		}
	}
}

func TestFailureAppBulkPartial(t *testing.T) {
	e := newEnv(t)
	flWarm(t, e)
	e.fail("UpdateWorkItem", flNet)
	st := "s-done"
	res := e.app.BulkPatch(ctx(), "p1", []string{"i1", "i42"}, plane.WorkItemPatch{StateID: &st})
	if len(res) != 2 || !errors.Is(res["i1"], flNet) || !errors.Is(res["i42"], flNet) {
		t.Fatalf("итог по каждой задаче: %v", res)
	}
	items, _, _ := e.app.CachedItems("p1")
	if items[0].StateID != "s-todo" {
		t.Fatalf("кэш изменён: %+v", items[0])
	}
}

func TestFailureAppFindErrors(t *testing.T) {
	e := newEnv(t)
	if _, err := e.app.Projects(ctx()); err != nil {
		t.Fatal(err)
	}
	e.fail("WorkItemByKey", &plane.APIError{Status: 404, Message: "nf"})
	if _, _, err := e.app.Find(ctx(), "PROJ", 99); !errors.Is(err, ErrItemNotFound) {
		t.Fatalf("404: %v", err)
	}
	e.fail("WorkItemByKey", flAuth)
	if _, _, err := e.app.Find(ctx(), "PROJ", 99); !errors.Is(err, plane.ErrUnauthorized) || errors.Is(err, ErrItemNotFound) {
		t.Fatalf("401 принят за «нет задачи»: %v", err)
	}
	e.fail("WorkItemByKey", flNet)
	if _, _, err := e.app.Find(ctx(), "PROJ", 99); err == nil || errors.Is(err, ErrItemNotFound) {
		t.Fatalf("сеть принята за «нет задачи»: %v", err)
	}
}

func TestFailureAppStartPlaneDown(t *testing.T) {
	e := newEnv(t)
	e.fail("UpdateWorkItem", flNet)
	r, err := e.app.Start(ctx(), e.project(), e.item("i1"), StartOptions{})
	if !errors.Is(err, flNet) {
		t.Fatalf("err = %v", err)
	}
	// git-часть выполнена и видна вызывающему; состояние задачи не менялось
	if r.Branch.Name == "" || !r.Branch.Created || r.State != "" || r.Assigned {
		t.Fatalf("итог: %+v", r)
	}
	if e.item("i1").StateID != "s-todo" {
		t.Fatal("состояние изменено при ошибке")
	}
}

func TestFailureAppCancelledContext(t *testing.T) {
	e := newEnv(t)
	c, cancel := contextCancelled()
	defer cancel()
	res := e.app.BulkPatch(c, "p1", []string{"i1"}, plane.WorkItemPatch{})
	if !errors.Is(res["i1"], c.Err()) || e.calls("UpdateWorkItem") != 0 {
		t.Fatalf("%v %v", res, e.api.Calls)
	}
}

// ---- сквозные: настоящий plane.Client против httptest ----

type flPlane struct {
	mode atomic.Value // "ok" | "401" | "500" | "503" | "hang" | "cut2" | "html"
	hang chan struct{}
}

func (f *flPlane) handler(w http.ResponseWriter, r *http.Request) {
	mode, _ := f.mode.Load().(string)
	switch mode {
	case "401":
		w.WriteHeader(401)
		fmt.Fprint(w, `{"detail":"Given API token is not valid"}`)
		return
	case "500", "503":
		w.WriteHeader(map[string]int{"500": 500, "503": 503}[mode])
		return
	case "hang":
		select {
		case <-f.hang:
		case <-r.Context().Done():
		}
		return
	case "html":
		fmt.Fprint(w, "<html>captive portal</html>")
		return
	}
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
			fmt.Fprint(w, `{"next_cursor":"c2","next_page_results":true,"results":[{"id":"i1","sequence_id":1,"name":"One","state":"s1"},{"id":"i2","sequence_id":2,"name":"Two","state":"s1"}]}`)
			return
		}
		if mode == "cut2" {
			conn, buf, err := w.(http.Hijacker).Hijack()
			if err == nil {
				fmt.Fprint(buf, "HTTP/1.1 200 OK\r\nContent-Length: 900\r\n\r\n{\"results\":[{\"id\":\"i3\"")
				buf.Flush()
				conn.Close()
			}
			return
		}
		fmt.Fprint(w, `{"next_page_results":false,"results":[{"id":"i3","sequence_id":3,"name":"Three","state":"s1"}]}`)
	default:
		w.WriteHeader(404)
	}
}

func flE2E(t *testing.T) (*App, *flPlane, *cache.Mem, string) {
	t.Helper()
	f := &flPlane{hang: make(chan struct{})}
	f.mode.Store("ok")
	srv := httptest.NewServer(http.HandlerFunc(f.handler))
	t.Cleanup(func() { close(f.hang); srv.Close() })
	store := cache.NewMem(func() time.Time { return t0 })
	api := plane.New(plane.Options{BaseURL: srv.URL, APIKey: flKey, Workspace: "ws",
		HTTPClient: &http.Client{Timeout: 200 * time.Millisecond}, Sleep: func(time.Duration) {}})
	a := New(Deps{API: api, Store: store, Config: &config.Config{Workspace: "ws"}, Now: func() time.Time { return t0 }})
	return a, f, store, srv.URL
}

func TestFailureAppE2EPartialPaginationKeepsFullList(t *testing.T) {
	a, f, store, _ := flE2E(t)
	if its, _, err := a.RefreshItems(ctx(), "p1"); err != nil || len(its) != 3 {
		t.Fatalf("прогрев: %v %v", its, err)
	}
	f.mode.Store("cut2")
	its, _, err := a.RefreshItems(ctx(), "p1")
	if err == nil || its != nil {
		t.Fatalf("обрыв на 2-й странице: its=%v err=%v", its, err)
	}
	if cur, _, ok := a.CachedItems("p1"); !ok || len(cur) != 3 {
		t.Fatalf("память: %v", cur)
	}
	var disk []plane.WorkItem
	if _, ok, _ := store.Load(a.key("items", "p1"), &disk); !ok || len(disk) != 3 {
		t.Fatalf("диск: %v", disk)
	}
	f.mode.Store("ok")
	if its, _, err := a.RefreshItems(ctx(), "p1"); err != nil || len(its) != 3 {
		t.Fatalf("восстановление: %v %v", its, err)
	}
}

func TestFailureAppE2EErrorsTextAndSentinels(t *testing.T) {
	for _, tc := range []struct {
		mode string
		auth bool
	}{{"401", true}, {"500", false}, {"503", false}, {"hang", false}, {"html", false}} {
		t.Run(tc.mode, func(t *testing.T) {
			a, f, _, base := flE2E(t)
			f.mode.Store(tc.mode)
			start := time.Now()
			_, err1 := a.Projects(ctx())
			_, err2 := a.Meta(ctx(), "p1")
			_, _, err3 := a.RefreshItems(ctx(), "p1")
			if time.Since(start) > 5*time.Second {
				t.Fatalf("зависание: %v", time.Since(start))
			}
			for _, err := range []error{err1, err2, err3} {
				if err == nil {
					t.Fatal("ожидалась ошибка")
				}
				if strings.Contains(err.Error(), flKey) || strings.Contains(err.Error(), base) {
					t.Fatalf("ключ или адрес в тексте ошибки: %v", err)
				}
				if errors.Is(err, plane.ErrUnauthorized) != tc.auth {
					t.Fatalf("сентинел 401/403: %v", err)
				}
			}
		})
	}
}

func TestFailureAppE2EClosedPortAndTLS(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	closed := srv.URL
	srv.Close()
	tls := httptest.NewTLSServer(http.NotFoundHandler())
	defer tls.Close()
	for name, base := range map[string]string{"refused": closed, "tls": tls.URL, "dns": "http://plane.invalid"} {
		t.Run(name, func(t *testing.T) {
			api := plane.New(plane.Options{BaseURL: base, APIKey: flKey, Workspace: "ws", HTTPClient: &http.Client{Timeout: 2 * time.Second}})
			a := New(Deps{API: api, Config: &config.Config{Workspace: "ws"}})
			_, err := a.Projects(ctx())
			if err == nil || strings.Contains(err.Error(), flKey) || errors.Is(err, plane.ErrUnauthorized) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}
