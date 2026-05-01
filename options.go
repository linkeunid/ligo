package ligo

type options struct {
	router      Router
	addr        string
	middlewares []Middleware
	debug       bool
	logger      Logger
}

// Logger is the interface for framework logging.
type Logger interface {
	Debug(msg string, args ...any)
	Error(msg string, args ...any)
}

// Option configures the App.
type Option func(*options)

func defaultOptions() options {
	return options{
		addr: ":8080",
	}
}

// WithRouter sets the HTTP router adapter.
func WithRouter(r Router) Option {
	return func(o *options) {
		o.router = r
	}
}

// WithAddr sets the server address.
func WithAddr(addr string) Option {
	return func(o *options) {
		o.addr = addr
	}
}

// WithDebug enables debug logging.
func WithDebug(debug bool) Option {
	return func(o *options) {
		o.debug = debug
	}
}

// WithMiddleware adds global middleware.
func WithMiddleware(mw ...Middleware) Option {
	return func(o *options) {
		o.middlewares = append(o.middlewares, mw...)
	}
}

// WithLogger sets the logger.
func WithLogger(l Logger) Option {
	return func(o *options) {
		o.logger = l
	}
}
