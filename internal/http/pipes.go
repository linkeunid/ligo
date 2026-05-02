package http

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// ValidatedBodyKey is the context key where ValidationPipe stores the bound and
// validated struct so handlers can retrieve it without re-binding.
const ValidatedBodyKey = "_ligo_validated"

// ValidationPipe binds the request body to T, validates it using struct tags,
// and stores the result in ctx under ValidatedBodyKey.
//
// Retrieve the validated struct in the handler with ValidatedBody[T]:
//
//	input := ligo.ValidatedBody[CreateUserInput](ctx)
func ValidationPipe[T any](_ *T) Pipe {
	return func(ctx Context) error {
		var input T
		if err := ctx.Bind(&input); err != nil {
			return fmt.Errorf("validation pipe: bind failed: %w", err)
		}
		if err := validate.Struct(input); err != nil {
			return fmt.Errorf("validation failed: %w", err)
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

// ParseIntPipe reads path parameter param, parses it as int, and stores the
// result in ctx under param's name.
func ParseIntPipe(param string) Pipe {
	return func(ctx Context) error {
		str := ctx.Param(param)
		i, err := strconv.Atoi(str)
		if err != nil {
			return fmt.Errorf("parse int pipe: param %q is not a valid integer", param)
		}
		ctx.Set(param, i)
		return nil
	}
}

// ParseBoolPipe reads path parameter param, parses it as bool, and stores the
// result in ctx under param's name.
// Accepts: 1, t, T, TRUE, true, True / 0, f, F, FALSE, false, False.
func ParseBoolPipe(param string) Pipe {
	return func(ctx Context) error {
		str := ctx.Param(param)
		b, err := strconv.ParseBool(str)
		if err != nil {
			return fmt.Errorf("parse bool pipe: param %q is not a valid boolean", param)
		}
		ctx.Set(param, b)
		return nil
	}
}

// UUIDPipe validates that path parameter param is a valid UUID and stores it
// in ctx under param's name.
func UUIDPipe(param string) Pipe {
	return func(ctx Context) error {
		str := ctx.Param(param)
		if err := validate.Var(str, "uuid"); err != nil {
			return fmt.Errorf("uuid pipe: param %q must be a valid UUID", param)
		}
		ctx.Set(param, str)
		return nil
	}
}

// TrimPipe removes leading and trailing whitespace from path parameter param
// and stores the trimmed value in ctx under param's name.
func TrimPipe(param string) Pipe {
	return func(ctx Context) error {
		ctx.Set(param, strings.TrimSpace(ctx.Param(param)))
		return nil
	}
}
