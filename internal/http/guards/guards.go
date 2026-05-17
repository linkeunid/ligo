package guards

import (
	"fmt"
	"sync"
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

// Throttler is an in-memory rate limiter. Each instance owns its own counter
// store and cleanup goroutine, so two app instances in the same process no
// longer share state (the prior package-level globals had this defect) and
// tests can scope a fresh Throttler per test. Call Close to stop the cleanup
// goroutine on shutdown.
type Throttler struct {
	maxRequests int
	window      time.Duration

	mu    sync.Mutex
	store map[string][]time.Time

	stop chan struct{}
	once sync.Once
}

// NewThrottler returns a Throttler with its own cleanup goroutine. Caller
// must invoke Close on application shutdown to stop the goroutine.
func NewThrottler(maxRequests int, window time.Duration) *Throttler {
	t := &Throttler{
		maxRequests: maxRequests,
		window:      window,
		store:       make(map[string][]time.Time),
		stop:        make(chan struct{}),
	}
	go t.cleanupLoop()
	return t
}

// Close stops the cleanup goroutine. Idempotent.
func (t *Throttler) Close() {
	t.once.Do(func() { close(t.stop) })
}

// Guard returns a GuardFunc using identifierKey to look up the client
// identifier in the request context. Identifiers without a stored value fall
// back to the literal string "default".
func (t *Throttler) Guard(identifierKey string) GuardFunc {
	return func(ctx Context) (bool, error) {
		identifier := ctx.Get(identifierKey)
		if identifier == nil {
			identifier = "default"
		}
		key := fmt.Sprintf("throttle:%v", identifier)
		now := time.Now()

		t.mu.Lock()
		defer t.mu.Unlock()

		counts := filterOldCounts(t.store[key], now.Add(-t.window))
		if len(counts) >= t.maxRequests {
			t.store[key] = counts
			return false, fmt.Errorf("rate limit exceeded: %d requests per %v", t.maxRequests, t.window)
		}
		t.store[key] = append(counts, now)
		return true, nil
	}
}

func (t *Throttler) cleanupLoop() {
	ticker := time.NewTicker(ThrottleCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-t.stop:
			return
		case <-ticker.C:
			t.cleanup()
		}
	}
}

func (t *Throttler) cleanup() {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	for key, counts := range t.store {
		counts = filterOldCounts(counts, now.Add(-time.Hour))
		if len(counts) == 0 {
			delete(t.store, key)
		} else {
			t.store[key] = counts
		}
	}
	if len(t.store) > MaxThrottleEntries {
		t.evictArbitraryEntries()
	}
}

// evictArbitraryEntries removes entries beyond MaxThrottleEntries to bound
// memory. Go map iteration order is randomized, so the entries dropped are
// arbitrary — not strictly the oldest. True LRU would require per-entry
// "last seen" tracking; for rate-limit memory bounds the current approach
// is sufficient. Callers must hold t.mu.
func (t *Throttler) evictArbitraryEntries() {
	buffer := int(float64(MaxThrottleEntries) * EvictionBufferPercentage)
	toRemove := (len(t.store) - MaxThrottleEntries) + buffer
	count := 0
	for key := range t.store {
		if count >= toRemove {
			break
		}
		delete(t.store, key)
		count++
	}
}

func filterOldCounts(counts []time.Time, cutoff time.Time) []time.Time {
	i := 0
	for _, ts := range counts {
		if ts.After(cutoff) {
			counts[i] = ts
			i++
		}
	}
	return counts[:i]
}

// defaultThrottler is the process-wide Throttler that backs the legacy
// ThrottleGuard function. Lazily initialized on first use and never closed
// for backwards compatibility — callers that want explicit lifecycle should
// use NewThrottler directly.
var defaultThrottler = sync.OnceValue(func() *Throttler {
	// maxRequests/window are set per-Guard, not on the Throttler — the legacy
	// API encodes them per call. Use sentinel values; the Throttler ignores
	// them and the wrapper below provides them per call.
	return NewThrottler(0, 0)
})

// ThrottleGuard creates a rate-limiting guard using a process-wide
// in-memory counter store. New code should prefer NewThrottler so each app
// has its own state and can clean up on shutdown.
//
// Usage: Guard(ThrottleGuard("ip", 10, time.Minute))
func ThrottleGuard(identifierKey string, maxRequests int, window time.Duration) GuardFunc {
	t := defaultThrottler()
	return func(ctx Context) (bool, error) {
		identifier := ctx.Get(identifierKey)
		if identifier == nil {
			identifier = "default"
		}
		key := fmt.Sprintf("throttle:%v", identifier)
		now := time.Now()

		t.mu.Lock()
		defer t.mu.Unlock()

		counts := filterOldCounts(t.store[key], now.Add(-window))
		if len(counts) >= maxRequests {
			t.store[key] = counts
			return false, fmt.Errorf("rate limit exceeded: %d requests per %v", maxRequests, window)
		}
		t.store[key] = append(counts, now)
		return true, nil
	}
}

// AdminGuard is a convenience guard that checks for admin role.
func AdminGuard(contextKey string) GuardFunc {
	return RolesGuard(contextKey, "admin")
}
