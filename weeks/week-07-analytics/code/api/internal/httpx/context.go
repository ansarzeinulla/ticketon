package httpx

import "context"

type contextKey string

const requestIDKey contextKey = "request_id"

// WithRequestID stores the per-request correlation id on the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext returns the correlation id, or "" when unset.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
