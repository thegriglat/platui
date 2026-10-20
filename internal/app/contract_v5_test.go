package app

import (
	"strings"
	"testing"
)

func TestContractAppV5StartKeepsCommitMsg(t *testing.T) {
	t.Run("differs", func(t *testing.T) {
		e := newEnv(t)
		e.git.CommitMsgs["PROJ-1-fix-login"] = "my own message"
		r, err := e.app.Start(ctx(), e.project(), e.item("i1"), StartOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if e.git.CommitMsgs["PROJ-1-fix-login"] != "my own message" {
			t.Fatalf("overwritten: %q", e.git.CommitMsgs["PROJ-1-fix-login"])
		}
		found := false
		for _, w := range r.Warnings {
			found = found || strings.Contains(w, "уже есть — не перезаписано")
		}
		if !found {
			t.Fatalf("no warning: %v", r.Warnings)
		}
	})
	t.Run("same", func(t *testing.T) {
		e := newEnv(t)
		e.git.CommitMsgs["PROJ-1-fix-login"] = "PROJ-1: Fix login"
		r, err := e.app.Start(ctx(), e.project(), e.item("i1"), StartOptions{})
		if err != nil || len(r.Warnings) != 0 {
			t.Fatalf("%v %v", r.Warnings, err)
		}
	})
}
