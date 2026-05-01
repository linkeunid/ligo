package http

import (
	"fmt"
)

type routeBuilder struct {
	router           Router
	method           string
	path             string
	guards           []Guard
	pipes            []Pipe
	interceptors     []Interceptor
	middleware       []Middleware
	exceptionFilters []ExceptionFilter
}

// NewRouteBuilder creates a new route builder.
func NewRouteBuilder(router Router, method, path string) RouteBuilder {
	return &routeBuilder{
		router: router,
		method: method,
		path:   path,
	}
}

func (rb *routeBuilder) Guard(guards ...Guard) RouteBuilder {
	rb.guards = append(rb.guards, guards...)
	return rb
}

func (rb *routeBuilder) Pipe(pipes ...Pipe) RouteBuilder {
	rb.pipes = append(rb.pipes, pipes...)
	return rb
}

func (rb *routeBuilder) Intercept(interceptors ...Interceptor) RouteBuilder {
	rb.interceptors = append(rb.interceptors, interceptors...)
	return rb
}

func (rb *routeBuilder) Use(middleware ...Middleware) RouteBuilder {
	rb.middleware = append(rb.middleware, middleware...)
	return rb
}

func (rb *routeBuilder) Filter(filters ...ExceptionFilter) RouteBuilder {
	rb.exceptionFilters = append(rb.exceptionFilters, filters...)
	return rb
}

func (rb *routeBuilder) Handle(handler HandlerFunc) {
	wrapped := handler

	// Apply interceptors (wrap the entire cycle)
	for i := len(rb.interceptors) - 1; i >= 0; i-- {
		interceptor := rb.interceptors[i]
		prev := wrapped
		wrapped = func(ctx Context) error {
			return interceptor(ctx, prev)
		}
	}

	// Apply pipes (transform input)
	if len(rb.pipes) > 0 {
		prev := wrapped
		wrapped = func(ctx Context) error {
			// Apply pipes to request body if present
			if ctx.Request().Body != nil {
				body, err := ctx.Request().GetBody()
				if err == nil {
					value := any(body)
					for _, pipe := range rb.pipes {
						transformed, err := pipe(value)
						if err != nil {
							return fmt.Errorf("pipe error: %w", err)
						}
						value = transformed
					}
					// Store transformed value in context
					ctx.Set("pipe:result", value)
				}
			}
			return prev(ctx)
		}
	}

	// Apply guards (authorization check)
	if len(rb.guards) > 0 {
		prev := wrapped
		wrapped = func(ctx Context) error {
			for _, guard := range rb.guards {
				allowed, err := guard(ctx)
				if err != nil {
					return err
				}
				if !allowed {
					return fmt.Errorf("guard denied access")
				}
			}
			return prev(ctx)
		}
	}

	// Apply middleware
	for i := len(rb.middleware) - 1; i >= 0; i-- {
		mw := rb.middleware[i]
		prev := wrapped
		wrapped = func(ctx Context) error {
			return mw(prev)(ctx)
		}
	}

	// Wrap with exception filters
	finalHandler := wrapped
	if len(rb.exceptionFilters) > 0 {
		finalHandler = func(ctx Context) error {
			err := wrapped(ctx)
			if err != nil {
				// Apply exception filters in order
				for _, filter := range rb.exceptionFilters {
					if filterErr := filter(err, ctx); filterErr != nil {
						return filterErr
					}
				}
				return err
			}
			return nil
		}
	}

	rb.router.Handle(rb.method, rb.path, finalHandler)
}
