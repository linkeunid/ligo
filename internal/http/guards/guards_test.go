package guards

import (
	"sync"
	"testing"
	"time"
)

type mockCtx struct{ vals map[string]any }

func (m *mockCtx) Get(key string) any { return m.vals[key] }

func TestThrottler_AllowsUnderLimit(t *testing.T) {
	tr := NewThrottler(3, time.Minute)
	defer tr.Close()
	g := tr.Guard("ip")
	ctx := &mockCtx{vals: map[string]any{"ip": "1.2.3.4"}}

	for i := range 3 {
		ok, err := g(ctx)
		if !ok || err != nil {
			t.Errorf("call %d: expected ok, got ok=%v err=%v", i, ok, err)
		}
	}
}

func TestThrottler_BlocksOverLimit(t *testing.T) {
	tr := NewThrottler(2, time.Minute)
	defer tr.Close()
	g := tr.Guard("ip")
	ctx := &mockCtx{vals: map[string]any{"ip": "1.2.3.4"}}

	_, _ = g(ctx)
	_, _ = g(ctx)
	ok, err := g(ctx)
	if ok {
		t.Error("expected limit exceeded")
	}
	if err == nil {
		t.Error("expected error on limit exceeded")
	}
}

func TestThrottler_IsolatedPerInstance(t *testing.T) {
	a := NewThrottler(1, time.Minute)
	defer a.Close()
	b := NewThrottler(1, time.Minute)
	defer b.Close()

	ctxA := &mockCtx{vals: map[string]any{"ip": "x"}}
	ctxB := &mockCtx{vals: map[string]any{"ip": "x"}}

	if ok, _ := a.Guard("ip")(ctxA); !ok {
		t.Error("first call on A should succeed")
	}
	// Second on A should fail, but B should still allow — proves isolation.
	if ok, _ := a.Guard("ip")(ctxA); ok {
		t.Error("second call on A should be limited")
	}
	if ok, _ := b.Guard("ip")(ctxB); !ok {
		t.Error("first call on B should succeed (isolated from A)")
	}
}

func TestThrottler_CloseIdempotent(t *testing.T) {
	tr := NewThrottler(1, time.Minute)
	tr.Close()
	tr.Close() // must not panic
}

func TestThrottler_Concurrent(t *testing.T) {
	tr := NewThrottler(100, time.Minute)
	defer tr.Close()
	g := tr.Guard("ip")

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			ctx := &mockCtx{vals: map[string]any{"ip": i}}
			_, _ = g(ctx)
		})
	}
	wg.Wait()
}
