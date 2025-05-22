package contextutil

import (
	"context"
	"time"

	"git.antanst.com/antanst/uid"
)

// ContextKey type for context values
type ContextKey string

// Context keys
const (
	CtxKeyURL       ContextKey = "url"        // Full URL being processed
	CtxKeyHost      ContextKey = "host"       // Host of the URL
	CtxKeyRequestID ContextKey = "request_id" // Unique ID for this processing request
	CtxKeyWorkerID  ContextKey = "worker_id"  // Worker ID processing this request
	CtxKeyStartTime ContextKey = "start_time" // When processing started
	CtxKeyComponent ContextKey = "component"  // Component name for logging
)

// NewRequestContext creates a new, cancellable context
// with a timeout and
func NewRequestContext(parentCtx context.Context, url string, host string, workerID int) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(parentCtx, 120*time.Second)
	requestID := uid.UID()
	ctx = context.WithValue(ctx, CtxKeyURL, url)
	ctx = context.WithValue(ctx, CtxKeyHost, host)
	ctx = context.WithValue(ctx, CtxKeyRequestID, requestID)
	ctx = context.WithValue(ctx, CtxKeyWorkerID, workerID)
	ctx = context.WithValue(ctx, CtxKeyStartTime, time.Now())
	return ctx, cancel
}

// Helper functions to get values from context

// GetURLFromContext retrieves the URL from the context
func GetURLFromContext(ctx context.Context) string {
	if url, ok := ctx.Value(CtxKeyURL).(string); ok {
		return url
	}
	return ""
}

// GetHostFromContext retrieves the host from the context
func GetHostFromContext(ctx context.Context) string {
	if host, ok := ctx.Value(CtxKeyHost).(string); ok {
		return host
	}
	return ""
}

// GetRequestIDFromContext retrieves the request ID from the context
func GetRequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(CtxKeyRequestID).(string); ok {
		return id
	}
	return ""
}

// GetWorkerIDFromContext retrieves the worker ID from the context
func GetWorkerIDFromContext(ctx context.Context) int {
	if id, ok := ctx.Value(CtxKeyWorkerID).(int); ok {
		return id
	}
	return -1
}

// GetStartTimeFromContext retrieves the start time from the context
func GetStartTimeFromContext(ctx context.Context) time.Time {
	if startTime, ok := ctx.Value(CtxKeyStartTime).(time.Time); ok {
		return startTime
	}
	return time.Time{}
}

// GetComponentFromContext retrieves the component name from the context
func GetComponentFromContext(ctx context.Context) string {
	if component, ok := ctx.Value(CtxKeyComponent).(string); ok {
		return component
	}
	return ""
}

// ContextWithComponent adds or updates the component name in the context
func ContextWithComponent(ctx context.Context, component string) context.Context {
	return context.WithValue(ctx, CtxKeyComponent, component)
}
