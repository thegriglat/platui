package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/thegriglat/platui/internal/app"
	"github.com/thegriglat/platui/internal/cache"
	"github.com/thegriglat/platui/internal/forge"
	"github.com/thegriglat/platui/internal/forge/forgetest"
	"github.com/thegriglat/platui/internal/gitbranch"
	"github.com/thegriglat/platui/internal/text"
)

// gitModel is a reference model with an MR fake wired to app (testutil does not wire it).
func gitModel(t *testing.T) (*model, *testEnv, *forgetest.Fake, *[]string) {
	t.Helper()
	m, e := newTestModelCfg(t, StartAt{Project: "PROJ"}, nil)
	fg := &forgetest.Fake{URL: "https://x/mr/1"}
	m.s.deps.App = app.New(app.Deps{API: e.api, Git: e.git, Forge: fg, Store: cache.NewMem(func() time.Time { return t0 }),
		Config: e.cfg, Now: func() time.Time { return t0 }})
	var opened []string
	m.s.deps.OpenURL = func(u string) error { opened = append(opened, u); return nil }
	view(m) // aligns the cursor
	return m, e, fg, &opened
}

func has(calls []string, prefix string) bool {
	return slices.ContainsFunc(calls, func(c string) bool { return strings.HasPrefix(c, prefix) })
}

