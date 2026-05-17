package ligo_test

import (
	"testing"

	"github.com/linkeunid/ligo"
	"github.com/linkeunid/ligo/adapters/echo"
)

// BenchmarkLigoAppCreation benchmarks Ligo app creation performance.
func BenchmarkLigoAppCreation(b *testing.B) {
	router := echo.NewAdapter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ligo.New(
			ligo.WithRouter(router),
			ligo.WithAddr(":0"),
		)
	}
}

// BenchmarkLigoModuleCreation benchmarks module creation performance.
func BenchmarkLigoModuleCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ligo.NewModule(
			"test",
			ligo.Providers(
				ligo.Value("test-value"),
				ligo.Factory[*BenchService](func() *BenchService {
					return &BenchService{}
				}),
			),
			ligo.Controllers(func() ligo.Controller {
				return &benchController{}
			}),
		)
	}
}

// BenchmarkLigoModuleWithMiddleware benchmarks module creation with middleware.
func BenchmarkLigoModuleWithMiddleware(b *testing.B) {
	testMiddleware := func(next ligo.HandlerFunc) ligo.HandlerFunc {
		return func(ctx *ligo.Context) error {
			return next(ctx)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ligo.NewModule(
			"test",
			ligo.Middlewares(
				func() ligo.Middleware {
					return testMiddleware
				},
			),
			ligo.Controllers(func() ligo.Controller {
				return &benchController{}
			}),
		)
	}
}

// BenchmarkLigoModuleWithLifecycleHooks benchmarks module creation with lifecycle hooks.
func BenchmarkLigoModuleWithLifecycleHooks(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ligo.NewModule(
			"test",
			ligo.OnModuleInit(func() error {
				return nil
			}),
			ligo.OnModuleDestroy(func() error {
				return nil
			}),
		)
	}
}

// BenchmarkLigoValueProvider benchmarks Value provider creation.
func BenchmarkLigoValueProvider(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ligo.Value("test-value")
	}
}

// BenchmarkLigoFactoryProvider benchmarks Factory provider creation.
func BenchmarkLigoFactoryProvider(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ligo.Factory[*BenchService](func() *BenchService {
			return &BenchService{}
		})
	}
}

// BenchmarkLigoTransientProvider benchmarks Transient provider creation.
func BenchmarkLigoTransientProvider(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ligo.Transient[*BenchService](func() *BenchService {
			return &BenchService{}
		})
	}
}

// BenchmarkLigoExportedProvider benchmarks Export provider creation.
func BenchmarkLigoExportedProvider(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ligo.Export(ligo.Factory[*BenchService](func() *BenchService {
			return &BenchService{}
		}))
	}
}

// BenchmarkLigoModuleRegistration benchmarks module registration performance.
func BenchmarkLigoModuleRegistration(b *testing.B) {
	router := echo.NewAdapter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		app := ligo.New(
			ligo.WithRouter(router),
			ligo.WithAddr(":0"),
		)
		module := ligo.NewModule(
			"test",
			ligo.Providers(
				ligo.Value("test-value"),
			),
		)
		b.StartTimer()
		app.Register(module)
	}
}

// BenchmarkLigoMultipleModules benchmarks registration of multiple modules.
func BenchmarkLigoMultipleModules(b *testing.B) {
	router := echo.NewAdapter()

	modules := []ligo.Module{
		ligo.NewModule(
			"module1",
			ligo.Providers(ligo.Value("value1")),
		),
		ligo.NewModule(
			"module2",
			ligo.Providers(ligo.Value("value2")),
		),
		ligo.NewModule(
			"module3",
			ligo.Providers(ligo.Value("value3")),
		),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		app := ligo.New(
			ligo.WithRouter(router),
			ligo.WithAddr(":0"),
		)
		b.StartTimer()
		app.Register(modules...)
	}
}

// BenchmarkLigoDynamicModule benchmarks dynamic module creation.
func BenchmarkLigoDynamicModule(b *testing.B) {
	factory := func(opts ...any) ligo.Module {
		name := "default"
		if len(opts) > 0 {
			if n, ok := opts[0].(string); ok {
				name = n
			}
		}
		return ligo.NewModule(
			name,
			ligo.Providers(ligo.Value(name)),
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ligo.NewModule(
			"dynamic",
			ligo.Dynamic(factory, "custom"),
		)
	}
}

// BenchmarkLigoControllerRegistration benchmarks controller registration.
func BenchmarkLigoControllerRegistration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ligo.NewModule(
			"test",
			ligo.Controllers(
				func() ligo.Controller {
					return &benchController{}
				},
				func() ligo.Controller {
					return &benchController{}
				},
				func() ligo.Controller {
					return &benchController{}
				},
			),
		)
	}
}

// BenchmarkLigoChainRouter benchmarks chain router creation.
func BenchmarkLigoChainRouter(b *testing.B) {
	router := echo.NewAdapter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ligo.NewChainRouter(router)
	}
}

// BenchmarkLigoGuardCreation benchmarks guard creation.
func BenchmarkLigoGuardCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ligo.RolesGuard("user", "admin")
	}
}

// BenchmarkLigoPipeCreation benchmarks pipe creation.
func BenchmarkLigoPipeCreation(b *testing.B) {
	type TestStruct struct {
		Name string `validate:"required"`
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ligo.ValidationPipe(&TestStruct{})
	}
}

// BenchmarkLigoInterceptorCreation benchmarks interceptor creation.
func BenchmarkLigoInterceptorCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ligo.TimeoutInterceptor(5)
	}
}

// BenchmarkLigoWithMultipleOptions benchmarks app creation with multiple options.
func BenchmarkLigoWithMultipleOptions(b *testing.B) {
	router := echo.NewAdapter()

	testMiddleware := func(next ligo.HandlerFunc) ligo.HandlerFunc {
		return func(ctx *ligo.Context) error {
			return next(ctx)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ligo.New(
			ligo.WithRouter(router),
			ligo.WithAddr(":0"),
			ligo.WithDebug(true),
			ligo.WithMiddleware(testMiddleware),
			ligo.OnStart(func(ctx any) error { return nil }),
			ligo.OnStop(func(ctx any) error { return nil }),
		)
	}
}

// Benchmark helper types

type BenchService struct{}

type benchController struct{}

func (c *benchController) Routes(r ligo.Router) {
	cr := ligo.NewChainRouter(r)
	cr.GET("/test", c.getTest).Handle()
}

func (c *benchController) getTest(ctx *ligo.Context) error {
	return ctx.OK(map[string]string{"status": "ok"})
}

// Test benchmark helpers

func TestBenchmarkHelpers(t *testing.T) {
	t.Run("bench service creation", func(t *testing.T) {
		svc := &BenchService{}
		// Verify service type is correct
		if _, ok := any(svc).(*BenchService); !ok {
			t.Error("Expected *BenchService")
		}
	})

	t.Run("bench controller creation", func(t *testing.T) {
		ctrl := &benchController{}
		// Verify controller type is correct
		if _, ok := any(ctrl).(*benchController); !ok {
			t.Error("Expected *benchController")
		}
	})
}
