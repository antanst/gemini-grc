package hostPool

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"gemini-grc/common/contextlog"
	"gemini-grc/contextutil"
	"gemini-grc/logging"
)

var hostPool = HostPool{hostnames: make(map[string]struct{})}

type HostPool struct {
	hostnames map[string]struct{}
	lock      sync.RWMutex
}

// RemoveHostFromPool removes a host from the pool with context awareness
func RemoveHostFromPool(ctx context.Context, key string) {
	// Create a hostPool-specific context
	hostCtx := contextutil.ContextWithComponent(ctx, "hostPool")

	contextlog.LogDebugWithContext(hostCtx, logging.GetSlogger(), "Removing host %s from pool", key)

	hostPool.lock.Lock()
	delete(hostPool.hostnames, key)
	hostPool.lock.Unlock()

	// Add some jitter
	time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

	contextlog.LogDebugWithContext(hostCtx, logging.GetSlogger(), "Host %s removed from pool", key)
}

// AddHostToHostPool adds a host to the host pool with context awareness.
// Blocks until the host is added or the context is canceled.
func AddHostToHostPool(ctx context.Context, key string) error {
	// Create a hostPool-specific context
	hostCtx := contextutil.ContextWithComponent(ctx, "hostPool")

	contextlog.LogDebugWithContext(hostCtx, logging.GetSlogger(), "Adding host %s to pool", key)

	// Use a ticker to periodically check if we can add the host
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		// Check if context is done before attempting to acquire lock
		select {
		case <-ctx.Done():
			contextlog.LogDebugWithContext(hostCtx, logging.GetSlogger(), "Context canceled while waiting to add host %s", key)
			return ctx.Err()
		default:
			// Continue with attempt to add host
		}

		hostPool.lock.Lock()
		_, exists := hostPool.hostnames[key]
		if !exists {
			hostPool.hostnames[key] = struct{}{}
			hostPool.lock.Unlock()
			contextlog.LogDebugWithContext(hostCtx, logging.GetSlogger(), "Added host %s to pool", key)
			return nil
		}
		hostPool.lock.Unlock()

		contextlog.LogDebugWithContext(hostCtx, logging.GetSlogger(), "Host %s busy, waiting...", key)

		// Wait for next tick or context cancellation
		select {
		case <-ticker.C:
			// Try again on next tick
		case <-ctx.Done():
			contextlog.LogDebugWithContext(hostCtx, logging.GetSlogger(), "Context canceled while waiting for host %s", key)
			return ctx.Err()
		}
	}
}
