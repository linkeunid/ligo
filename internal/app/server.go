package app

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/linkeunid/ligo/internal/core/logger"
	"github.com/linkeunid/ligo/internal/http"
)

const maxPortRetries = 10

// IncrementPort increments the port number in an address string.
func IncrementPort(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return "", err
	}
	portNum++
	return net.JoinHostPort(host, strconv.Itoa(portNum)), nil
}

// IsAddrInUse checks if the error is due to address already in use.
func IsAddrInUse(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "address already in use") ||
		strings.Contains(errStr, "bind: address already in use") ||
		strings.Contains(errStr, "EADDRINUSE")
}

// ServeOptions holds options for serving.
type ServeOptions struct {
	Router          http.Router
	Logger          logger.Logger
	Addr            string
	AutoPort        bool
	GracefulTimeout time.Duration
	ModuleHooks     *ModuleHooks
	OnStop          []func(any) error
}

// ServeWithRetry attempts to start the server, incrementing the port on address-in-use errors.
func ServeWithRetry(opts ServeOptions) error {
	if opts.AutoPort {
		return retryWithPortIncrement(opts.Addr, func(addr string) error {
			return serveWithGracefulShutdownAt(addr, opts)
		}, opts.Logger)
	}
	return serveWithGracefulShutdownAt(opts.Addr, opts)
}

func retryWithPortIncrement(addr string, serveFunc func(string) error, log logger.Logger) error {
	currentAddr := addr
	var lastErr error

	for attempt := 0; attempt <= maxPortRetries; attempt++ {
		if attempt > 0 {
			if log != nil {
				log.Warn("Port already in use, trying next port",
					logger.Field{Key: "attempt", Value: attempt},
					logger.Field{Key: "addr", Value: currentAddr},
					logger.Field{Key: "last_error", Value: lastErr.Error()},
				)
			}
		}

		err := serveFunc(currentAddr)
		if err == nil {
			return nil
		}

		if !IsAddrInUse(err) {
			return err
		}

		lastErr = err
		nextAddr, err := IncrementPort(currentAddr)
		if err != nil {
			return fmt.Errorf("failed to increment port: %w", err)
		}
		currentAddr = nextAddr
	}

	return fmt.Errorf("failed to start server after %d attempts: %w", maxPortRetries, lastErr)
}

func serveWithGracefulShutdownAt(addr string, opts ServeOptions) error {
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(shutdownChan)

	errChan := make(chan error, 1)

	go func() {
		errChan <- opts.Router.Serve(addr)
	}()

	select {
	case <-shutdownChan:
		if opts.Logger != nil {
			opts.Logger.Info("Shutting down gracefully...", logger.Field{Key: "context", Value: logger.ContextLifecycle})
		}

		ctx, cancel := context.WithTimeout(context.Background(), opts.GracefulTimeout)
		defer cancel()

		if opts.ModuleHooks != nil {
			for i := len(opts.ModuleHooks.OnDestroy) - 1; i >= 0; i-- {
				for j := len(opts.ModuleHooks.OnDestroy[i]) - 1; j >= 0; j-- {
					if err := opts.ModuleHooks.OnDestroy[i][j](); err != nil {
						if opts.Logger != nil {
							opts.Logger.Error("OnModuleDestroy hook failed", logger.Field{Key: "error", Value: err})
						}
					}
				}
			}
		}

		for _, hook := range opts.OnStop {
			if err := hook(ctx); err != nil {
				if opts.Logger != nil {
					opts.Logger.Error("OnStop hook failed", logger.Field{Key: "error", Value: err})
				}
			}
		}

		if gs, ok := opts.Router.(http.GracefulServer); ok {
			if err := gs.Shutdown(ctx); err != nil {
				return err
			}
		}
		return nil
	case err := <-errChan:
		return err
	}
}
