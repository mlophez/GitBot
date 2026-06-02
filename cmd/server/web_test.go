package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestServeIndex verifies the embedded operator panel is served at the root with
// an HTML content type and the expected markup, confirming the go:embed wiring.
func TestServeIndex(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", serveIndex)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	if len(indexHTML) == 0 {
		t.Fatal("indexHTML is empty: embed failed")
	}
	if !strings.Contains(string(indexHTML), "kubeops-agent") {
		t.Error("embedded panel does not contain expected marker")
	}
}
