package app

import "testing"

// Find with the network unavailable: project and task come from the cache, the app is marked offline.
func TestFindOfflineFromCache(t *testing.T) {
	e := newEnv(t)
	flWarm(t, e)
	if _, err := e.app.Projects(ctx()); err != nil {
		t.Fatal(err)
	}
	e.rebuild() // new process, same disk
	e.failAll(flNet)
	p, it, err := e.app.Find(ctx(), "PROJ", 42)
	if err != nil || p.ID != "p1" || it.ID != "i42" {
		t.Fatalf("Find: %v %v %v", p, it, err)
	}
	if off, at := e.app.Offline(); !off || !at.Equal(t0) {
		t.Fatalf("Offline = %v %v", off, at)
	}
	// task not in the cache: the network error is not hidden
	if _, _, err := e.app.Find(ctx(), "PROJ", 99); err == nil {
		t.Fatal("expected an error for a task outside the cache")
	}
}
