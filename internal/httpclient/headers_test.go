package httpclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHeaders(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = r.Header }))
	defer srv.Close()
	c, err := New(Options{Timeout: time.Second, Headers: map[string]string{"CF-Access-Client-Id": "id", "CF-Access-Client-Secret": "s"}})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got.Get("CF-Access-Client-Id") != "id" || got.Get("CF-Access-Client-Secret") != "s" {
		t.Fatal(got)
	}
}
