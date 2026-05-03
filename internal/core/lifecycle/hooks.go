package lifecycle

// OnModuleInit is called when the module containing this provider is initialized.
// Runs per-module, depth-first during app startup.
type OnModuleInit interface {
	OnModuleInit() error
}

// OnApplicationBootstrap is called after all modules are initialized,
// but before the application starts serving (HTTP or signals).
// Runs once for all providers after OnModuleInit completes.
type OnApplicationBootstrap interface {
	OnApplicationBootstrap() error
}

// OnModuleDestroy is called when the module containing this provider is destroyed.
// Runs per-module, reverse depth-first during shutdown.
type OnModuleDestroy interface {
	OnModuleDestroy() error
}

// OnApplicationShutdown is called during application shutdown,
// before OnModuleDestroy. Runs once in reverse order.
type OnApplicationShutdown interface {
	OnApplicationShutdown() error
}
