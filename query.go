package ligo

import (
	nethttp "net/http"

	"github.com/linkeunid/ligo/internal/http"
)

// Query returns a single query-string value or "" when absent.
func Query(r *nethttp.Request, key string) string {
	return http.Query(r, key)
}

// QueryDefault returns the query value or def when absent/empty.
func QueryDefault(r *nethttp.Request, key, def string) string {
	return http.QueryDefault(r, key, def)
}

// QueryInt parses a query value as int, returning def on missing/invalid.
func QueryInt(r *nethttp.Request, key string, def int) int {
	return http.QueryInt(r, key, def)
}

// BindQuery decodes URL query parameters into the struct pointed to by v.
// Fields are matched by the `query:"name"` tag; untagged fields are ignored.
// Supported field kinds: string, signed/unsigned ints, bool, floats, and
// slices of those (accepted as repeated params or comma-separated).
//
// Example:
//
//	type UserFilter struct {
//	    Name  string `query:"name"`
//	    Email string `query:"email"`
//	    Sort  string `query:"sort"`
//	}
//	var f UserFilter
//	if err := ligo.BindQuery(ctx.Request(), &f); err != nil { ... }
func BindQuery(r *nethttp.Request, v any) error {
	return http.BindQuery(r, v)
}
