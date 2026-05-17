package di

import (
	"errors"
	"fmt"
	"strings"
)

// errEntryEmpty is wrapped in a DIError when a registered provider entry
// carries neither an eager value nor a factory function.
var errEntryEmpty = errors.New("ligo: provider entry has neither eager value nor factory")

// formatDIError renders the common
//
//	"ligo: <prefix> <type> [(required by <parent>)][: <cause>]"
//
// shape shared by DIError and ErrMissingDependency. cause may be nil.
func formatDIError(prefix, typ, requiredBy string, cause error) string {
	var b strings.Builder
	fmt.Fprintf(&b, "ligo: %s %s", prefix, typ)
	if requiredBy != "" {
		fmt.Fprintf(&b, " (required by %s)", requiredBy)
	}
	if cause != nil {
		fmt.Fprintf(&b, ": %v", cause)
	}
	return b.String()
}

// ErrMissingDependency is returned when a required provider is not found.
type ErrMissingDependency struct {
	Type       string
	RequiredBy string
	Cause      error
}

func (e *ErrMissingDependency) Error() string {
	return formatDIError("missing dependency", e.Type, e.RequiredBy, nil)
}

func (e *ErrMissingDependency) Unwrap() error { return e.Cause }

// ErrCircularDependency is returned when a circular dependency is detected.
type ErrCircularDependency struct {
	Chain []string
}

func (e *ErrCircularDependency) Error() string {
	return fmt.Sprintf("ligo: circular dependency detected: %v", e.Chain)
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
	return formatDIError("cannot resolve", e.Type, e.RequiredBy, e.Cause)
}

func (e *DIError) Unwrap() error { return e.Cause }
