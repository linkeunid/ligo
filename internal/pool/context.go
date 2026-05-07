package pool

import "sync"

// Pool is a generic sync.Pool wrapper for type-safe pooling.
type Pool[T any] struct {
	pool  sync.Pool
	reset func(T)
}

// NewPool creates a new generic pool with the given factory function.
// The factory function is called when the pool needs to create a new value.
func NewPool[T any](factory func() T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				return factory()
			},
		},
	}
}

// NewPoolWithReset creates a new generic pool with factory and reset functions.
// The reset function is called before returning a value to the pool.
func NewPoolWithReset[T any](factory func() T, reset func(T)) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any {
				return factory()
			},
		},
		reset: reset,
	}
}

// Get retrieves a value from the pool, creating one if necessary.
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put returns a value to the pool.
// If a reset function was provided, it will be called before putting the value back.
func (p *Pool[T]) Put(v T) {
	if p.reset != nil {
		p.reset(v)
	}
	p.pool.Put(v)
}

// ContextPool is a specialized pool for context-like objects that need reset.
type ContextPool[T any] struct {
	pool *Pool[T]
}

// NewContextPool creates a new context pool with factory and reset functions.
// The reset function should clear any state that might leak between requests.
func NewContextPool[T any](factory func() T, reset func(T)) *ContextPool[T] {
	return &ContextPool[T]{
		pool: NewPoolWithReset(factory, reset),
	}
}

// Get retrieves a value from the context pool.
func (cp *ContextPool[T]) Get() T {
	return cp.pool.Get()
}

// Put returns a value to the context pool after resetting it.
func (cp *ContextPool[T]) Put(v T) {
	cp.pool.Put(v)
}
