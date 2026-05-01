package ligo

type options struct {
	router      Router
	addr        string
	middlewares []Middleware
	debug       bool
	logger      Logger
	onStart     []LifecycleHook
	onStop      []LifecycleHook
}

// Logger is the interface for framework logging.
type Logger interface {
	Debug(msg string, args ...any)
	Error(msg string, args ...any)
}

// LifecycleHook is a function called during app lifecycle events.
type LifecycleHook func(ctx any) error

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

// OnStart adds a hook to run on app startup.
func OnStart(hook LifecycleHook) Option {
	return func(o *options) {
		o.onStart = append(o.onStart, hook)
	}
}

// OnStop adds a hook to run on app shutdown.
func OnStop(hook LifecycleHook) Option {
	return func(o *options) {
		o.onStop = append(o.onStop, hook)
	}
}
