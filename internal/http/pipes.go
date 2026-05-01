package http

import (
	"fmt"
	"strconv"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// extractString extracts a string value from input of various types.
// For map types, param is used as the key to extract.
func extractString(input any, param string) (string, error) {
	switch v := input.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case map[string]string:
		str, ok := v[param]
		if !ok {
			return "", fmt.Errorf("key %q not found", param)
		}
		return str, nil
	default:
		return "", fmt.Errorf("unsupported input type %T", input)
	}
}

// ValidationPipe validates a struct using struct tags.
// It uses go-playground/validator for validation.
func ValidationPipe[T any](v *T) Pipe {
	return func(input any) (any, error) {
		if input == nil {
			return nil, nil
		}

		typed, ok := input.(T)
		if !ok {
			if ptr, ok := input.(*T); ok {
				if ptr == nil {
					return nil, nil
				}
				typed = *ptr
			} else {
				return nil, fmt.Errorf("validation pipe: type mismatch, expected %T or *%T", *new(T), *new(T))
			}
		}

		if err := validate.Struct(typed); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
		return input, nil
	}
}

// ParseIntPipe parses a string parameter to int.
// The param is the key to extract from map-like inputs.
func ParseIntPipe(param string) Pipe {
	return func(input any) (any, error) {
		str, err := extractString(input, param)
		if err != nil {
			return nil, fmt.Errorf("parse int pipe: %w", err)
		}

		i, err := strconv.Atoi(str)
		if err != nil {
			return nil, fmt.Errorf("parse int pipe: %w", err)
		}
		return i, nil
	}
}

// ParseBoolPipe parses a string parameter to bool.
// Accepts: 1, t, T, TRUE, true, True for true
//          0, f, F, FALSE, false, False for false
func ParseBoolPipe(param string) Pipe {
	return func(input any) (any, error) {
		str, err := extractString(input, param)
		if err != nil {
			return nil, fmt.Errorf("parse bool pipe: %w", err)
		}

		b, err := strconv.ParseBool(str)
		if err != nil {
			return nil, fmt.Errorf("parse bool pipe: %w", err)
		}
		return b, nil
	}
}

// UUIDPipe validates that a string is a valid UUID format.
func UUIDPipe(param string) Pipe {
	return func(input any) (any, error) {
		str, err := extractString(input, param)
		if err != nil {
			return nil, fmt.Errorf("uuid pipe: %w", err)
		}

		if err := validate.Var(str, "uuid"); err != nil {
			return nil, fmt.Errorf("uuid pipe: %w", err)
		}
		return str, nil
	}
}

// TrimPipe removes leading and trailing whitespace from a string.
func TrimPipe(param string) Pipe {
	return func(input any) (any, error) {
		str, err := extractString(input, param)
		if err != nil {
			return nil, fmt.Errorf("trim pipe: %w", err)
		}

		if m, ok := input.(map[string]string); ok {
			m[param] = str
			return input, nil
		}
		return str, nil
	}
}
