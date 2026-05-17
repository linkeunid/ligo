package http

import (
	"github.com/linkeunid/ligo/internal/http/pipes"
)

// Re-exported pipe types and functions

// ValidationPipe binds the request body to T, validates it using struct tags,
// and stores the result in ctx under ValidatedBodyKey.
//
// Retrieve the validated struct in the handler with ValidatedBody[T]:
//
//	input := ligo.ValidatedBody[CreateUserInput](ctx)
func ValidationPipe[T any](v *T) Pipe {
	p := pipes.ValidationPipe(v)
	return func(ctx Context) error {
		return p(ctx)
	}
}

// ValidatedBody retrieves the validated body stored by ValidationPipe[T].
// Panics with a clear message if ValidationPipe was not added to the route.
//
//	input := ligo.ValidatedBody[CreateUserInput](ctx)
func ValidatedBody[T any](ctx Context) *T {
	return pipes.ValidatedBody[T](ctx)
}

// Get retrieves a value from context and asserts it to type T.
// Use this to read values stored by pipes (ParseIntPipe, ParseBoolPipe, UUIDPipe, TrimPipe).
// Returns the zero value of T if the key is missing or the type does not match.
//
//	id   := ligo.Get[int](ctx, "id")     // set by ParseIntPipe("id")
//	ok   := ligo.Get[bool](ctx, "active") // set by ParseBoolPipe("active")
//	uuid := ligo.Get[string](ctx, "id")   // set by UUIDPipe("id")
func Get[T any](ctx Context, key string) T {
	v, _ := ctx.Get(key).(T)
	return v
}

// ParseIntPipe reads path parameter param, parses it as int, and stores the
// result in ctx under param's name.
func ParseIntPipe(param string) Pipe {
	p := pipes.ParseIntPipe(param)
	return func(ctx Context) error {
		return p(ctx)
	}
}

// ParseBoolPipe reads path parameter param, parses it as bool, and stores the
// result in ctx under param's name.
// Accepts: 1, t, T, TRUE, true, True / 0, f, F, FALSE, false, False.
func ParseBoolPipe(param string) Pipe {
	p := pipes.ParseBoolPipe(param)
	return func(ctx Context) error {
		return p(ctx)
	}
}

// UUIDPipe validates that path parameter param is a valid UUID and stores it
// in ctx under param's name.
func UUIDPipe(param string) Pipe {
	p := pipes.UUIDPipe(param)
	return func(ctx Context) error {
		return p(ctx)
	}
}

// TrimPipe removes leading and trailing whitespace from path parameter param
// and stores the trimmed value in ctx under param's name.
func TrimPipe(param string) Pipe {
	p := pipes.TrimPipe(param)
	return func(ctx Context) error {
		return p(ctx)
	}
}

// Re-exported pipe constants

const (
	// ValidatedBodyKey is the context key where ValidationPipe stores the bound and
	// validated struct so handlers can retrieve it without re-binding.
	ValidatedBodyKey = pipes.ValidatedBodyKey
)

// Re-exported pipe errors

// ErrBadRequest is the sentinel error wrapped by pipes when client input is invalid.
// Exception handlers can detect it with errors.Is(err, ligo.ErrBadRequest).
var ErrBadRequest = pipes.ErrBadRequest
