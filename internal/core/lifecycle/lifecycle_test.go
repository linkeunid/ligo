package lifecycle

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	l := New()
	if l == nil {
		t.Fatal("New() returned nil")
	}
	if l.started {
		t.Error("New() lifecycle should not be started")
	}
	if l.server != nil {
		t.Error("New() should not have a server")
	}
}

func TestAddServer(t *testing.T) {
	l := New()
	server := &http.Server{}

	l.AddServer(server)
	if l.server != server {
		t.Error("AddServer() did not set server")
	}
}

func TestAppendStartHook(t *testing.T) {
	l := New()
	called := false

	hook := func(ctx context.Context) error {
		called = true
		return nil
	}

	l.AppendStartHook(hook)
	if len(l.onStart) != 1 {
		t.Errorf("AppendStartHook() added %d hooks, want 1", len(l.onStart))
	}

	ctx := context.Background()
	l.Start(ctx)
	if !called {
		t.Error("StartHook was not called")
	}
}

func TestAppendStopHook(t *testing.T) {
	l := New()
	called := false

	hook := func(ctx context.Context) error {
		called = true
		return nil
	}

	l.AppendStopHook(hook)
	if len(l.onStop) != 1 {
		t.Errorf("AppendStopHook() added %d hooks, want 1", len(l.onStop))
	}

	ctx := context.Background()
	l.Stop(ctx)
	if !called {
		t.Error("StopHook was not called")
	}
}

func TestStart(t *testing.T) {
	t.Run("first start succeeds", func(t *testing.T) {
		l := New()
		ctx := context.Background()

		err := l.Start(ctx)
		if err != nil {
			t.Errorf("Start() returned error: %v", err)
		}
		if !l.IsStarted() {
			t.Error("Start() did not set started flag")
		}
	})

	t.Run("second start panics", func(t *testing.T) {
		l := New()
		ctx := context.Background()

		l.Start(ctx)
		defer func() {
			if r := recover(); r == nil {
				t.Error("Start() should panic on second call")
			}
		}()
		l.Start(ctx)
	})

	t.Run("hooks execute in order", func(t *testing.T) {
		l := New()
		order := []int{}

		l.AppendStartHook(func(ctx context.Context) error {
			order = append(order, 1)
			return nil
		})
		l.AppendStartHook(func(ctx context.Context) error {
			order = append(order, 2)
			return nil
		})
		l.AppendStartHook(func(ctx context.Context) error {
			order = append(order, 3)
			return nil
		})

		l.Start(context.Background())

		if len(order) != 3 {
			t.Errorf("Start() executed %d hooks, want 3", len(order))
		}
		if order[0] != 1 || order[1] != 2 || order[2] != 3 {
			t.Errorf("Start() hooks executed in wrong order: %v", order)
		}
	})

	t.Run("hook error stops execution", func(t *testing.T) {
		l := New()
		called := []string{}

		l.AppendStartHook(func(ctx context.Context) error {
			called = append(called, "first")
			return nil
		})
		l.AppendStartHook(func(ctx context.Context) error {
			called = append(called, "second")
			return errors.New("test error")
		})
		l.AppendStartHook(func(ctx context.Context) error {
			called = append(called, "third")
			return nil
		})

		err := l.Start(context.Background())
		if err == nil {
			t.Error("Start() should return error from hook")
		}
		if len(called) != 2 {
			t.Errorf("Start() executed %d hooks, want 2 (stopped at error)", len(called))
		}
		if called[1] != "second" {
			t.Errorf("Second hook was not called")
		}
	})
}

func TestStop(t *testing.T) {
	t.Run("stop without server", func(t *testing.T) {
		l := New()
		called := false

		l.AppendStopHook(func(ctx context.Context) error {
			called = true
			return nil
		})

		err := l.Stop(context.Background())
		if err != nil {
			t.Errorf("Stop() returned error: %v", err)
		}
		if !called {
			t.Error("Stop() did not execute hooks")
		}
	})

	t.Run("stop hooks execute in reverse order", func(t *testing.T) {
		l := New()
		order := []int{}

		l.AppendStopHook(func(ctx context.Context) error {
			order = append(order, 1)
			return nil
		})
		l.AppendStopHook(func(ctx context.Context) error {
			order = append(order, 2)
			return nil
		})
		l.AppendStopHook(func(ctx context.Context) error {
			order = append(order, 3)
			return nil
		})

		l.Stop(context.Background())

		if len(order) != 3 {
			t.Errorf("Stop() executed %d hooks, want 3", len(order))
		}
		if order[0] != 3 || order[1] != 2 || order[2] != 1 {
			t.Errorf("Stop() hooks not in reverse order: %v", order)
		}
	})

	t.Run("stop with server", func(t *testing.T) {
		l := New()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		l.AddServer(server.Config)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := l.Stop(ctx)
		if err != nil {
			t.Errorf("Stop() with server returned error: %v", err)
		}
	})

	t.Run("hook error stops execution", func(t *testing.T) {
		l := New()
		called := []string{}

		l.AppendStopHook(func(ctx context.Context) error {
			called = append(called, "first")
			return nil
		})
		l.AppendStopHook(func(ctx context.Context) error {
			called = append(called, "second")
			return errors.New("test error")
		})
		l.AppendStopHook(func(ctx context.Context) error {
			called = append(called, "third")
			return nil
		})

		err := l.Stop(context.Background())
		if err == nil {
			t.Error("Stop() should return error from hook")
		}
		if len(called) != 2 {
			t.Errorf("Stop() executed %d hooks, want 2 (stopped at error)", len(called))
		}
	})
}

