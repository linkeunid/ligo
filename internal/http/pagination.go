package http

import (
	nethttp "net/http"
	"reflect"
	"strconv"
)

// ListMeta is the meta block for plain list responses (no pagination).
type ListMeta struct {
	Count int `json:"count"`
}

// PageMeta is the meta block for paginated list responses.
type PageMeta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// ListResponse is { "data": [...], "meta": { "count": N } }.
type ListResponse struct {
	Data any      `json:"data"`
	Meta ListMeta `json:"meta"`
}

// PageResponse is { "data": [...], "meta": { page, per_page, total, total_pages } }.
type PageResponse struct {
	Data any      `json:"data"`
	Meta PageMeta `json:"meta"`
}

// NewListResponse wraps items in a list envelope. A nil items value (untyped
// nil or a typed nil slice) is coerced to an empty slice so the JSON body
// renders "data": [] instead of "data": null.
func NewListResponse(items any) ListResponse {
	items = safeSlice(items)
	return ListResponse{
		Data: items,
		Meta: ListMeta{Count: sliceLen(items)},
	}
}

// NewPageResponse wraps items in a paginated envelope. items follows the same
// nil-coercion rule as NewListResponse. totalPages is computed as
// ceil(total / perPage), and is 0 when perPage <= 0.
func NewPageResponse(items any, page, perPage int, total int64) PageResponse {
	items = safeSlice(items)
	totalPages := 0
	if perPage > 0 {
		totalPages = int((total + int64(perPage) - 1) / int64(perPage))
	}
	return PageResponse{
		Data: items,
		Meta: PageMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}
}

// ListQuery captures common pagination query params. PerPageSet records
// whether ?per_page= was present in the request, so Normalize can distinguish
// "absent" (apply default) from "explicitly 0" (honor as LIMIT 0).
type ListQuery struct {
	Page       int
	PerPage    int
	PerPageSet bool
}

// Normalize clamps the query into a safe range:
//
//   - Page < 1 → 1.
//   - PerPage falls back to defaultPerPage only when ?per_page= was absent
//     (PerPageSet == false). An explicit ?per_page=0 stays 0.
//   - PerPage is clamped to [0, maxPerPage] when maxPerPage > 0; negatives
//     become 0.
func (q *ListQuery) Normalize(defaultPerPage, maxPerPage int) {
	if q.Page < 1 {
		q.Page = 1
	}
	if !q.PerPageSet {
		q.PerPage = defaultPerPage
	}
	if q.PerPage < 0 {
		q.PerPage = 0
	}
	if maxPerPage > 0 && q.PerPage > maxPerPage {
		q.PerPage = maxPerPage
	}
}

// Offset returns the SQL OFFSET for the (normalized) page.
func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.PerPage
}

// ParseListQuery reads ?page= and ?per_page= from the request. Missing or
// invalid values stay zero; call Normalize to apply defaults.
func ParseListQuery(r *nethttp.Request) ListQuery {
	if r == nil || r.URL == nil {
		return ListQuery{}
	}
	qs := r.URL.Query()
	page, _ := strconv.Atoi(qs.Get("page"))
	raw, ok := qs["per_page"]
	perPage := 0
	set := false
	if ok && len(raw) > 0 && raw[0] != "" {
		if n, err := strconv.Atoi(raw[0]); err == nil {
			perPage = n
			set = true
		}
	}
	return ListQuery{Page: page, PerPage: perPage, PerPageSet: set}
}

func safeSlice(v any) any {
	if v == nil {
		return []any{}
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice && rv.IsNil() {
		return []any{}
	}
	return v
}

func sliceLen(v any) int {
	if v == nil {
		return 0
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return rv.Len()
	}
	return 0
}
