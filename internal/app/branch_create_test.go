package app

import (
	"errors"
	"testing"

	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/plane"
)

func TestBranchOnly(t *testing.T) {
	e := newEnv(t)
	p := plane.Project{ID: "p1", Identifier: "PROJ"}
	it := plane.WorkItem{ID: "i1", ProjectID: "p1", Sequence: 3, Name: "Fix it"}
	res, err := e.app.Branch(p, it, "", nil, false)
	if err != nil || !res.Created || res.Name != "PROJ-3-fix-it" {
		t.Fatalf("%+v %v", res, err)
	}
	if len(e.api.Calls) != 0 {
		t.Errorf("Plane must not be called: %v", e.api.Calls)
	}
	res, err = e.app.Branch(p, it, "", nil, false)
	if !errors.Is(err, gitbranch.ErrBranchExists) || res.Created || res.Name != "PROJ-3-fix-it" {
		t.Fatalf("exists: %+v %v", res, err)
	}
	if _, err = e.app.Branch(p, it, "", nil, true); err != nil || e.git.Calls[len(e.git.Calls)-1] != "SwitchPROJ-3-fix-it" {
		t.Fatalf("switch: %v %v", err, e.git.Calls)
	}
	e.app.d.Git = nil
	if _, err = e.app.Branch(p, it, "", nil, false); !errors.Is(err, ErrNoRepo) {
		t.Errorf("no repo: %v", err)
	}
}
