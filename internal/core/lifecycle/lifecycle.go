package lifecycle

import (
	"context"
	"net/http"
	"sync"
)

// AppLifecycle manages application startup, shutdown, and graceful termination.
type AppLifecycle struct {
	mu      sync.Mutex
	started bool
	hooks   [][]LifecycleHook
	onStart []func(ctx context.Context) error
	onStop  []func(ctx context.Context) error
	server  *http.Server
}

// LifecycleHook is a function called during app lifecycle events.
type LifecycleHook func(ctx context.Context) error

// New creates a new AppLifecycle instance.
func New() *AppLifecycle {
	return &AppLifecycle{}
}

// AddServer attaches an HTTP server to manage during shutdown.
func (l *AppLifecycle) AddServer(srv *http.Server) {
	l.server = srv
}

// AppendStartHook adds a hook to run on startup.
func (l *AppLifecycle) AppendStartHook(hook func(ctx context.Context) error) {
	l.onStart = append(l.onStart, hook)
}

// AppendStopHook adds a hook to run on shutdown.
func (l *AppLifecycle) AppendStopHook(hook func(ctx context.Context) error) {
	l.onStop = append(l.onStop, hook)
}

// Start runs all startup hooks sequentially.
func (l *AppLifecycle) Start(ctx context.Context) error {
	l.mu.Lock()
	if l.started {
		l.mu.Unlock()
		panic("ligo: lifecycle already started")
	}
	l.started = true
	l.mu.Unlock()

	for _, hook := range l.onStart {
		if err := hook(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Stop runs all shutdown hooks in reverse order.
func (l *AppLifecycle) Stop(ctx context.Context) error {
	// Stop HTTP server first
	if l.server != nil {
		if err := l.server.Shutdown(ctx); err != nil {
			return err
		}
	}

	// Run stop hooks in reverse
	for i := len(l.onStop) - 1; i >= 0; i-- {
		if err := l.onStop[i](ctx); err != nil {
			return err
		}
	}
	return nil
}

// IsStarted returns whether the lifecycle has been started.
func (l *AppLifecycle) IsStarted() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.started
}
