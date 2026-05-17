package pool

import (
	"sync/atomic"
	"testing"
)

func TestPool_FactoryAllocatesWhenEmpty(t *testing.T) {
	var allocs atomic.Int32
	p := NewPool(func() *int {
		allocs.Add(1)
		v := 0
		return &v
	})

	v := p.Get()
	if v == nil {
		t.Fatal("Get returned nil")
	}
	if allocs.Load() != 1 {
		t.Errorf("factory called %d times, want 1", allocs.Load())
	}
}

func TestPool_PutReturnsValue(t *testing.T) {
	p := NewPool(func() *int { v := 0; return &v })
	v := p.Get()
	*v = 42
	p.Put(v)
	// Cannot assert Get returns same instance — sync.Pool may discard.
}

func TestPool_ResetFnRunsOnPut(t *testing.T) {
	type box struct{ n int }
	resets := 0
	p := NewPoolWithReset(
		func() *box { return &box{n: 0} },
		func(b *box) { resets++; b.n = 0 },
	)

	v := p.Get()
	v.n = 99
	p.Put(v)

	if resets != 1 {
		t.Errorf("reset called %d times, want 1", resets)
	}
	if v.n != 0 {
		t.Errorf("reset did not zero field, n=%d", v.n)
	}
}

func TestPool_NoResetFnIsHarmless(t *testing.T) {
	p := NewPool(func() *int { v := 0; return &v })
	v := p.Get()
	// Should not panic without reset fn.
	p.Put(v)
}
