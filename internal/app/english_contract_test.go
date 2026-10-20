package app

import (
	"strings"
	"testing"
	"time"
)

// TestContractE2TimerComment pins the timer comment HTML (contract E2 v1).
func TestContractE2TimerComment(t *testing.T) {
	e := newEnv(t)
	_ = e.app.StartTimer("p1", "i42")
	e.now = t0.Add(85*time.Minute + 10*time.Second)
	if _, err := e.app.StopTimer(ctx(), "p1", "i42", ""); err != nil {
		t.Fatal(err)
	}
	cm := e.api.CommentsV["i42"]
	if len(cm) != 1 || !strings.HasPrefix(cm[0].HTML, "<p>Time spent: 1h 26m</p>") {
		t.Fatalf("%+v", cm)
	}
}
