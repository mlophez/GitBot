package main

import (
	"crypto/rand"
	"fmt"
	"net/http"

	"gitbot/internal/adapters"
)

// requestID is a middleware that injects a unique request ID into every request
// context and sets X-Request-ID on the response header.
// If the incoming request already carries an X-Request-ID header, that value is reused,
// allowing distributed tracing across services.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = newRequestID()
		}
		ctx := adapters.WithRequestID(r.Context(), id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newRequestID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
