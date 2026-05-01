package ligo

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/linkeunid/ligo/internal/core/logger"
	"github.com/linkeunid/ligo/internal/http"
)

const (
	maxPortRetries = 10
)

// incrementPort increments the port number in an address string.
func incrementPort(addr string) (string, error) {
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

// isAddrInUse checks if the error is due to address already in use.
func isAddrInUse(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "address already in use") ||
		strings.Contains(errStr, "bind: address already in use") ||
		strings.Contains(errStr, "EADDRINUSE")
}

// serveWithRetry attempts to start the server, incrementing the port on address-in-use errors.
func (a *App) serveWithRetry(addr string) error {
	return a.retryWithPortIncrement(addr, func(addr string) error {
		return a.opts.router.Serve(addr)
	})
}

// retryWithPortIncrement retries serving with port increment on address-in-use errors.
func (a *App) retryWithPortIncrement(addr string, serveFunc func(string) error) error {
	currentAddr := addr
	var lastErr error

	for attempt := 0; attempt <= maxPortRetries; attempt++ {
		if attempt > 0 {
			a.opts.logger.Warn("Port already in use, trying next port",
				logger.Field{Key: "attempt", Value: attempt},
				logger.Field{Key: "addr", Value: currentAddr},
				logger.Field{Key: "last_error", Value: lastErr.Error()},
			)
		}

		err := serveFunc(currentAddr)
		if err == nil {
			return nil
		}

		if !isAddrInUse(err) {
			return err
		}

		// Port is in use, try next port
		lastErr = err
		nextAddr, err := incrementPort(currentAddr)
		if err != nil {
			return fmt.Errorf("failed to increment port: %w", err)
		}
		currentAddr = nextAddr
	}

	return fmt.Errorf("failed to start server after %d attempts: %w", maxPortRetries, lastErr)
}

// runWithGracefulShutdown runs the server with graceful shutdown on SIGINT/SIGTERM.
func (a *App) runWithGracefulShutdown() error {
	if a.opts.autoPort {
		return a.runWithGracefulShutdownAndRetry()
	}
	return a.serveWithGracefulShutdownAt(a.opts.addr)
}

// runWithGracefulShutdownAndRetry runs the server with graceful shutdown and port retry logic.
func (a *App) runWithGracefulShutdownAndRetry() error {
	return a.retryWithPortIncrement(a.opts.addr, a.serveWithGracefulShutdownAt)
}

// serveWithGracefulShutdownAt runs the server at a specific address with graceful shutdown.
func (a *App) serveWithGracefulShutdownAt(addr string) error {
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(shutdownChan)

	errChan := make(chan error, 1)

	go func() {
		errChan <- a.opts.router.Serve(addr)
	}()

	select {
	case <-shutdownChan:
		a.opts.logger.Info("Shutting down gracefully...", logger.Field{Key: "context", Value: logger.ContextLifecycle})

		ctx, cancel := context.WithTimeout(context.Background(), a.opts.gracefulTimeout)
		defer cancel()

		// Execute OnModuleDestroy hooks (reverse order)
		for i := len(a.moduleHooks.onDestroy) - 1; i >= 0; i-- {
			for j := len(a.moduleHooks.onDestroy[i]) - 1; j >= 0; j-- {
				if err := a.moduleHooks.onDestroy[i][j](); err != nil {
					a.opts.logger.Error("OnModuleDestroy hook failed", logger.Field{Key: "error", Value: err})
				}
			}
		}

		for _, hook := range a.opts.onStop {
			if err := hook(ctx); err != nil {
				a.opts.logger.Error("OnStop hook failed", logger.Field{Key: "error", Value: err})
			}
		}

		if gs, ok := a.opts.router.(http.GracefulServer); ok {
			if err := gs.Shutdown(ctx); err != nil {
				return err
			}
		}
		return nil
	case err := <-errChan:
		return err
	}
}
