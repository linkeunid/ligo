package main

import (
	"fmt"
	"net/http"

	"github.com/linkeunid/ligo"
	"github.com/linkeunid/ligo/adapters/echo"
	"github.com/linkeunid/ligo/cmd/api/internal/middleware"
	"github.com/linkeunid/ligo/cmd/api/internal/user"
)

func main() {
	router := echo.NewAdapter()
	app := ligo.New(
		ligo.WithRouter(router),
		ligo.WithAddr(":8080"),
		ligo.WithMiddleware(middleware.LoggingMiddleware),
	)

	// Register modules - controllers get auto-injected with dependencies
	app.Register(user.Module())

	fmt.Println("Starting server on :8080")
	fmt.Println("Try:")
	fmt.Println("  curl http://localhost:8080/users/")
	fmt.Println("  curl http://localhost:8080/users/1")

	if err := app.Run(); err != nil {
		if err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}
}