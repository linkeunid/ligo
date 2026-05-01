package http

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// HasRole is an interface that types can implement for role checking.
type HasRole interface {
	HasRole(role string) bool
}

// RolesGuard creates a guard that checks if the user has one of the required roles.
// The user must be stored in the context with the given key.
// The user value should implement the HasRole interface.
// Usage: Guard(RolesGuard("user", "admin"))
func RolesGuard(contextKey string, requiredRoles ...string) Guard {
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
func ThrottleGuard(identifierKey string, maxRequests int, window time.Duration) Guard {
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
		ticker := time.NewTicker(5 * time.Minute)
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
			throttleMu.Unlock()
		}
	}()
}

// AdminGuard is a convenience guard that checks for admin role.
func AdminGuard(contextKey string) Guard {
	startThrottleCleanup()
	return RolesGuard(contextKey, "admin")
}
