package ligo

// Package ligo provides error types for the Ligo framework.

import "github.com/linkeunid/ligo/internal/core/container"

// ErrAppAlreadyStarted is returned when trying to modify an app after Run() has been called.
// This includes calling Register() or Provide() after the application has started.
type ErrAppAlreadyStarted struct{}

func (e *ErrAppAlreadyStarted) Error() string {
	return "ligo: cannot modify app after Run() has been called"
}

// Re-exported container error types

// ErrMissingDependency is returned when a required dependency cannot be found in the container.
type ErrMissingDependency = container.ErrMissingDependency

// ErrCircularDependency is returned when a circular dependency is detected in the provider graph.
type ErrCircularDependency = container.ErrCircularDependency

// ErrDuplicateProvider is returned when a provider for a type is already registered.
// The first provider is used, and subsequent providers for the same type are ignored.
type ErrDuplicateProvider = container.ErrDuplicateProvider

// DIError is a general error type for dependency injection operations.
type DIError = container.DIError
