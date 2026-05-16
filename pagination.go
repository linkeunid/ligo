package ligo

import (
	nethttp "net/http"

	"github.com/linkeunid/ligo/internal/http"
)

// ListMeta is the meta block for plain list responses (no pagination).
type ListMeta = http.ListMeta

// PageMeta is the meta block for paginated list responses.
type PageMeta = http.PageMeta

// ListResponse is { "data": [...], "meta": { "count": N } }.
type ListResponse = http.ListResponse

// PageResponse is { "data": [...], "meta": { page, per_page, total, total_pages } }.
type PageResponse = http.PageResponse

// ListQuery captures common pagination query params (?page=, ?per_page=).
type ListQuery = http.ListQuery

// NewListResponse builds a ListResponse, coercing nil slices to [].
func NewListResponse(items any) ListResponse {
	return http.NewListResponse(items)
}

// NewPageResponse builds a PageResponse, coercing nil slices to [] and
// computing total_pages from total / perPage.
func NewPageResponse(items any, page, perPage int, total int64) PageResponse {
	return http.NewPageResponse(items, page, perPage, total)
}

// ParseListQuery reads ?page= and ?per_page= from the request. Missing or
// invalid values stay zero; call Normalize to apply defaults.
//
// Example:
//
//	func (c *UserController) List(ctx ligo.Context) error {
//	    q := ligo.ParseListQuery(ctx.Request())
//	    q.Normalize(20, 100) // default 20, max 100
//	    users, total, err := c.repo.FindPage(ctx.Request().Context(), q.PerPage, q.Offset())
//	    if err != nil { return err }
//	    return ctx.Paginated(users, q.Page, q.PerPage, total)
//	}
func ParseListQuery(r *nethttp.Request) ListQuery {
	return http.ParseListQuery(r)
}

// Paginate is a one-shot helper: ParseListQuery + Normalize. Honors
// ?per_page=0 (LIMIT 0); falls back to defaultPerPage only when ?per_page=
// is absent. Caps PerPage at maxPerPage when maxPerPage > 0.
func Paginate(r *nethttp.Request, defaultPerPage, maxPerPage int) ListQuery {
	q := http.ParseListQuery(r)
	q.Normalize(defaultPerPage, maxPerPage)
	return q
}
