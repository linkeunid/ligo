package ligo

import "github.com/linkeunid/ligo/internal/core/container"

// ErrAppAlreadyStarted is returned when trying to modify an app after Run().
type ErrAppAlreadyStarted struct{}

func (e *ErrAppAlreadyStarted) Error() string {
	return "ligo: cannot modify app after Run() has been called"
}

// Re-export container error types
type ErrMissingDependency = container.ErrMissingDependency
type ErrCircularDependency = container.ErrCircularDependency
type ErrDuplicateProvider = container.ErrDuplicateProvider
type DIError = container.DIError
