package ligo

import "fmt"

// ErrAppAlreadyStarted is returned when trying to modify an app after Run().
type ErrAppAlreadyStarted struct{}

func (e *ErrAppAlreadyStarted) Error() string {
	return "ligo: cannot modify app after Run() has been called"
}

// ErrMissingDependency is returned when a required provider is not found.
type ErrMissingDependency struct {
	Type       string
	RequiredBy string
}

func (e *ErrMissingDependency) Error() string {
	return fmt.Sprintf("ligo: missing dependency %s (required by %s)", e.Type, e.RequiredBy)
}

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

// DIError wraps container resolution failures with context.
type DIError struct {
	Type       any
	RequiredBy any
	Cause      error
}

func (e *DIError) Error() string {
	return fmt.Sprintf("ligo: cannot resolve %v for %v: %v", e.Type, e.RequiredBy, e.Cause)
}
