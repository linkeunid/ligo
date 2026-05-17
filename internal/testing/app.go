package testing

import (
	"github.com/linkeunid/ligo"
	"github.com/linkeunid/ligo/internal/di"
)

// NewTestApp creates an app with the given modules, runs it, and returns it.
// The app is configured with no router and no logger for testing.
func NewTestApp(modules ...ligo.Module) *ligo.App {
	app := ligo.New()
	app.Register(modules...)
	_ = app.Run() // test helper — caller doesn't care about Run's error
	return app
}

// NewTestContainer creates a fresh container for unit testing.
func NewTestContainer() *di.Container {
	return di.New()
}

// NewTestAppWithOverrides creates an app with the given modules and overrides.
// Overrides replace providers before running.
func NewTestAppWithOverrides(modules []ligo.Module, overrides ...ligo.Provider) *ligo.App {
	app := ligo.New()
	app.Provide(overrides...)
	app.Register(modules...)
	_ = app.Run() // test helper — caller doesn't care about Run's error
	return app
}
