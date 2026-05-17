package reflectutil

import (
	"reflect"
)

// ExtractTypeName extracts the type name from a value or function.
// For pointers, it returns the name of the pointed-to type.
// For functions, it returns the name of the return type.
func ExtractTypeName(v any) string {
	if v == nil {
		return "unknown"
	}

	typ := reflect.TypeOf(v)

	// For functions, get the return type
	if typ.Kind() == reflect.Func {
		if typ.NumOut() > 0 {
			retType := typ.Out(0)
			if retType.Kind() == reflect.Ptr {
				retType = retType.Elem()
			}
			if retType.Name() != "" {
				return retType.Name()
			}
		}
		return typ.Name()
	}

	// For pointers, get the element type
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	return typ.Name()
}

// IsPointerType checks if the given value is a pointer.
func IsPointerType(v any) bool {
	if v == nil {
		return false
	}
	return reflect.TypeOf(v).Kind() == reflect.Ptr
}

// GetElementType returns the element type for pointers, or the type itself for non-pointers.
func GetElementType(v any) reflect.Type {
	if v == nil {
		return nil
	}
	typ := reflect.TypeOf(v)
	if typ.Kind() == reflect.Ptr {
		return typ.Elem()
	}
	return typ
}
