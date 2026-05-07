package http

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ErrBadRequest is the sentinel error wrapped by pipes when client input is invalid.
// Exception handlers can detect it with errors.Is(err, ligo.ErrBadRequest).
var ErrBadRequest = errors.New("bad request")

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
			return fmt.Errorf("validation pipe: bind failed: %w", errors.Join(err, ErrBadRequest))
		}
		if err := validateExhaustive(validate, &input); err != nil {
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

// TransformPipe creates a generic pipe that transforms a path parameter string to type T.
// This reduces code duplication for similar transformation pipes.
func TransformPipe[T any](param string, transform func(string) (T, error), pipeName string) Pipe {
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

// tagRequired is the go-playground/validator tag name for the required constraint.
// When required fails, the validator skips all subsequent tags on that field (hasValue check),
// so a second pass is needed to surface min/email/oneof etc. for empty fields.
const tagRequired = "required"

// validateExhaustive runs two validation passes so fields that fail "required" also
// report all other tag failures in the same response.
//
// Why Two Passes Are Needed:
// The go-playground/validator library has a behavior where if a field fails the "required"
// validation, it skips all other validation tags for that field. This means a user would
// only see "required" error even if the field also fails "email", "min", etc.
//
// Two-Pass Strategy:
// 1. First pass: Run normal validation, collect all errors
// 2. If any "required" errors found:
//    - Create a copy of the struct
//    - Replace empty strings with "x" (passes required check)
//    - Run validation again on the modified struct
//    - Merge results from both passes
//
// Trade-offs:
// - Pro: Users see all validation errors at once
// - Con: Requires copying the entire struct (O(n) where n = struct size)
// - Alternative: Would require forking validator library or using custom validator
func validateExhaustive(v *validator.Validate, s any) error {
	err1 := v.Struct(s)
	if err1 == nil {
		return nil
	}
	var ve1 validator.ValidationErrors
	if !errors.As(err1, &ve1) {
		return err1
	}

	seen := make(map[string]struct{}, len(ve1))
	hasRequired := collectValidationErrors(ve1, seen)

	if !hasRequired {
		return ve1
	}

	cpy := createStructCopy(s)
	combined := append(validator.ValidationErrors(nil), ve1...)

	if err2 := v.Struct(cpy.Addr().Interface()); err2 != nil {
		var ve2 validator.ValidationErrors
		if !errors.As(err2, &ve2) {
			return combined
		}
		combined = combineValidationResults(ve2, seen, combined)
	}
	return combined
}

// collectValidationErrors collects validation errors and checks for required field errors.
func collectValidationErrors(ve validator.ValidationErrors, seen map[string]struct{}) bool {
	hasRequired := false
	for _, fe := range ve {
		seen[fe.Field()+"|"+fe.Tag()] = struct{}{}
		if fe.Tag() == tagRequired {
			hasRequired = true
		}
	}
	return hasRequired
}

// createStructCopy creates a modified copy of the struct for second-pass validation.
// Empty strings are replaced with "x" to pass required validation.
func createStructCopy(s any) reflect.Value {
	rv := reflect.ValueOf(s)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	cpy := reflect.New(rv.Type()).Elem()
	for i := 0; i < rv.NumField(); i++ {
		src := rv.Field(i)
		dst := cpy.Field(i)
		if !dst.CanSet() {
			continue
		}
		dst.Set(src)
		if src.Kind() == reflect.String && src.String() == "" {
			dst.SetString("x")
		}
	}
	return cpy
}

// combineValidationResults merges second-pass validation errors with first-pass results.
func combineValidationResults(ve2 validator.ValidationErrors, seen map[string]struct{}, combined validator.ValidationErrors) validator.ValidationErrors {
	for _, fe := range ve2 {
		tag, field := fe.Tag(), fe.Field()
		if tag == tagRequired {
			continue
		}
		key := field + "|" + tag
		if _, dup := seen[key]; dup {
			continue
		}
		combined = append(combined, fe)
		seen[key] = struct{}{}
	}
	return combined
}

// ParseIntPipe reads path parameter param, parses it as int, and stores the
// result in ctx under param's name.
func ParseIntPipe(param string) Pipe {
	return TransformPipe(param, strconv.Atoi, "parse int")
}

// ParseBoolPipe reads path parameter param, parses it as bool, and stores the
// result in ctx under param's name.
// Accepts: 1, t, T, TRUE, true, True / 0, f, F, FALSE, false, False.
func ParseBoolPipe(param string) Pipe {
	return TransformPipe(param, strconv.ParseBool, "parse bool")
}

// UUIDPipe validates that path parameter param is a valid UUID and stores it
// in ctx under param's name.
func UUIDPipe(param string) Pipe {
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
func TrimPipe(param string) Pipe {
	return func(ctx Context) error {
		ctx.Set(param, strings.TrimSpace(ctx.Param(param)))
		return nil
	}
}
