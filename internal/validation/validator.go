package validation

import (
	"errors"
	"reflect"

	"github.com/go-playground/validator/v10"
)

// Validator wraps the go-playground/validator with common patterns.
type Validator struct {
	validate *validator.Validate
}

// New creates a new Validator instance.
func New() *Validator {
	return &Validator{
		validate: validator.New(),
	}
}

// Validate validates a struct and returns all validation errors.
// This uses exhaustive validation to ensure all errors are reported.
func (v *Validator) Validate(s any) error {
	return validateExhaustive(v.validate, s)
}

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
	hasRequired := false
	for _, fe := range ve1 {
		seen[fe.Field()+"|"+fe.Tag()] = struct{}{}
		if fe.Tag() == "required" {
			hasRequired = true
		}
	}
	if !hasRequired {
		return ve1
	}
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
	combined := append(validator.ValidationErrors(nil), ve1...)
	if err2 := v.Struct(cpy.Addr().Interface()); err2 != nil {
		var ve2 validator.ValidationErrors
		if !errors.As(err2, &ve2) {
			return combined
		}
		for _, fe := range ve2 {
			tag, field := fe.Tag(), fe.Field()
			if tag == "required" {
				continue
			}
			key := field + "|" + tag
			if _, dup := seen[key]; dup {
				continue
			}
			combined = append(combined, fe)
			seen[key] = struct{}{}
		}
	}
	return combined
}

// DefaultValidator is the default validator instance.
var DefaultValidator = New()
