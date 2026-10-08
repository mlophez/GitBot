package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRequiredTokenAuth checks that requiredTokenAuth fails closed: an unconfigured
// token rejects every request with 503, a missing or wrong Bearer token gets 401,
// and in every rejection the protected handler (which would call the provider) is
// never reached.
func TestRequiredTokenAuth(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		header     string
		wantStatus int
		wantCalled bool
	}{
		{"token not configured", "", "Bearer anything", http.StatusServiceUnavailable, false},
		{"token not configured, no header", "", "", http.StatusServiceUnavailable, false},
		{"missing header", "secret", "", http.StatusUnauthorized, false},
		{"wrong token", "secret", "Bearer nope", http.StatusUnauthorized, false},
		{"token without Bearer prefix", "secret", "secret", http.StatusUnauthorized, false},
		{"correct token", "secret", "Bearer secret", http.StatusOK, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodPost, "/api/v1/getparams.execute", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			requiredTokenAuth(tt.configured, next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if called != tt.wantCalled {
				t.Errorf("next called = %v, want %v", called, tt.wantCalled)
			}
		})
	}
}