func TestIsStarted(t *testing.T) {
	l := New()

	if l.IsStarted() {
		t.Error("IsStarted() should be false initially")
	}

	l.AppendStartHook(func(ctx context.Context) error {
		if !l.IsStarted() {
			t.Error("IsStarted() should be true during hook execution")
		}
		return nil
	})

	l.Start(context.Background())

	if !l.IsStarted() {
		t.Error("IsStarted() should be true after Start()")
	}
}

func TestConcurrentStart(t *testing.T) {
	l := New()
	ctx := context.Background()

	var wg sync.WaitGroup
	panicCount := 0
	var mu sync.Mutex

	for range 10 {
		wg.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					panicCount++
					mu.Unlock()
				}
			}()
			l.Start(ctx)
		})
	}

	wg.Wait()

	if panicCount == 0 {
		t.Error("Expected at least one panic from concurrent Start() calls")
	}
}

func TestAddServer_PanicsAfterStart(t *testing.T) {
	l := New()
	if err := l.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on AddServer after Start")
		}
	}()
	l.AddServer(&http.Server{})
}

func TestAppendStartHook_PanicsAfterStart(t *testing.T) {
	l := New()
	if err := l.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on AppendStartHook after Start")
		}
	}()
	l.AppendStartHook(func(context.Context) error { return nil })
}

func TestAppendStopHook_PanicsAfterStart(t *testing.T) {
	l := New()
	if err := l.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on AppendStopHook after Start")
		}
	}()
	l.AppendStopHook(func(context.Context) error { return nil })
}

func TestConcurrentAppend_NoRace(t *testing.T) {
	l := New()
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			l.AppendStartHook(func(context.Context) error { return nil })
			if i%2 == 0 {
				l.AppendStopHook(func(context.Context) error { return nil })
			}
		})
	}
	wg.Wait()
	if len(l.onStart) != 50 {
		t.Errorf("expected 50 start hooks, got %d", len(l.onStart))
	}
	if len(l.onStop) != 25 {
		t.Errorf("expected 25 stop hooks, got %d", len(l.onStop))
	}
}

func TestStop_Idempotent(t *testing.T) {
	l := New()
	count := 0
	l.AppendStopHook(func(context.Context) error { count++; return nil })

	if err := l.Stop(context.Background()); err != nil {
		t.Fatalf("first Stop: %v", err)
	}
	if err := l.Stop(context.Background()); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
	if count != 1 {
		t.Errorf("expected hook called once, got %d", count)
	}
}

func TestStop_Concurrent(t *testing.T) {
	l := New()
	var calls atomic.Int32
	l.AppendStopHook(func(context.Context) error { calls.Add(1); return nil })

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			_ = l.Stop(context.Background())
		})
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Errorf("expected hook called exactly once, got %d", calls.Load())
	}
}

func TestStart_RollsBackOnHookFailure(t *testing.T) {
	l := New()
	stopOrder := []int{}

	l.AppendStartHook(func(context.Context) error { return nil })
	l.AppendStopHook(func(context.Context) error { stopOrder = append(stopOrder, 0); return nil })
	l.AppendStartHook(func(context.Context) error { return nil })
	l.AppendStopHook(func(context.Context) error { stopOrder = append(stopOrder, 1); return nil })
	l.AppendStartHook(func(context.Context) error { return errors.New("boom") })
	l.AppendStopHook(func(context.Context) error { stopOrder = append(stopOrder, 2); return nil })

	err := l.Start(context.Background())
	if err == nil {
		t.Fatal("expected Start error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected error to contain 'boom', got %v", err)
	}
	if len(stopOrder) != 2 || stopOrder[0] != 1 || stopOrder[1] != 0 {
		t.Errorf("expected reverse rollback [1, 0], got %v", stopOrder)
	}
}

func TestStart_RollbackJoinsErrors(t *testing.T) {
	l := New()
	l.AppendStartHook(func(context.Context) error { return nil })
	l.AppendStopHook(func(context.Context) error { return errors.New("rollback-fail") })
	l.AppendStartHook(func(context.Context) error { return errors.New("start-fail") })

	err := l.Start(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "start-fail") {
		t.Errorf("missing start-fail: %v", err)
	}
	if !strings.Contains(err.Error(), "rollback-fail") {
		t.Errorf("missing rollback-fail: %v", err)
	}
}

func TestLifecycleIntegration(t *testing.T) {
	t.Run("full lifecycle", func(t *testing.T) {
		l := New()
		startCalls := []string{}
		stopCalls := []string{}

		l.AppendStartHook(func(ctx context.Context) error {
			startCalls = append(startCalls, "init1")
			return nil
		})
		l.AppendStartHook(func(ctx context.Context) error {
			startCalls = append(startCalls, "init2")
			return nil
		})
		l.AppendStopHook(func(ctx context.Context) error {
			stopCalls = append(stopCalls, "cleanup1")
			return nil
		})
		l.AppendStopHook(func(ctx context.Context) error {
			stopCalls = append(stopCalls, "cleanup2")
			return nil
		})

		ctx := context.Background()

		if err := l.Start(ctx); err != nil {
			t.Fatalf("Start() failed: %v", err)
		}

		if len(startCalls) != 2 {
			t.Errorf("Expected 2 start calls, got %d", len(startCalls))
		}

		if err := l.Stop(ctx); err != nil {
			t.Fatalf("Stop() failed: %v", err)
		}

		if len(stopCalls) != 2 {
			t.Errorf("Expected 2 stop calls, got %d", len(stopCalls))
		}

		if stopCalls[0] != "cleanup2" || stopCalls[1] != "cleanup1" {
			t.Errorf("Stop hooks not in reverse order: %v", stopCalls)
		}
	})
}
