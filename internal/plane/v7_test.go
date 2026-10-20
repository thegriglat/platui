package plane

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func v7Client(h http.HandlerFunc) (*Client, *[]time.Duration) {
	srv := httptest.NewServer(h)
	var sleeps []time.Duration
	c := New(Options{BaseURL: srv.URL, APIKey: "k-secret", Workspace: "w", MaxRetries: 1,
		Sleep: func(d time.Duration) { sleeps = append(sleeps, d) }})
	return c, &sleeps
}

func TestContractPlaneV7RetryAfterDate(t *testing.T) {
	c, sleeps := v7Client(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", time.Now().Add(20*time.Second).UTC().Format(http.TimeFormat))
		w.WriteHeader(429)
	})
	_, _ = c.Projects(context.Background())
	if len(*sleeps) == 0 || (*sleeps)[0] < 10*time.Second || (*sleeps)[0] > 20*time.Second {
		t.Fatalf("sleeps=%v", *sleeps)
	}
}

func TestContractPlaneV7NoResults(t *testing.T) {
	c, _ := v7Client(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"detail":"x"}`) })
	_, err := c.Projects(context.Background())
	var ae *APIError
	if !errors.As(err, &ae) || ae.Status != 200 || ae.Message != "unexpected response" {
		t.Fatalf("err=%v", err)
	}
}

func TestContractPlaneV7NoID(t *testing.T) {
	c, _ := v7Client(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{}`) })
	if _, err := c.WorkItem(context.Background(), "p", "i"); err == nil {
		t.Fatal("item")
	}
	_, err := c.Me(context.Background())
	if err == nil || !strings.HasPrefix(err.Error(), "plane:") {
		t.Fatalf("me: %v", err)
	}
}

func TestContractPlaneV7KeyRedacted(t *testing.T) {
	c, _ := v7Client(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		fmt.Fprintf(w, `{"detail":"bad %s"}`, r.Header.Get("X-API-Key"))
	})
	_, err := c.Me(context.Background())
	if err == nil || strings.Contains(err.Error(), "k-secret") || !strings.Contains(err.Error(), "***") {
		t.Fatalf("err=%v", err)
	}
}
