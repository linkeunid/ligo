package ligo

import "reflect"

// Provider represents a dependency provider.
type Provider struct {
	typ       reflect.Type
	fn        any // raw factory function
	eager     any
	transient bool
	exported  bool
}

// Value registers a pre-built instance as a singleton.
func Value[T any](instance T) Provider {
	var zero T
	return Provider{
		typ:   reflect.TypeOf(zero),
		eager: instance,
	}
}

// Factory registers a factory function that produces a singleton.
// The function can have dependencies as parameters; they are auto-injected.
func Factory[T any](fn any) Provider {
	var zero T
	return Provider{
		typ: reflect.TypeOf(zero),
		fn:  fn,
	}
}

// Transient registers a factory function that produces a new instance on each resolve.
func Transient[T any](fn any) Provider {
	p := Factory[T](fn)
	p.transient = true
	return p
}

// Type returns the type this provider produces.
func (p Provider) Type() reflect.Type {
	return p.typ
}

// IsExported returns true if the provider is exported to sibling modules.
func (p Provider) IsExported() bool {
	return p.exported
}

// IsTransient returns true if the provider creates new instances per resolve.
func (p Provider) IsTransient() bool {
	return p.transient
}

// Export marks a provider as exported, making it visible to sibling modules.
func Export(p Provider) Provider {
	p.exported = true
	return p
}

// Fn returns the factory function (for internal use).
func (p Provider) Fn() any {
	return p.fn
}

// Eager returns the eager value if set (for internal use).
func (p Provider) Eager() any {
	return p.eager
}
