package http

import (
	"time"

	"github.com/linkeunid/ligo/internal/http/guards"
)

// HasRole is an interface that types can implement for role checking.
type HasRole = guards.HasRole

// Re-exported guard functions

// RolesGuard creates a guard that checks if the user has one of the required roles.
// The user must be stored in the context with the given key.
// The user value should implement the HasRole interface.
// Usage: Guard(RolesGuard("user", "admin"))
func RolesGuard(contextKey string, requiredRoles ...string) Guard {
	g := guards.RolesGuard(contextKey, requiredRoles...)
	return func(ctx *Context) (bool, error) {
		return g(ctx)
	}
}

// ThrottleGuard creates a rate-limiting guard using a process-wide in-memory
// counter store. New code should prefer NewThrottler so each app has its own
// state and can clean up on shutdown.
//
// Usage: Guard(ThrottleGuard("ip", 10, time.Minute))
func ThrottleGuard(identifierKey string, maxRequests int, window time.Duration) Guard {
	g := guards.ThrottleGuard(identifierKey, maxRequests, window)
	return func(ctx *Context) (bool, error) {
		return g(ctx)
	}
}

// AdminGuard is a convenience guard that checks for admin role.
func AdminGuard(contextKey string) Guard {
	g := guards.AdminGuard(contextKey)
	return func(ctx *Context) (bool, error) {
		return g(ctx)
	}
}

// Throttler is an in-memory rate limiter scoped to a single app instance.
// Use NewThrottler when you need explicit Close on shutdown; ThrottleGuard
// (the package-level function) is the legacy process-wide alternative.
type Throttler = guards.Throttler

// NewThrottler returns an app-scoped Throttler. Caller must invoke
// (*Throttler).Close() on application shutdown.
func NewThrottler(maxRequests int, window time.Duration) *Throttler {
	return guards.NewThrottler(maxRequests, window)
}

// Re-exported guard constants

const (
	// MaxThrottleEntries is the maximum number of throttle entries before eviction begins.
	// This prevents unbounded memory growth in high-traffic scenarios.
	MaxThrottleEntries = guards.MaxThrottleEntries
	// ThrottleCleanupInterval is how often the throttle store is cleaned up.
	// Balance between memory usage and cleanup overhead.
	ThrottleCleanupInterval = guards.ThrottleCleanupInterval
	// EvictionBufferPercentage is the extra percentage of entries to remove during eviction.
	// Removing 10% extra avoids frequent eviction cycles.
	EvictionBufferPercentage = guards.EvictionBufferPercentage
)
