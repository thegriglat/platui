package app

import (
	"context"
	"errors"
	"io"
	"net/url"
	"testing"
	"time"

	"github.com/thegriglat/platui/internal/plane"
)

func TestContractAppV7OfflineFromCache(t *testing.T) {
	e := newEnv(t)
	flWarm(t, e)
	if _, err := e.app.Projects(ctx()); err != nil {
		t.Fatal(err)
	}
	e.rebuild()
	e.failAll(flNet)
	if off, _ := e.app.Offline(); off {
		t.Fatal("offline before any call")
	}
	if m, err := e.app.Me(ctx()); err != nil || m.ID != "u-me" {
		t.Fatalf("Me: %v %v", m, err)
	}
	if m, err := e.app.Meta(ctx(), "p1"); err != nil || len(m.States) == 0 {
		t.Fatalf("Meta: %v", err)
	}
	if off, at := e.app.Offline(); !off || !at.Equal(t0) {
		t.Fatalf("Offline = %v %v", off, at)
	}
	// authorization is not replaced by the cache
	e.rebuild()
	e.failAll(flAuth)
	if _, err := e.app.Me(ctx()); err == nil {
		t.Fatal("401 replaced by the cache")
	}
	// network is back
	e.failAll(nil)
	if _, err := e.app.Me(ctx()); err != nil {
		t.Fatal(err)
	}
	if off, _ := e.app.Offline(); off {
		t.Fatal("offline after success")
	}
}

func TestContractAppV7ReplaceKeepsTime(t *testing.T) {
	e := newEnv(t)
	flWarm(t, e)
	e.now = e.now.Add(time.Hour)
	st := "s-done"
	if _, err := e.app.Update(ctx(), "p1", "i1", plane.WorkItemPatch{StateID: &st}); err != nil {
		t.Fatal(err)
	}
	if _, at, _ := e.app.CachedItems("p1"); !at.Equal(t0) {
		t.Fatalf("time changed: %v", at)
	}
}

func TestContractAppV7IsNetworkError(t *testing.T) {
	for err, want := range map[error]bool{
		flNet:                    true,
		io.ErrUnexpectedEOF:      true,
		context.DeadlineExceeded: true,
		&url.Error{Op: "Get", URL: "x", Err: io.EOF}: true,
		&plane.APIError{Status: 503}:                 true,
		&plane.APIError{Status: 429}:                 true,
		&plane.APIError{Status: 400}:                 false,
		flAuth:                                       false,
		errors.New("boom"):                           false,
		errors.New("invalid character 'x'"):          false,
		context.Canceled:                             false,
	} {
		if got := IsNetworkError(err); got != want {
			t.Errorf("%v: %v", err, got)
		}
	}
}
