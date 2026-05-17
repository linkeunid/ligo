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
	return ValidateExhaustive(v.validate, s)
}

// formatSensitiveTags lists validator tags whose checks fail against the
// "x" sentinel used in the second pass. When a field already reported
// "required" in the first pass, contradictory second-pass errors from these
// tags ("must be valid email", "must be one of …") are suppressed so the
// user sees just the original "required" message.
var formatSensitiveTags = map[string]struct{}{
	"email":        {},
	"url":          {},
	"uri":          {},
	"uuid":         {},
	"uuid3":        {},
	"uuid4":        {},
	"uuid5":        {},
	"ip":           {},
	"ipv4":         {},
	"ipv6":         {},
	"cidr":         {},
	"oneof":        {},
	"hexadecimal":  {},
	"hexcolor":     {},
	"rgb":          {},
	"rgba":         {},
	"alpha":        {},
	"alphanum":     {},
	"numeric":      {},
	"hostname":     {},
	"fqdn":         {},
	"datetime":     {},
	"isbn":         {},
	"isbn10":       {},
	"isbn13":       {},
	"contains":     {},
	"containsany":  {},
	"containsrune": {},
	"startswith":   {},
	"endswith":     {},
	"e164":         {},
	"base64":       {},
	"base64url":    {},
	"json":         {},
	"jwt":          {},
}

// ValidateExhaustive runs two validation passes so fields that fail "required"
// also report other tag failures. Shared with internal/http/pipes so a fix
// here propagates everywhere.
//
// Why two passes: when a field fails "required", go-playground/validator
// short-circuits and skips the remaining tags on that field. A user who
// submits an empty email would otherwise only see "required" without
// learning that "min=5,email" would also have failed.
//
// Two-pass strategy:
//  1. First pass: run normal validation, collect all errors.
//  2. If any "required" failed: copy the struct, substitute "x" for empty
//     strings, run validation again, and surface tag failures that weren't
//     already reported.
//
// UX guard: the "x" substitution itself fails format-sensitive tags
// (email/uuid/oneof/url/regex/etc.), which would surface contradictory
// messages ("required" AND "must be valid email") for the same empty input.
// For fields that failed "required" in pass 1, we drop pass-2 errors whose
// tag is in formatSensitiveTags.
func ValidateExhaustive(v *validator.Validate, s any) error {
	err1 := v.Struct(s)
	if err1 == nil {
		return nil
	}
	var ve1 validator.ValidationErrors
	if !errors.As(err1, &ve1) {
		return err1
	}
	seen := make(map[string]struct{}, len(ve1))
	requiredFields := make(map[string]struct{})
	for _, fe := range ve1 {
		seen[fe.Field()+"|"+fe.Tag()] = struct{}{}
		if fe.Tag() == "required" {
			requiredFields[fe.Field()] = struct{}{}
		}
	}
	if len(requiredFields) == 0 {
		return ve1
	}
	rv := reflect.ValueOf(s)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	cpy := reflect.New(rv.Type()).Elem()
	for i := range rv.NumField() {
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
			if _, requiredHere := requiredFields[field]; requiredHere {
				if _, formatTag := formatSensitiveTags[tag]; formatTag {
					continue
				}
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
