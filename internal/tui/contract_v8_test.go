package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/thegriglat/platui/internal/config"
	"github.com/thegriglat/platui/internal/text"

	tea "github.com/charmbracelet/bubbletea"
)

// C4 v8: start without network, "offline" marker, recovery, saving $EDITOR text.
func TestContractTUIV8OfflineAndRecovery(t *testing.T) {
	_, e := flEnvNew(t, 0, nil)
	e.failAll(flNet)
	m := flModel(t, e.api, e.store, 0, nil)
	want := "offline, data from " + t0.Format("15:04")
	if len(m.s.items) != 5 || !strings.Contains(m.s.status.text, want) || !strings.Contains(view(m), want) {
		t.Fatalf("tasks %d, status %+v", len(m.s.items), m.s.status)
	}
	// a non-error status does not displace the marker
	m.s.setStatus("cancelled", false)
	if v := view(m); !strings.Contains(v, want) || !strings.Contains(v, "cancelled") {
		t.Fatalf("marker lost:\n%s", v)
	}
	// network is back: R clears offline, the "refreshed" message (reload), then background: "connection restored" once
	e.failAll(nil)
	press(t, m, "R")
	if !strings.Contains(m.s.status.text, "connection restored") || !strings.Contains(m.s.status.text, "updated") {
		t.Fatalf("R: %q", m.s.status.text)
	}
	if m.s.net.offline || m.s.status.isErr || strings.Contains(view(m), "offline") {
		t.Fatalf("offline not cleared: %+v", m.s.status)
	}
	e.setErr("WorkItems", flNet)
	feed(t, m, pollMsg{})
	e.setErr("WorkItems", nil)
	feed(t, m, pollMsg{})
	if m.s.status.text != "connection restored" || m.s.status.isErr {
		t.Fatalf("status %+v", m.s.status)
	}
	feed(t, m, pollMsg{})
	if m.s.status.text != "connection restored" {
		t.Fatal("repeated check")
	}
	m.s.setStatus("x", false)
	feed(t, m, pollMsg{})
	if m.s.status.text != "x" {
		t.Fatalf("\"connection restored\" repeated: %q", m.s.status.text)
	}
}

func TestContractTUIV8EditorDraftRetry(t *testing.T) {
	syncExec(t)
	var path string
	inner := runEditor
	runEditor = func(sess *text.EditSession, editor string, fn tea.ExecCallback) tea.Cmd {
		path = sess.Path
		return inner(sess, editor, fn)
	}
	m, e := flEnvNew(t, 0, func(c *config.Config) { c.Editor = `sed -i '1i precious text'` })
	t.Cleanup(func() {
		if path != "" {
			os.Remove(path)
		}
	})
	e.setErr("CreateComment", flNet)
	m.s.cursor = "i1"
	press(t, m, "c")
	if !m.s.status.isErr || !strings.Contains(m.s.status.text, "text saved: "+path) {
		t.Fatalf("status %+v", m.s.status)
	}
	if b, err := os.ReadFile(path); err != nil || !strings.Contains(string(b), "precious text") {
		t.Fatalf("file: %v", err)
	}
	// retry: offers the saved text, y sends it without the editor
	e.setErr("CreateComment", nil)
	press(t, m, "c")
	if _, ok := m.top().(*confirmScreen); !ok {
		t.Fatalf("no offer: %T", m.top())
	}
	pressD(t, m, "y")
	if m.s.status.isErr || !strings.Contains(m.s.status.text, "comment added") {
		t.Fatalf("status %+v", m.s.status)
	}
	e.api.Mu.Lock()
	cs := e.api.CommentsV["i1"]
	e.api.Mu.Unlock()
	if len(cs) != 1 || strings.Contains(cs[0].HTML, "Comment on") || !strings.Contains(cs[0].HTML, "precious text") {
		t.Fatalf("hints or wrong text went to Plane: %+v", cs)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("file not removed after success")
	}
}

// N: the draft file is removed, the editor opens with the description from Plane.
func TestContractTUIV8EditorDraftDecline(t *testing.T) {
	syncExec(t)
	var paths []string
	inner := runEditor
	runEditor = func(sess *text.EditSession, editor string, fn tea.ExecCallback) tea.Cmd {
		paths = append(paths, sess.Path)
		return inner(sess, editor, fn)
	}
	m, e := flEnvNew(t, 0, func(c *config.Config) { c.Editor = `sed -i '1i keep me'` })
	t.Cleanup(func() {
		for _, p := range paths {
			os.Remove(p)
		}
	})
	e.api.Mu.Lock()
	e.api.ItemsV["p1"][0].DescHTML = "<p>from plane</p>"
	e.api.Mu.Unlock()
	m.s.items[0].DescHTML = "<p>from plane</p>"
	e.setErr("UpdateWorkItem", flNet)
	m.s.cursor = "i1"
	press(t, m, "e")
	e.setErr("UpdateWorkItem", nil)
	press(t, m, "e")
	pressD(t, m, "n")
	if len(paths) != 2 || paths[0] == paths[1] {
		t.Fatalf("expected a new session: %v", paths)
	}
	e.api.Mu.Lock()
	d := e.api.ItemsV["p1"][0].DescHTML
	e.api.Mu.Unlock()
	if !strings.Contains(d, "from plane") || strings.Count(d, "keep me") != 1 {
		t.Fatalf("editor got text other than from Plane: %q %+v %v", d, m.s.status, paths)
	}
	if _, err := os.Stat(paths[0]); !os.IsNotExist(err) {
		t.Fatal("draft file not removed")
	}
}

// Refusing "description changed, overwrite?" keeps the text.
func TestContractTUIV8OverwriteDeclineKeepsText(t *testing.T) {
	syncExec(t)
	var path string
	inner := runEditor
	runEditor = func(sess *text.EditSession, editor string, fn tea.ExecCallback) tea.Cmd {
		path = sess.Path
		e0 := inner(sess, editor, fn)
		return e0
	}
	m, _ := flEnvNew(t, 0, func(c *config.Config) { c.Editor = `sed -i '1i my text'` })
	t.Cleanup(func() {
		if path != "" {
			os.Remove(path)
		}
	})
	m.s.cursor = "i1"
	m.s.items[0].DescHTML = "<p>old</p>"
	m.s.items[0].UpdatedAt = t0.Add(-time.Hour) // "loaded" before the change in Plane
	press(t, m, "e")
	if _, ok := m.top().(*confirmScreen); !ok {
		t.Fatalf("no question: %T %+v", m.top(), m.s.status)
	}
	pressD(t, m, "n")
	if !m.s.status.isErr || !strings.Contains(m.s.status.text, "text saved: "+path) {
		t.Fatalf("status %+v", m.s.status)
	}
	if b, err := os.ReadFile(path); err != nil || !strings.Contains(string(b), "my text") {
		t.Fatalf("file: %v", err)
	}
}

// The first successful poll after an error reports "connection restored", including with changes.
func TestContractTUIV8RecoveryWithChanges(t *testing.T) {
	m, e := flEnvNew(t, 30*time.Second, nil)
	e.setErr("WorkItems", flNet)
	feed(t, m, pollMsg{})
	e.setErr("WorkItems", nil)
	e.api.Mu.Lock()
	e.api.ItemsV["p1"][0].Name = "Renamed"
	e.api.ItemsV["p1"][0].UpdatedAt = t0.Add(time.Hour)
	e.api.Mu.Unlock()
	feed(t, m, pollMsg{})
	if !strings.HasPrefix(m.s.status.text, "connection restored; changes") || m.s.status.isErr {
		t.Fatalf("status %+v", m.s.status)
	}
}
