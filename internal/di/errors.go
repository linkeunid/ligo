package di

import (
	"fmt"
)

// ErrMissingDependency is returned when a required provider is not found.
type ErrMissingDependency struct {
	Type       string
	RequiredBy string
	Cause      error
}

func (e *ErrMissingDependency) Error() string {
	return fmt.Sprintf("ligo: missing dependency %s (required by %s)", e.Type, e.RequiredBy)
}

func (e *ErrMissingDependency) Unwrap() error { return e.Cause }

// ErrCircularDependency is returned when a circular dependency is detected.
type ErrCircularDependency struct {
	Chain []string
}

func (e *ErrCircularDependency) Error() string {
	return fmt.Sprintf("ligo: circular dependency detected: %v", e.Chain)
}

// ErrDuplicateProvider is returned when a provider is registered twice for the same type.
type ErrDuplicateProvider struct {
	Type string
}

func (e *ErrDuplicateProvider) Error() string {
	return fmt.Sprintf("ligo: duplicate provider for type %s", e.Type)
}

// ErrAmbiguousDependency is returned when multiple registered types implement the requested interface.
type ErrAmbiguousDependency struct {
	Interface    string
	Implementors []string
}

func (e *ErrAmbiguousDependency) Error() string {
	return fmt.Sprintf("ligo: ambiguous dependency: multiple types implement %s: %v", e.Interface, e.Implementors)
}

// DIError wraps container resolution failures with context.
type DIError struct {
	Type       string
	RequiredBy string
	Cause      error
}

func (e *DIError) Error() string {
	return fmt.Sprintf("ligo: cannot resolve %s for %s: %v", e.Type, e.RequiredBy, e.Cause)
}
