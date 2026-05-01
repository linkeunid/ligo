package main

import (
	"fmt"
	"net/http"

	"github.com/linkeunid/ligo"
	"github.com/linkeunid/ligo/adapters/echo"
)

// Simple example showing basic Ligo usage

func main() {
	router := echo.NewAdapter()
	app := ligo.New(
		ligo.WithRouter(router),
		ligo.WithAddr(":8080"),
		ligo.OnStart(func(ctx any) error {
			fmt.Println("App starting...")
			return nil
		}),
		ligo.OnStop(func(ctx any) error {
			fmt.Println("App stopping...")
			return nil
		}),
	)

	// Register a simple inline module
	app.Register(
		ligo.NewModule("hello",
			ligo.Providers(
				ligo.Value("Hello, World!"),
			),
			ligo.Controllers(func(msg string) ligo.Controller {
				return &helloController{msg: msg}
			}),
		),
	)

	fmt.Println("Starting server on :8080")
	fmt.Println("Try: curl http://localhost:8080")

	if err := app.Run(); err != nil {
		if err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}
}

type helloController struct {
	msg string
}

func (c *helloController) Routes(r ligo.Router) {
	r.Handle("GET", "/", func(ctx ligo.Context) error {
		return ctx.String(200, c.msg)
	})
}
