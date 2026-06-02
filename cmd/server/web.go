package main

import (
	"net/http"
	_ "embed"
)

// indexHTML is the self-contained operator panel (HTML + CSS + JS) served at the
// site root. It is embedded into the binary so the container image is fully
// self-contained and needs no external static-file mount.
//
//go:embed index.html
var indexHTML []byte

// serveIndex handles GET /{$} (the exact site root).
// It returns the embedded operator panel. The panel is served unauthenticated;
// the browser supplies the API Bearer token on each XHR call to the protected
// /api/v1/* endpoints.
func serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}
