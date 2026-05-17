package http

import (
	"testing"
)

func TestChainRouter_EmbedsRouter(t *testing.T) {
	mock := &mockRouter{}
	cr := NewChainRouter(mock)
	cr.Use(func(next HandlerFunc) HandlerFunc { return next })
	if mock.useCount != 1 {
		t.Errorf("Use forwarding: useCount = %d", mock.useCount)
	}

	cr.Group("/api")
	if len(mock.groupPrefixes) != 1 || mock.groupPrefixes[0] != "/api" {
		t.Errorf("Group forwarding: prefixes = %v", mock.groupPrefixes)
	}
}

func TestChainRouter_VerbsBuildRoutes(t *testing.T) {
	verbs := []struct {
		name   string
		method string
		fn     func(ChainRouter) RouteBuilder
	}{
		{"GET", "GET", func(cr ChainRouter) RouteBuilder { return cr.GET("/x") }},
		{"POST", "POST", func(cr ChainRouter) RouteBuilder { return cr.POST("/x") }},
		{"PUT", "PUT", func(cr ChainRouter) RouteBuilder { return cr.PUT("/x") }},
		{"DELETE", "DELETE", func(cr ChainRouter) RouteBuilder { return cr.DELETE("/x") }},
		{"PATCH", "PATCH", func(cr ChainRouter) RouteBuilder { return cr.PATCH("/x") }},
		{"OPTIONS", "OPTIONS", func(cr ChainRouter) RouteBuilder { return cr.OPTIONS("/x") }},
		{"HEAD", "HEAD", func(cr ChainRouter) RouteBuilder { return cr.HEAD("/x") }},
	}
	for _, v := range verbs {
		t.Run(v.name, func(t *testing.T) {
			mock := &mockRouter{}
			var captured string
			mock.handleFunc = func(method, _ string, _ HandlerFunc) { captured = method }

			cr := NewChainRouter(mock)
			v.fn(cr).Handle(func(Context) error { return nil })

			if captured != v.method {
				t.Errorf("method = %q, want %q", captured, v.method)
			}
		})
	}
}
