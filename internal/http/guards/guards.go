package guards

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// MaxThrottleEntries is the maximum number of throttle entries before eviction begins.
	// This prevents unbounded memory growth in high-traffic scenarios.
	MaxThrottleEntries = 10000
	// ThrottleCleanupInterval is how often the throttle store is cleaned up.
	// Balance between memory usage and cleanup overhead.
	ThrottleCleanupInterval = 5 * time.Minute
	// EvictionBufferPercentage is the extra percentage of entries to remove during eviction.
	// Removing 10% extra avoids frequent eviction cycles.
	EvictionBufferPercentage = 0.10
)

// GuardFunc is a function that checks if a request should proceed.
type GuardFunc func(ctx Context) (bool, error)

// Context is the interface for request context.
type Context interface {
	Get(key string) any
}

// HasRole is an interface that types can implement for role checking.
type HasRole interface {
	HasRole(role string) bool
}

// RolesGuard creates a guard that checks if the user has one of the required roles.
// The user must be stored in the context with the given key.
// The user value should implement the HasRole interface.
// Usage: Guard(RolesGuard("user", "admin"))
func RolesGuard(contextKey string, requiredRoles ...string) GuardFunc {
	return func(ctx Context) (bool, error) {
		user := ctx.Get(contextKey)
		if user == nil {
			return false, fmt.Errorf("unauthorized: user not found in context")
		}

		if hasRole, ok := user.(HasRole); ok {
			for _, role := range requiredRoles {
				if hasRole.HasRole(role) {
					return true, nil
				}
			}
			return false, fmt.Errorf("forbidden: insufficient permissions")
		}

		return false, fmt.Errorf("forbidden: user does not implement HasRole interface")
	}
}

// ThrottleGuard creates a rate-limiting guard using a simple in-memory counter.
// Usage: Guard(ThrottleGuard("ip", 10, time.Minute))
func ThrottleGuard(identifierKey string, maxRequests int, window time.Duration) GuardFunc {
	// Initialize cleanup goroutine on first use
	startThrottleCleanup()

	return func(ctx Context) (bool, error) {
		identifier := ctx.Get(identifierKey)
		if identifier == nil {
			identifier = "default"
		}
		key := fmt.Sprintf("throttle:%v", identifier)

		throttleMu.Lock()
		defer throttleMu.Unlock()

		now := time.Now()
		if _, ok := throttleStore[key]; !ok {
			throttleStore[key] = &throttleEntry{}
		}

		entry := throttleStore[key]
		entry.mu.Lock()
		defer entry.mu.Unlock()

		entry.counts = filterOldCounts(entry.counts, now.Add(-window))

		if len(entry.counts) >= maxRequests {
			return false, fmt.Errorf("rate limit exceeded: %d requests per %v", maxRequests, window)
		}

		entry.counts = append(entry.counts, now)
		return true, nil
	}
}

var (
	throttleStore = make(map[string]*throttleEntry)
	throttleMu    sync.Mutex
	cleanupStarted atomic.Bool
)

type throttleEntry struct {
	mu     sync.Mutex
	counts []time.Time
}

func filterOldCounts(counts []time.Time, cutoff time.Time) []time.Time {
	i := 0
	for _, t := range counts {
		if t.After(cutoff) {
			counts[i] = t
			i++
		}
	}
	return counts[:i]
}

// startThrottleCleanup runs periodic cleanup of old throttle entries.
// Called automatically on first use of ThrottleGuard.
func startThrottleCleanup() {
	if cleanupStarted.Swap(true) {
		return
	}

	go func() {
		ticker := time.NewTicker(ThrottleCleanupInterval)
		defer ticker.Stop()

		for range ticker.C {
			throttleMu.Lock()
			now := time.Now()
			for key, entry := range throttleStore {
				entry.mu.Lock()
				entry.counts = filterOldCounts(entry.counts, now.Add(-time.Hour))
				if len(entry.counts) == 0 {
					delete(throttleStore, key)
				}
				entry.mu.Unlock()
			}
			// Enforce maximum entries to prevent unbounded memory growth
			if len(throttleStore) > MaxThrottleEntries {
				evictOldestEntries()
			}
			throttleMu.Unlock()
		}
	}()
}

// evictOldestEntries removes oldest entries to maintain MaxThrottleEntries limit.
//
// Eviction Strategy:
// - Uses an eviction buffer (10% of max) to avoid frequent eviction cycles
// - Removes extra entries beyond the limit to reduce cleanup frequency
// - Simple map iteration removal (not true LRU, but sufficient for rate limiting)
//
// Trade-offs:
// - True LRU would require tracking last access time per entry (more memory)
// - Random eviction would be simpler but less predictable
// - Current approach balances memory overhead with eviction frequency
func evictOldestEntries() {
	buffer := int(float64(MaxThrottleEntries) * EvictionBufferPercentage)
	toRemove := (len(throttleStore) - MaxThrottleEntries) + buffer
	count := 0
	for key := range throttleStore {
		if count >= toRemove {
			break
		}
		delete(throttleStore, key)
		count++
	}
}

// AdminGuard is a convenience guard that checks for admin role.
func AdminGuard(contextKey string) GuardFunc {
	return RolesGuard(contextKey, "admin")
}
