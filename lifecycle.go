package ligo

// Lifecycle hooks allow providers and controllers to execute code at specific
// application lifecycle stages.
//
// To use lifecycle hooks, implement the hook methods directly on your provider
// or controller structs. The framework will automatically detect and execute
// them at the appropriate time.
//
// Example:
//
//	type DatabaseService struct {
//	    db *sql.DB
//	}
//
//	func (s *DatabaseService) OnModuleInit() error {
//	    var err error
//	    s.db = sql.Open("postgres", "dsn")
//	    return err
//	}
//
//	func (s *DatabaseService) BeforeApplicationShutdown() error {
//	    // Stop accepting new connections, finish in-flight requests
//	    return s.db.Close()
//	}
//
//	func (s *DatabaseService) OnApplicationShutdown() error {
//	    // Final cleanup after all connections are drained
//	    return nil
//	}
//
// Available hooks (in execution order):
//
//   - OnModuleInit() error — Called when the module containing this provider
//     is initialized. Runs per-module, depth-first during app startup.
//
//   - OnApplicationBootstrap() error — Called after all modules are initialized,
//     but before the application starts serving (HTTP or signals).
//
//   - BeforeApplicationShutdown() error — Called before shutdown begins,
//     before OnApplicationShutdown. Useful for graceful drain-stop scenarios.
//     Runs once in reverse order.
//
//   - OnApplicationShutdown() error — Called during application shutdown,
//     after BeforeApplicationShutdown, before OnModuleDestroy. Runs once in reverse order.
//
//   - OnModuleDestroy() error — Called when the module containing this provider
//     is destroyed. Runs per-module, reverse depth-first during shutdown.
//
// For module-level functional hooks, see OnModuleInit and OnModuleDestroy
// in the module package.
