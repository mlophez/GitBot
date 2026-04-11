// Package server contains HTTP infrastructure: middleware, server setup,
// and any other concern that belongs to the transport layer.
package server

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
)

// contextKey is an unexported type for context keys in this package,
// preventing collisions with keys defined in other packages.
type contextKey string

const requestIDKey contextKey = "request_id"

// RequestID is a middleware that injects a unique request ID into every request
// context and sets X-Request-ID on the response header.
// If the incoming request already carries an X-Request-ID header, that value is reused,
// allowing distributed tracing across services.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = newRequestID()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logger returns a slog.Logger with the request_id from ctx pre-set as a structured field.
// Use this in all use case handlers instead of calling slog directly.
func Logger(ctx context.Context) *slog.Logger {
	return slog.With("request_id", requestIDFromCtx(ctx))
}

func requestIDFromCtx(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

func newRequestID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
