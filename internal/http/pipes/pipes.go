package pipes

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/linkeunid/ligo/internal/validation"
)

// ErrBadRequest is the sentinel error wrapped by pipes when client input is invalid.
// Exception handlers can detect it with errors.Is(err, ligo.ErrBadRequest).
var ErrBadRequest = errors.New("bad request")

var validate = validator.New()

// ValidatedBodyKey is the context key where ValidationPipe stores the bound and
// validated struct so handlers can retrieve it without re-binding.
const ValidatedBodyKey = "_ligo_validated"

// PipeFunc is a function that transforms and validates request data.
type PipeFunc func(ctx Context) error

// Context is the interface for request context.
type Context interface {
	Bind(v any) error
	Get(key string) any
	Set(key string, val any)
	Param(key string) string
}

// ValidationPipe binds the request body to T, validates it using struct tags,
// and stores the result in ctx under ValidatedBodyKey.
//
// Retrieve the validated struct in the handler with ValidatedBody[T]:
//
//	input := ligo.ValidatedBody[CreateUserInput](ctx)
func ValidationPipe[T any](_ *T) PipeFunc {
	return func(ctx Context) error {
		var input T
		if err := ctx.Bind(&input); err != nil {
			return fmt.Errorf("validation pipe: bind failed: %w", errors.Join(err, ErrBadRequest))
		}
		if err := validation.ValidateExhaustive(validate, &input); err != nil {
			return fmt.Errorf("validation failed: %w", errors.Join(err, ErrBadRequest))
		}
		ctx.Set(ValidatedBodyKey, &input)
		return nil
	}
}

// ValidatedBody retrieves the validated body stored by ValidationPipe[T].
// Panics with a clear message if ValidationPipe was not added to the route.
//
//	input := ligo.ValidatedBody[CreateUserInput](ctx)
func ValidatedBody[T any](ctx Context) *T {
	v, ok := ctx.Get(ValidatedBodyKey).(*T)
	if !ok {
		var zero T
		panic(fmt.Sprintf("ligo: ValidatedBody[%T]: no validated body in context — did you add ValidationPipe[%T] to this route?", zero, zero))
	}
	return v
}

// TransformPipe creates a generic pipe that transforms a path parameter string to type T.
// This reduces code duplication for similar transformation pipes.
func TransformPipe[T any](param string, transform func(string) (T, error), pipeName string) PipeFunc {
	return func(ctx Context) error {
		str := ctx.Param(param)
		result, err := transform(str)
		if err != nil {
			return fmt.Errorf("%s pipe: param %q: %w", pipeName, param, errors.Join(err, ErrBadRequest))
		}
		ctx.Set(param, result)
		return nil
	}
}

// ParseIntPipe reads path parameter param, parses it as int, and stores the
// result in ctx under param's name.
func ParseIntPipe(param string) PipeFunc {
	return TransformPipe(param, strconv.Atoi, "parse int")
}

// ParseBoolPipe reads path parameter param, parses it as bool, and stores the
// result in ctx under param's name.
// Accepts: 1, t, T, TRUE, true, True / 0, f, F, FALSE, false, False.
func ParseBoolPipe(param string) PipeFunc {
	return TransformPipe(param, strconv.ParseBool, "parse bool")
}

// UUIDPipe validates that path parameter param is a valid UUID and stores it
// in ctx under param's name.
func UUIDPipe(param string) PipeFunc {
	return func(ctx Context) error {
		str := ctx.Param(param)
		if err := validate.Var(str, "uuid"); err != nil {
			return fmt.Errorf("uuid pipe: param %q must be a valid UUID: %w", param, ErrBadRequest)
		}
		ctx.Set(param, str)
		return nil
	}
}

// TrimPipe removes leading and trailing whitespace from path parameter param
// and stores the trimmed value in ctx under param's name.
func TrimPipe(param string) PipeFunc {
	return func(ctx Context) error {
		ctx.Set(param, strings.TrimSpace(ctx.Param(param)))
		return nil
	}
}

// The two-pass validateExhaustive algorithm lives in internal/validation
// (validation.ValidateExhaustive). This package keeps only the binding/context
// glue around it.
