package http_test

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	httpifc "github.com/linkeunid/ligo/internal/http"
	ligotest "github.com/linkeunid/ligo/internal/testing"
)

func newTestCtx() (*httpifc.Context, *ligotest.MockContext) {
	mock := ligotest.NewMockContext()
	return httpifc.NewContext(mock), mock
}

func TestContext_SuccessHelpers(t *testing.T) {
	cases := []struct {
		name     string
		call     func(*httpifc.Context) error
		wantCode int
	}{
		{"OK", func(c *httpifc.Context) error { return c.OK("x") }, 200},
		{"Created", func(c *httpifc.Context) error { return c.Created("x") }, 201},
		{"Accepted", func(c *httpifc.Context) error { return c.Accepted("x") }, 202},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, mock := newTestCtx()
			if err := tc.call(ctx); err != nil {
				t.Fatalf("%s returned error: %v", tc.name, err)
			}
			if mock.LastJSONCode != tc.wantCode {
				t.Errorf("expected code %d, got %d", tc.wantCode, mock.LastJSONCode)
			}
		})
	}
}

func TestContext_NoContent(t *testing.T) {
	ctx, mock := newTestCtx()
	if err := ctx.NoContent(); err != nil {
		t.Fatalf("NoContent: %v", err)
	}
	if mock.LastStringCode != 204 || mock.LastStringBody != "" {
		t.Errorf("expected 204 empty, got %d %q", mock.LastStringCode, mock.LastStringBody)
	}
}

func TestContext_List(t *testing.T) {
	ctx, mock := newTestCtx()
	if err := ctx.List([]string{"a", "b"}); err != nil {
		t.Fatalf("List: %v", err)
	}
	got, ok := mock.LastJSONBody.(httpifc.ListResponse)
	if !ok {
		t.Fatalf("expected ListResponse, got %T", mock.LastJSONBody)
	}
	if got.Meta.Count != 2 {
		t.Errorf("expected count 2, got %d", got.Meta.Count)
	}
}

func TestContext_Paginated(t *testing.T) {
	ctx, mock := newTestCtx()
	if err := ctx.Paginated([]string{"a"}, 1, 20, 42); err != nil {
		t.Fatalf("Paginated: %v", err)
	}
	got, ok := mock.LastJSONBody.(httpifc.PageResponse)
	if !ok {
		t.Fatalf("expected PageResponse, got %T", mock.LastJSONBody)
	}
	if got.Meta.Total != 42 || got.Meta.Page != 1 {
		t.Errorf("meta wrong: %+v", got.Meta)
	}
}

