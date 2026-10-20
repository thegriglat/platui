package planetest

import (
	"context"
	"errors"
	"testing"

	"github.com/thegriglat/platui/internal/plane"
)

func TestFake(t *testing.T) {
	ctx := context.Background()
	f := NewFake()
	a, err := f.CreateWorkItem(ctx, "p", plane.NewWorkItem{Name: "a"})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := f.CreateWorkItem(ctx, "p", plane.NewWorkItem{Name: "b"})
	if a.Sequence != 1 || b.Sequence != 2 {
		t.Fatalf("seq %d %d", a.Sequence, b.Sequence)
	}
	st, pr := "s1", "high"
	u, err := f.UpdateWorkItem(ctx, "p", a.ID, plane.WorkItemPatch{StateID: &st, Priority: &pr, AssigneeIDs: &[]string{"u"}})
	if err != nil || u.StateID != "s1" || u.Priority != "high" || u.AssigneeIDs[0] != "u" {
		t.Fatalf("%+v %v", u, err)
	}
	got, _ := f.WorkItem(ctx, "p", a.ID)
	if got.StateID != "s1" {
		t.Fatal("not persisted")
	}
	if _, err := f.WorkItem(ctx, "p", "zzz"); !errors.Is(err, plane.ErrNotFound) {
		t.Fatal(err)
	}
	f.Err["WorkItems"] = plane.ErrRateLimited
	if _, err := f.WorkItems(ctx, "p"); !errors.Is(err, plane.ErrRateLimited) {
		t.Fatal(err)
	}
	delete(f.Err, "WorkItems")
	if l, _ := f.WorkItems(ctx, "p"); len(l) != 2 {
		t.Fatal(l)
	}
	found := false
	for _, c := range f.Calls {
		found = found || c == "UpdateWorkItem:p:"+a.ID
	}
	if !found {
		t.Fatalf("calls %v", f.Calls)
	}
	if f.WebURL(plane.Project{Identifier: "PT"}, a) != "https://app.plane.so/ws/browse/PT-1/" {
		t.Fatal(f.WebURL(plane.Project{Identifier: "PT"}, a))
	}
}

func TestFakeV4(t *testing.T) {
	ctx := context.Background()
	f := NewFake()
	f.MeV = plane.Member{ID: "me"}
	it, _ := f.CreateWorkItem(ctx, "p", plane.NewWorkItem{Name: "a", StateID: "s", Priority: "low", LabelIDs: []string{"l"}})
	if it.StateID != "s" || it.Priority != "low" || it.LabelIDs[0] != "l" {
		t.Fatalf("%+v", it)
	}
	nm := "b"
	u, _ := f.UpdateWorkItem(ctx, "p", it.ID, plane.WorkItemPatch{Name: &nm, LabelIDs: &[]string{}})
	if u.Name != "b" || len(u.LabelIDs) != 0 {
		t.Fatalf("%+v", u)
	}
	cm, err := f.CreateComment(ctx, "p", it.ID, "<p>x</p>")
	cs, _ := f.Comments(ctx, "p", it.ID)
	if err != nil || cm.Author != "me" || len(cs) != 1 {
		t.Fatal(cm, err, cs)
	}
}

func TestFakeV5(t *testing.T) {
	ctx := context.Background()
	f := NewFake()
	f.ProjectsV = []plane.Project{{ID: "p", Identifier: "PROJ"}, {ID: "q", Identifier: "Q"}}
	a, _ := f.CreateWorkItem(ctx, "p", plane.NewWorkItem{Name: "Fix Bug", AssigneeIDs: []string{"u"}})
	f.CreateWorkItem(ctx, "q", plane.NewWorkItem{Name: "bug two"})
	f.CreateWorkItem(ctx, "q", plane.NewWorkItem{Name: "bug three"})
	g, err := f.WorkItemByKey(ctx, "proj", 1)
	if err != nil || g.ID != a.ID {
		t.Fatal(g, err)
	}
	g.AssigneeIDs[0] = "x"
	if l, _ := f.WorkItems(ctx, "p"); l[0].AssigneeIDs[0] != "u" {
		t.Fatal("shallow copy")
	}
	if _, err := f.WorkItemByKey(ctx, "ZZ", 1); !errors.Is(err, plane.ErrNotFound) {
		t.Fatal(err)
	}
	if h, _ := f.Search(ctx, "BUG", "", 50); len(h) != 3 {
		t.Fatal(h)
	}
	if h, _ := f.Search(ctx, "bug", "q", 1); len(h) != 1 || h[0].ProjectIdentifier != "Q" {
		t.Fatal(h)
	}
}
