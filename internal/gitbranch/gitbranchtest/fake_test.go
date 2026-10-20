package gitbranchtest

import (
	"errors"
	"testing"

	"github.com/thegriglat/platui/internal/gitbranch"
)

func TestFake(t *testing.T) {
	f := NewFake()
	if _, err := f.Current(); !errors.Is(err, gitbranch.ErrDetached) {
		t.Fatalf("current: %v", err)
	}
	if res, err := f.Create("a", gitbranch.Options{Checkout: true}); err != nil || !res.Created || !res.Switched {
		t.Fatalf("create: %+v %v", res, err)
	}
	if _, err := f.Create("a", gitbranch.Options{}); !errors.Is(err, gitbranch.ErrBranchExists) {
		t.Fatalf("dup: %v", err)
	}
	if _, err := f.Create("b", gitbranch.Options{}); err != nil {
		t.Fatal(err)
	}
	if err := f.Delete("a", false); !errors.Is(err, gitbranch.ErrCurrentBranch) {
		t.Fatalf("del current: %v", err)
	}
	if _, err := f.AddWorktree("b", "/w/b", gitbranch.Options{}); err != nil {
		t.Fatal(err)
	}
	if res, err := f.AddWorktree("b", "/w/c", gitbranch.Options{}); !errors.Is(err, gitbranch.ErrWorktreeBusy) || res.Path != "/w/b" {
		t.Fatalf("busy: %+v %v", res, err)
	}
	if err := f.Delete("b", true); !errors.Is(err, gitbranch.ErrWorktreeBusy) {
		t.Fatalf("del busy: %v", err)
	}
	if err := f.Push("origin", "a"); err != nil {
		t.Fatal(err)
	}
	st, _ := f.Statuses([]string{"a"}, "origin", "")
	if !st["a"].Remote || !st["a"].Current {
		t.Fatalf("status: %+v", st["a"])
	}
	if _, inst, _ := f.InstallHook(); !inst {
		t.Fatal("first install")
	}
	if _, inst, _ := f.InstallHook(); inst {
		t.Fatal("second install")
	}
	f.Err["Dirty"] = errors.New("boom")
	if _, err := f.Dirty(); err == nil {
		t.Fatal("injected error")
	}
	if len(f.Calls) == 0 || len(f.Pushed) != 1 {
		t.Fatalf("calls: %v %v", f.Calls, f.Pushed)
	}
}
