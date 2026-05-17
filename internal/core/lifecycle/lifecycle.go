package lifecycle

import (
	"context"
	"errors"
	"net/http"
	"sync"
)

// AppLifecycle manages application startup, shutdown, and graceful termination.
type AppLifecycle struct {
	mu      sync.Mutex
	started bool
	stopped bool
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
// Panics if called after Start().
func (l *AppLifecycle) AddServer(srv *http.Server) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.started {
		panic("ligo: lifecycle already started")
	}
	l.server = srv
}

// AppendStartHook adds a hook to run on startup.
// Panics if called after Start().
func (l *AppLifecycle) AppendStartHook(hook func(ctx context.Context) error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.started {
		panic("ligo: lifecycle already started")
	}
	l.onStart = append(l.onStart, hook)
}

// AppendStopHook adds a hook to run on shutdown.
// Panics if called after Start().
func (l *AppLifecycle) AppendStopHook(hook func(ctx context.Context) error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.started {
		panic("ligo: lifecycle already started")
	}
	l.onStop = append(l.onStop, hook)
}

// Start runs all startup hooks sequentially. On hook failure, runs the stop
// hooks corresponding to successful start hooks in reverse order and returns
// the joined errors.
func (l *AppLifecycle) Start(ctx context.Context) error {
	l.mu.Lock()
	if l.started {
		l.mu.Unlock()
		panic("ligo: lifecycle already started")
	}
	l.started = true
	startHooks := append([]func(context.Context) error(nil), l.onStart...)
	stopHooks := append([]func(context.Context) error(nil), l.onStop...)
	l.mu.Unlock()

	for i, hook := range startHooks {
		if err := hook(ctx); err != nil {
			errs := []error{err}
			for j := i - 1; j >= 0; j-- {
				if j >= len(stopHooks) {
					continue
				}
				if rbErr := stopHooks[j](ctx); rbErr != nil {
					errs = append(errs, rbErr)
				}
			}
			return errors.Join(errs...)
		}
	}
	return nil
}

// Stop runs all shutdown hooks in reverse order. Idempotent: subsequent calls
// return nil without re-running hooks.
func (l *AppLifecycle) Stop(ctx context.Context) error {
	l.mu.Lock()
	if l.stopped {
		l.mu.Unlock()
		return nil
	}
	l.stopped = true
	server := l.server
	stopHooks := append([]func(context.Context) error(nil), l.onStop...)
	l.mu.Unlock()

	if server != nil {
		if err := server.Shutdown(ctx); err != nil {
			return err
		}
	}

	for i := len(stopHooks) - 1; i >= 0; i-- {
		if err := stopHooks[i](ctx); err != nil {
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
