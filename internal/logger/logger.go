package logger

import (
	"context"
	"log/slog"
)

// contextKey is an unexported type for context keys used within this package,
// preventing collisions with keys defined in other packages.
type contextKey string

const requestIDKey contextKey = "request_id"

// WithRequestID returns a copy of ctx carrying the given request ID.
// Called by the requestID middleware in cmd/server to inject the ID before handlers run.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// Logger returns a slog.Logger with the request_id from ctx pre-set as a structured field.
// Use this in all use case handlers instead of calling slog directly.
func Logger(ctx context.Context) *slog.Logger {
	id, _ := ctx.Value(requestIDKey).(string)
	return slog.With("request_id", id)
}