func TestContext_ErrorHelpers_StatusAndDefaultMessage(t *testing.T) {
	cases := []struct {
		name string
		call func(*httpifc.Context, ...string) error
		code int
		text string
	}{
		{"BadRequest", (*httpifc.Context).BadRequest, 400, "Bad Request"},
		{"Unauthorized", (*httpifc.Context).Unauthorized, 401, "Unauthorized"},
		{"Forbidden", (*httpifc.Context).Forbidden, 403, "Forbidden"},
		{"NotFound", (*httpifc.Context).NotFound, 404, "Not Found"},
		{"MethodNotAllowed", (*httpifc.Context).MethodNotAllowed, 405, "Method Not Allowed"},
		{"NotAcceptable", (*httpifc.Context).NotAcceptable, 406, "Not Acceptable"},
		{"RequestTimeout", (*httpifc.Context).RequestTimeout, 408, "Request Timeout"},
		{"Conflict", (*httpifc.Context).Conflict, 409, "Conflict"},
		{"Gone", (*httpifc.Context).Gone, 410, "Gone"},
		{"PreconditionFailed", (*httpifc.Context).PreconditionFailed, 412, "Precondition Failed"},
		{"PayloadTooLarge", (*httpifc.Context).PayloadTooLarge, 413, "Request Entity Too Large"},
		{"UnsupportedMediaType", (*httpifc.Context).UnsupportedMediaType, 415, "Unsupported Media Type"},
		{"UnprocessableEntity", (*httpifc.Context).UnprocessableEntity, 422, "Unprocessable Entity"},
		{"TooManyRequests", (*httpifc.Context).TooManyRequests, 429, "Too Many Requests"},
		{"ImATeapot", (*httpifc.Context).ImATeapot, 418, "I'm a teapot"},
		{"InternalServerError", (*httpifc.Context).InternalServerError, 500, "Internal Server Error"},
		{"NotImplemented", (*httpifc.Context).NotImplemented, 501, "Not Implemented"},
		{"BadGateway", (*httpifc.Context).BadGateway, 502, "Bad Gateway"},
		{"ServiceUnavailable", (*httpifc.Context).ServiceUnavailable, 503, "Service Unavailable"},
		{"GatewayTimeout", (*httpifc.Context).GatewayTimeout, 504, "Gateway Timeout"},
		{"HTTPVersionNotSupported", (*httpifc.Context).HTTPVersionNotSupported, 505, "HTTP Version Not Supported"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, mock := newTestCtx()
			if err := tc.call(ctx); err != nil {
				t.Fatalf("returned error: %v", err)
			}
			if mock.LastJSONCode != tc.code {
				t.Errorf("expected code %d, got %d", tc.code, mock.LastJSONCode)
			}
			body, ok := mock.LastJSONBody.(map[string]string)
			if !ok {
				t.Fatalf("expected map[string]string body, got %T", mock.LastJSONBody)
			}
			if body["error"] != tc.text {
				t.Errorf("expected default message %q, got %q", tc.text, body["error"])
			}
		})
	}
}

func TestContext_ErrorHelpers_CustomMessage(t *testing.T) {
	ctx, mock := newTestCtx()
	if err := ctx.NotFound("user missing"); err != nil {
		t.Fatalf("NotFound: %v", err)
	}
	body := mock.LastJSONBody.(map[string]string)
	if body["error"] != "user missing" {
		t.Errorf("expected custom message, got %q", body["error"])
	}
}

func TestContext_ErrorHelpers_EmptyMessageFallsBackToDefault(t *testing.T) {
	ctx, mock := newTestCtx()
	if err := ctx.NotFound(""); err != nil {
		t.Fatalf("NotFound: %v", err)
	}
	body := mock.LastJSONBody.(map[string]string)
	if body["error"] != "Not Found" {
		t.Errorf("expected fallback, got %q", body["error"])
	}
}

func TestContext_QueryHelpers(t *testing.T) {
	mock := ligotest.NewMockContext()
	mock.SetRequest(httptest.NewRequest("GET", "/?page=3&size=abc", nil))
	ctx := httpifc.NewContext(mock)
	if got := ctx.QueryDefault("page", "1"); got != "3" {
		t.Errorf("QueryDefault: expected 3, got %q", got)
	}
	if got := ctx.QueryDefault("missing", "fallback"); got != "fallback" {
		t.Errorf("QueryDefault fallback: got %q", got)
	}
	if got := ctx.QueryInt("page", 1); got != 3 {
		t.Errorf("QueryInt: expected 3, got %d", got)
	}
	if got := ctx.QueryInt("size", 10); got != 10 {
		t.Errorf("QueryInt parse-fail fallback: got %d", got)
	}
}

func TestContext_Paginate(t *testing.T) {
	mock := ligotest.NewMockContext()
	mock.SetRequest(httptest.NewRequest("GET", "/?page=2&per_page=50", nil))
	ctx := httpifc.NewContext(mock)
	q := ctx.Paginate(20, 100)
	if q.Page != 2 || q.PerPage != 50 {
		t.Errorf("Paginate: got %+v", q)
	}
}

func TestContext_PropagatesJSONError(t *testing.T) {
	ctx, mock := newTestCtx()
	mock.JSONErr = errors.New("write failed")
	if err := ctx.OK("x"); err == nil || !strings.Contains(err.Error(), "write failed") {
		t.Errorf("expected JSONErr to bubble, got %v", err)
	}
}
