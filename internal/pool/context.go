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
