package app

import (
	"context"
	"testing"
	"time"

	"github.com/thegriglat/platui/internal/cache"
	"github.com/thegriglat/platui/internal/plane"
)

func TestRefreshItemsIncremental(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	if _, _, err := e.app.RefreshItems(ctx, "p1"); err != nil { // no snapshot yet: full read
		t.Fatal(err)
	}
	if e.calls("WorkItemsUpdated") != 0 || e.calls("WorkItems") != 1 {
		t.Fatal(e.api.Calls)
	}
	t1 := t0.Add(time.Hour)
	e.api.Mu.Lock()
	e.api.ItemsV["p1"][0].Name, e.api.ItemsV["p1"][0].UpdatedAt = "Fix login now", t1
	e.api.ItemsV["p1"] = append(e.api.ItemsV["p1"], plane.WorkItem{ID: "i43", ProjectID: "p1", Sequence: 43, Name: "New",
		CreatedAt: t1, UpdatedAt: t1})
	e.api.Mu.Unlock()
	its, ch, err := e.app.RefreshItems(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if e.calls("WorkItemsUpdated") != 1 || e.calls("WorkItems") != 2 {
		t.Fatal(e.api.Calls)
	}
	if len(its) != 3 || its[0].ID != "i43" || len(ch.New) != 1 || len(ch.Updated) != 1 {
		t.Fatalf("%+v %+v", its, ch)
	}
	if got := e.app.syncMark("p1"); !got.Equal(t1) {
		t.Fatal(got)
	}

	// a removed task: the merged count exceeds the total → full read
	e.api.Mu.Lock()
	e.api.ItemsV["p1"] = e.api.ItemsV["p1"][1:]
	e.api.Mu.Unlock()
	its, ch, err = e.app.RefreshItems(ctx, "p1")
	if err != nil || len(its) != 2 || len(ch.Removed) != 1 {
		t.Fatalf("%v %+v %+v", err, its, ch)
	}
	if e.calls("WorkItemsUpdated") != 2 || e.calls("WorkItems") != 4 {
		t.Fatal(e.api.Calls)
	}
}

func TestSyncMarkSurvivesEdit(t *testing.T) {
	e := newEnv(t)
	if _, _, err := e.app.RefreshItems(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
	its, _, _ := e.app.CachedItems("p1")
	its[0].UpdatedAt = t0.Add(time.Hour)
	e.app.storeItems("p1", its, t0) // as after editing one task
	if got := e.app.syncMark("p1"); !got.Equal(t0) {
		t.Fatal(got)
	}
	e.rebuild() // from the store
	if got := e.app.syncMark("p1"); !got.Equal(t0) {
		t.Fatal(got)
	}
}

func TestRefreshItemsSkipsUnchangedWrite(t *testing.T) {
	e := newEnv(t)
	e.store = cache.NewMem(func() time.Time { return e.now })
	e.rebuild()
	if _, _, err := e.app.RefreshItems(ctx(), "p1"); err != nil {
		t.Fatal(err)
	}
	e.now = t0.Add(time.Hour)
	if _, ch, err := e.app.RefreshItems(ctx(), "p1"); err != nil || !ch.Empty() {
		t.Fatal(err, ch)
	}
	var its []plane.WorkItem
	if saved, ok, _ := e.store.Load(e.app.itemsKey("p1"), &its); !ok || !saved.Equal(t0) {
		t.Fatal("list rewritten without changes", saved, ok)
	}
	if _, at, _ := e.app.CachedItems("p1"); !at.Equal(e.now) {
		t.Fatal("list time not moved", at)
	}
	e.rebuild() // from the store: the time comes from itemsat
	if _, at, _ := e.app.CachedItems("p1"); !at.Equal(e.now) {
		t.Fatal("itemsat not saved", at)
	}
}
