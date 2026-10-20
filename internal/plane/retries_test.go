package plane

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestContractRetriesDisabled(t *testing.T) {
	for _, tc := range []struct{ max, want int32 }{{-1, 1}, {0, 4}, {2, 3}} {
		var n int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&n, 1)
			w.WriteHeader(503)
		}))
		c := New(Options{BaseURL: srv.URL, Workspace: ws, MaxRetries: int(tc.max), Sleep: func(d time.Duration) {}})
		_, _ = c.Projects(context.Background())
		srv.Close()
		if n != tc.want {
			t.Errorf("MaxRetries %d: requests %d, want %d", tc.max, n, tc.want)
		}
	}
}
