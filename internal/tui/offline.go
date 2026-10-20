package tui

import (
	"fmt"
	"os"
	"time"

	"github.com/thegriglat/platui/internal/text"

	"github.com/thegriglat/platui/internal/app"
)

// netState is the connection to Plane and the $EDITOR drafts saved on failure (C4 v8).
type netState struct {
	offline bool      // data is shown from the cache, network unavailable
	offAt   time.Time // time of the oldest data shown
	failing bool      // the last background refresh or load failed
	drafts  map[string]draft
}

// offlineText is "offline, data from HH:MM".
func (n netState) offlineText() string {
	return "offline, data from " + n.offAt.Format("15:04")
}

// errText is the load error text for the status line; a network error when a
// cache exists is marked as offline.
func (s *session) errText(err error) string {
	s.net.failing = true
	msg := err.Error()
	if app.IsCertError(err) {
		msg += " (" + app.CertHint + ")"
	}
	if !app.IsNetworkError(err) {
		return msg
	}
	var at time.Time
	if off, t := s.deps.App.Offline(); off {
		at = t
	}
	if s.project != nil {
		if _, t, ok := s.deps.App.CachedItems(s.project.ID); ok && (at.IsZero() || t.Before(at)) {
			at = t
		}
	}
	if at.IsZero() {
		return msg
	}
	s.net.offline, s.net.offAt = true, at
	return s.net.offlineText() + ": " + msg
}

// restored clears offline and error; true if the connection really recovered.
func (s *session) restored() bool {
	was := s.net.offline || s.net.failing
	s.net.offline, s.net.failing = false, false
	return was
}

// draft is an edit saved after an error: the session (with hints) and the task load time.
type draft struct {
	sess   *text.EditSession
	loaded time.Time
}

// editFailMsg means sending the $EDITOR text failed; the file is saved.
type editFailMsg struct {
	kind, itemID string
	sess         *text.EditSession
	loaded       time.Time
	err          error
}

func draftKey(kind, itemID string) string { return kind + ":" + itemID }

func (s *session) onEditFail(m editFailMsg) {
	if s.net.drafts == nil {
		s.net.drafts = map[string]draft{}
	}
	s.net.drafts[draftKey(m.kind, m.itemID)] = draft{m.sess, m.loaded}
	s.setStatus(fmt.Sprintf("not sent, text saved: %s: %v", m.sess.Path, m.err), true)
}

// savedDraft is an edit saved after an error (forgotten when handed out; the file is
// removed by a successful send, or by editing again in the editor on refusal).
func (s *session) savedDraft(kind, itemID string) (draft, bool) {
	k := draftKey(kind, itemID)
	d, ok := s.net.drafts[k]
	if !ok {
		return draft{}, false
	}
	delete(s.net.drafts, k)
	if _, err := os.Stat(d.sess.Path); err != nil {
		return draft{}, false
	}
	return d, true
}
