package http

import (
	"fmt"
	nethttp "net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// Query returns the raw value of a single URL query parameter, or "" when
// absent. Equivalent to r.URL.Query().Get(key) but cheaper to type from a
// handler.
func Query(r *nethttp.Request, key string) string {
	return r.URL.Query().Get(key)
}

// QueryDefault returns the value of a query parameter, falling back to def
// when the parameter is absent or empty.
func QueryDefault(r *nethttp.Request, key, def string) string {
	if v := r.URL.Query().Get(key); v != "" {
		return v
	}
	return def
}

// QueryInt parses a query parameter as an int. Returns def for missing,
// empty, or non-numeric values.
func QueryInt(r *nethttp.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// BindQuery decodes URL query parameters into the struct pointed to by v.
// Fields are matched by the `query:"name"` tag; untagged fields are ignored.
// Supported field kinds: string, int/int8/int16/int32/int64, uint/.../uint64,
// bool, float32/float64, and slices of those base kinds (multi-valued or
// comma-separated). Parse errors are aggregated and returned as a single error.
func BindQuery(r *nethttp.Request, v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("ligo: BindQuery requires a non-nil pointer to a struct")
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("ligo: BindQuery target must be a struct, got %s", rv.Kind())
	}

	fields := queryFieldsOf(rv.Type())
	values := r.URL.Query()
	var errs []string
	for _, f := range fields {
		raw, ok := values[f.name]
		if !ok || len(raw) == 0 {
			continue
		}
		if err := setQueryField(rv.FieldByIndex(f.index), raw); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", f.name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("ligo: BindQuery: %s", strings.Join(errs, "; "))
	}
	return nil
}

type queryField struct {
	name  string
	index []int
}

var queryFieldsCache sync.Map

func queryFieldsOf(t reflect.Type) []queryField {
	if cached, ok := queryFieldsCache.Load(t); ok {
		return cached.([]queryField)
	}
	var out []queryField
	var walk func(rt reflect.Type, parent []int)
	walk = func(rt reflect.Type, parent []int) {
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			path := append(append([]int{}, parent...), i)
			if f.Anonymous && f.Type.Kind() == reflect.Struct {
				walk(f.Type, path)
				continue
			}
			tag := f.Tag.Get("query")
			if tag == "" || tag == "-" {
				continue
			}
			name := strings.SplitN(tag, ",", 2)[0]
			out = append(out, queryField{name: name, index: path})
		}
	}
	walk(t, nil)
	queryFieldsCache.Store(t, out)
	return out
}

func setQueryField(dst reflect.Value, raw []string) error {
	if dst.Kind() == reflect.Slice {
		// Accept both ?k=a&k=b and ?k=a,b forms.
		var items []string
		for _, r := range raw {
			if r == "" {
				continue
			}
			items = append(items, strings.Split(r, ",")...)
		}
		slice := reflect.MakeSlice(dst.Type(), len(items), len(items))
		for i, s := range items {
			if err := assignScalar(slice.Index(i), strings.TrimSpace(s)); err != nil {
				return err
			}
		}
		dst.Set(slice)
		return nil
	}
	return assignScalar(dst, raw[0])
}

func assignScalar(dst reflect.Value, s string) error {
	switch dst.Kind() {
	case reflect.String:
		dst.SetString(s)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		dst.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, dst.Type().Bits())
		if err != nil {
			return err
		}
		dst.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, dst.Type().Bits())
		if err != nil {
			return err
		}
		dst.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(s, dst.Type().Bits())
		if err != nil {
			return err
		}
		dst.SetFloat(n)
	default:
		return fmt.Errorf("unsupported kind %s", dst.Kind())
	}
	return nil
}
