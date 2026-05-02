package resolver

import (
	"fmt"
	"reflect"

	"github.com/linkeunid/ligo/internal/core/container"
)

// Resolver handles dependency resolution with cycle detection.
type Resolver struct {
	container *container.Container
	chain     []reflect.Type
}

// New creates a new Resolver.
func New(c *container.Container) *Resolver {
	return &Resolver{container: c}
}

// Resolve resolves a type by its interface type.
func (r *Resolver) Resolve(iface any) any {
	typ := reflect.TypeOf(iface)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// Find the concrete type from container that implements this interface
	for _, t := range r.container.Types() {
		if t.Implements(typ) || t == typ {
			return r.resolve(t)
		}
	}
	panic(fmt.Errorf("ligo: no provider found for %s", typ.String()))
}

// resolve with cycle detection.
func (r *Resolver) resolve(typ reflect.Type) any {
	// Check for cycle
	for _, t := range r.chain {
		if t == typ {
			chainStrs := make([]string, len(r.chain))
			for i, ct := range r.chain {
				chainStrs[i] = ct.String()
			}
			panic(fmt.Errorf("ligo: circular dependency detected: %v", chainStrs))
		}
	}

	r.chain = append(r.chain, typ)
	defer func() { r.chain = r.chain[:len(r.chain)-1] }()

	val, err := container.ResolveByType(r.container, typ)
	if err != nil {
		panic(err)
	}
	return val
}
