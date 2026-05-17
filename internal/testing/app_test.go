package testing

import "testing"

func TestNewTestContainer_ReturnsNonNil(t *testing.T) {
	c := NewTestContainer()
	if c == nil {
		t.Fatal("nil container")
	}
}

// NOTE: NewTestApp and NewTestAppWithOverrides call app.Run() synchronously,
// which blocks on a SIGINT/SIGTERM signal. They are intended for tests that
// drive shutdown via signal themselves; calling them directly from a unit
// test would deadlock until something else signals the process. We exercise
// the full lifecycle path via integration_test.go instead.
