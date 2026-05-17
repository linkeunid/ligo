package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/linkeunid/ligo/internal/http"
)

func TestIsAddrInUse_FrameworkSentinel(t *testing.T) {
	err := fmt.Errorf("listen: %w", ErrAddrInUse)
	if !IsAddrInUse(err) {
		t.Error("expected sentinel-wrapped error to be recognized")
	}
}

func TestIsAddrInUse_RawSyscall(t *testing.T) {
	opErr := &net.OpError{
		Op:  "listen",
		Net: "tcp",
		Err: &os.SyscallError{Syscall: "bind", Err: syscall.EADDRINUSE},
	}
	if !IsAddrInUse(opErr) {
		t.Error("expected raw syscall.EADDRINUSE to be recognized")
	}
}

func TestIsAddrInUse_UnrelatedError(t *testing.T) {
	if IsAddrInUse(errors.New("nope")) {
		t.Error("unrelated error should not match")
	}
	if IsAddrInUse(nil) {
		t.Error("nil should not match")
	}
}

// fakeGracefulRouter is enough Router for serveWithGracefulShutdownAt.
type fakeGracefulRouter struct {
	serveBlock chan struct{}
	shutdown   func(context.Context) error
}

func (f *fakeGracefulRouter) Group(prefix string) http.Router                { return f }
func (f *fakeGracefulRouter) Use(mw ...http.Middleware)                      {}
func (f *fakeGracefulRouter) Handle(method, path string, h http.HandlerFunc) {}
func (f *fakeGracefulRouter) Serve(addr string) error {
	<-f.serveBlock
	return nil
}

func (f *fakeGracefulRouter) Shutdown(ctx context.Context) error {
	close(f.serveBlock)
	if f.shutdown != nil {
		return f.shutdown(ctx)
	}
	return nil
}

func TestServe_JoinsShutdownErrors(t *testing.T) {
	router := &fakeGracefulRouter{
		serveBlock: make(chan struct{}),
		shutdown:   func(context.Context) error { return errors.New("router boom") },
	}

	appErr := errors.New("app boom")
	hookErr := errors.New("hook boom")

	opts := ServeOptions{
		Router:          router,
		Addr:            ":0",
		GracefulTimeout: time.Second,
		AppShutdown:     func() error { return appErr },
		OnStop:          []func(any) error{func(any) error { return hookErr }},
	}

	done := make(chan error, 1)
	go func() { done <- serveWithGracefulShutdownAt(":0", opts) }()

	// Give the goroutine time to start listening on the signal channel.
	time.Sleep(20 * time.Millisecond)
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("find self: %v", err)
	}
	if err := p.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal: %v", err)
	}

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected joined error, got nil")
		}
		if !errors.Is(err, appErr) {
			t.Errorf("missing app err in chain: %v", err)
		}
		if !errors.Is(err, hookErr) {
			t.Errorf("missing hook err in chain: %v", err)
		}
		if !contains(err.Error(), "router boom") {
			t.Errorf("missing router err in chain: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not return within 3s")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