func TestContractTUIGit(t *testing.T) {
	t.Run("n", func(t *testing.T) {
		m, e, _, _ := gitModel(t)
		old := runEditor
		defer func() { runEditor = old }()
		runEditor = func(sess *text.EditSession, _ string, fn tea.ExecCallback) tea.Cmd {
			return func() tea.Msg {
				_ = os.WriteFile(sess.Path, []byte("Description **bold**\n"), 0o600)
				return fn(nil)
			}
		}
		gpress(t, m, "n")
		if m.top().Scope() != "dialog" {
			t.Fatal("no dialog")
		}
		gpress(t, m, "Brand new !high @Anna #Bug", "ctrl+e")
		if !strings.Contains(view(m), "description: Description") {
			t.Errorf("description not shown:\n%s", view(m))
		}
		gpress(t, m, "enter")
		if !has(e.api.Calls, "CreateWorkItem:p1") || len(m.stack) != 1 || !strings.Contains(m.s.status.text, "created PROJ-13") {
			t.Fatalf("calls=%v stack=%d status=%q", e.api.Calls, len(m.stack), m.s.status.text)
		}
		if it, ok := m.s.itemByID(m.s.cursor); !ok || it.Name != "Brand new" || it.Priority != "high" || it.DescHTML == "" {
			t.Errorf("cursor item %+v", it)
		}
		if it, _ := m.s.itemByID(m.s.cursor); !slices.Equal(it.AssigneeIDs, []string{"u-an"}) || !slices.Equal(it.LabelIDs, []string{"l1"}) {
			t.Errorf("assignee/label: %+v", it)
		}
		// unknown label: error in the dialog, no call
		before := e.count("CreateWorkItem")
		gpress(t, m, "n", "x #nolabel", "enter")
		if len(m.stack) != 2 || !m.s.status.isErr || !strings.Contains(m.s.status.text, "nolabel") || e.count("CreateWorkItem") != before {
			t.Errorf("#unknown: stack=%d status=%+v", len(m.stack), m.s.status)
		}
		gpress(t, m, "esc")
		// parse error: the dialog stays
		gpress(t, m, "n", "!bad", "enter")
		if len(m.stack) != 2 || !m.s.status.isErr {
			t.Errorf("stack=%d status=%+v", len(m.stack), m.s.status)
		}
		gpress(t, m, "esc")
		if len(m.stack) != 1 {
			t.Error("esc did not close")
		}
	})
	t.Run("b", func(t *testing.T) {
		m, e, _, _ := gitModel(t)
		it, _ := m.target()
		name, _ := m.s.deps.App.BranchName(*m.s.project, it)
		gpress(t, m, "b")
		if !strings.Contains(view(m), name) {
			t.Fatalf("no name %q:\n%s", name, view(m))
		}
		gpress(t, m, "enter")
		if !has(e.git.Calls, "Create"+name) || len(m.stack) != 1 || !strings.Contains(m.s.status.text, name) {
			t.Fatalf("git=%v status=%q", e.git.Calls, m.s.status.text)
		}
		if e.count("UpdateWorkItem") != 0 {
			t.Error("b changed the task in Plane")
		}
	})
	t.Run("b existing", func(t *testing.T) {
		m, e, _, _ := gitModel(t)
		it, _ := m.target()
		name, _ := m.s.deps.App.BranchName(*m.s.project, it)
		e.git.BranchesV[name] = gitbranch.BranchStatus{Name: name, Local: true}
		gpress(t, m, "b", "enter")
		if m.top().Scope() != "dialog" || !strings.Contains(view(m), "switch to it") || has(e.git.Calls, "Switch") {
			t.Fatalf("no question, or Switch before y: %v\n%s", e.git.Calls, view(m))
		}
		gpress(t, m, "enter") // Enter means no
		if has(e.git.Calls, "Switch") || len(m.stack) != 1 || m.s.status.text != "cancelled" {
			t.Fatalf("enter: %v stack=%d status=%q", e.git.Calls, len(m.stack), m.s.status.text)
		}
		gpress(t, m, "b", "enter", "n")
		if has(e.git.Calls, "Switch") || len(m.stack) != 1 {
			t.Fatalf("n: %v stack=%d", e.git.Calls, len(m.stack))
		}
		gpress(t, m, "b", "enter", "esc")
		if has(e.git.Calls, "Switch") || len(m.stack) != 1 {
			t.Fatalf("esc: %v", e.git.Calls)
		}
		gpress(t, m, "b", "enter", "Y")
		if !has(e.git.Calls, "Switch"+name) || len(m.stack) != 1 || e.count("UpdateWorkItem") != 0 {
			t.Errorf("y: %v stack=%d", e.git.Calls, len(m.stack))
		}
	})
	t.Run("b without a repository", func(t *testing.T) {
		m, e, _, _ := gitModel(t)
		m.s.deps.App = app.New(app.Deps{API: e.api, Forge: &forgetest.Fake{}, Config: e.cfg})
		gpress(t, m, "b", "enter")
		if !m.s.status.isErr || !strings.Contains(m.s.status.text, "git") {
			t.Errorf("status=%+v", m.s.status)
		}
	})
	t.Run("S", func(t *testing.T) {
		m, e, _, _ := gitModel(t)
		it, _ := m.target()
		name, _ := m.s.deps.App.BranchName(*m.s.project, it)
		gpress(t, m, "S")
		if !has(e.git.Calls, "Create"+name) || e.count("UpdateWorkItem") != 1 {
			t.Fatalf("git=%v api=%v", e.git.Calls, e.api.Calls)
		}
		if !strings.Contains(m.s.status.text, "branch created") || !strings.Contains(m.s.status.text, "In Progress") {
			t.Errorf("status=%q", m.s.status.text)
		}
	})
	t.Run("W", func(t *testing.T) {
		m, e, fg, _ := gitModel(t)
		e.git.CurrentV = "PROJ-1-task"
		gpress(t, m, "W")
		if len(e.git.Pushed) != 1 || len(fg.Reqs) != 1 || fg.Reqs[0].Head != "PROJ-1-task" {
			t.Fatalf("pushed=%v reqs=%v", e.git.Pushed, fg.Reqs)
		}
		if !strings.Contains(m.s.status.text, "https://x/mr/1") {
			t.Errorf("status=%q", m.s.status.text)
		}
	})
	t.Run("M", func(t *testing.T) {
		m, _, fg, opened := gitModel(t)
		gpress(t, m, "M")
		if !strings.Contains(m.s.status.text, "no MRs found") {
			t.Errorf("status=%q", m.s.status.text)
		}
		fg.MRs = []forge.MR{{IID: 1, Title: "one", URL: "https://x/mr/1", State: "opened"}}
		gpress(t, m, "M")
		if len(*opened) != 1 || (*opened)[0] != "https://x/mr/1" {
			t.Fatalf("opened=%v", *opened)
		}
		fg.MRs = append(fg.MRs, forge.MR{IID: 2, Title: "two", URL: "https://x/mr/2", State: "merged"})
		gpress(t, m, "M")
		if m.top().Scope() != "dialog" || !strings.Contains(view(m), "two") {
			t.Fatalf("no choice:\n%s", view(m))
		}
		gpress(t, m, "down", "enter")
		if len(*opened) != 2 || (*opened)[1] != "https://x/mr/2" || len(m.stack) != 1 {
			t.Errorf("opened=%v stack=%d", *opened, len(m.stack))
		}
		fg.Err = forge.ErrNotFound
		gpress(t, m, "M")
		if !m.s.status.isErr || !strings.Contains(m.s.status.text, "glab") {
			t.Errorf("status=%+v", m.s.status)
		}
	})
	t.Run("X", func(t *testing.T) {
		m, e, _, _ := gitModel(t)
		gpress(t, m, "X")
		if m.s.status.text != "no branches of finished tasks" {
			t.Fatalf("status=%q", m.s.status.text)
		}
		e.git.BranchesV["PROJ-9-task-number-9"] = gitbranch.BranchStatus{Name: "PROJ-9-task-number-9", Local: true} // i9 — Done
		e.git.BranchesV["PROJ-6-task-number-6"] = gitbranch.BranchStatus{Name: "PROJ-6-task-number-6", Local: true} // In Progress
		gpress(t, m, "X")
		v := view(m)
		if m.top().Scope() != "dialog" || !strings.Contains(v, "PROJ-9-task-number-9") || strings.Contains(v, "PROJ-6-") {
			t.Fatalf("list:\n%s", v)
		}
		gpress(t, m, "enter") // nothing marked
		if has(e.git.Calls, "Delete") || len(m.stack) != 2 {
			t.Fatalf("delete without marks: %v stack=%d", e.git.Calls, len(m.stack))
		}
		gpress(t, m, "j", "k", "q") // vim keys do not work
		if len(m.stack) != 2 || m.top().(*staleList).cur != 0 || len(m.top().(*staleList).marked) != 0 {
			t.Fatalf("j/k/q: stack=%d", len(m.stack))
		}
		gpress(t, m, "esc")
		gpress(t, m, "X", " ", "enter")
		if !strings.Contains(view(m), "delete 1 branches?") || has(e.git.Calls, "Delete") {
			t.Fatalf("no confirmation:\n%s", view(m))
		}
		gpress(t, m, "enter")
		if has(e.git.Calls, "Delete") || m.s.status.text != "cancelled" {
			t.Error("Delete after Enter")
		}
		gpress(t, m, "esc")
		// ErrNotMerged leads to a force question
		e.git.Err["Delete"] = gitbranch.ErrNotMerged
		gpress(t, m, "X", " ", "enter", "y")
		if !has(e.git.Calls, "DeletePROJ-9-task-number-9false") || !strings.Contains(view(m), "force delete") {
			t.Fatalf("no force question: %v\n%s", e.git.Calls, view(m))
		}
		delete(e.git.Err, "Delete")
		gpress(t, m, "y")
		if !has(e.git.Calls, "DeletePROJ-9-task-number-9true") || !strings.Contains(m.s.status.text, "branches deleted: 1") {
			t.Errorf("git=%v status=%q", e.git.Calls, m.s.status.text)
		}
		// regular deletion of marked branches
		e.git.BranchesV["PROJ-9-task-number-9"] = gitbranch.BranchStatus{Name: "PROJ-9-task-number-9", Local: true}
		gpress(t, m, "esc", "X", " ", "enter", "y")
		if !strings.Contains(m.s.status.text, "branches deleted: 1") {
			t.Errorf("status=%q", m.s.status.text)
		}
		// the list after deletion has no deleted branches, marks cleared
		if sl, ok := m.top().(*staleList); !ok || len(sl.list) != 0 || len(sl.marked) != 0 {
			t.Errorf("list after deletion: %T", m.top())
		}
		n := len(e.git.Calls)
		gpress(t, m, "enter")
		if len(e.git.Calls) != n {
			t.Error("repeated deletion")
		}
	})
}

// gdrain runs a command, unwrapping Batch and Sequence (in order); a command that does not return within 1 s is a test error (not silently skipped).
func gdrain(t *testing.T, m *model, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	ch := make(chan tea.Msg, 1)
	go func() { ch <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-ch:
	case <-time.After(time.Second):
		t.Fatal("gdrain: command did not return within 1 s")
	}
	if msg == nil {
		return
	}
	if v := reflect.ValueOf(msg); v.Kind() == reflect.Slice && v.Type().Elem() == reflect.TypeOf(cmd) {
		for i := 0; i < v.Len(); i++ { // BatchMsg and sequenceMsg
			gdrain(t, m, v.Index(i).Interface().(tea.Cmd))
		}
		return
	}
	_, next := m.Update(msg)
	gdrain(t, m, next)
}

// gpress is like press but supports tea.Sequence.
func gpress(t *testing.T, m *model, keys ...string) {
	t.Helper()
	for _, k := range keys {
		_, cmd := m.Update(keyMsg(k))
		gdrain(t, m, cmd)
	}
}
