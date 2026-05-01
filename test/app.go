package test

import (
	"github.com/linkeunid/ligo"
	"github.com/linkeunid/ligo/internal/container"
)

// NewTestApp creates an app with the given modules, runs it, and returns it.
// The app is configured with no router and no logger for testing.
func NewTestApp(modules ...ligo.Module) *ligo.App {
	app := ligo.New()
	app.Register(modules...)
	app.Run()
	return app
}

// NewTestContainer creates a fresh container for unit testing.
func NewTestContainer() *container.Container {
	return container.New()
}
