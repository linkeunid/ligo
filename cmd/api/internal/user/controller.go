package user

import (
	"github.com/linkeunid/ligo"
)

// UserController handles HTTP routes for users
type UserController struct {
	svc *UserService
}

// NewUserController creates a new user controller with injected service
// This function signature allows Ligo to auto-inject dependencies
func NewUserController(svc *UserService) ligo.Controller {
	return &UserController{svc: svc}
}

// Routes registers all routes for the user module
func (c *UserController) Routes(r ligo.Router) {
	g := r.Group("/users")
	g.Handle("GET", "/", c.List)
	g.Handle("GET", "/:id", c.Get)
}

// Get handles GET /users/:id
func (c *UserController) Get(ctx ligo.Context) error {
	id := ctx.Param("id")
	user := c.svc.GetUser(id)
	if user == "" {
		return ctx.JSON(404, map[string]string{"error": "user not found"})
	}
	return ctx.JSON(200, map[string]string{"id": id, "name": user})
}

// List handles GET /users
func (c *UserController) List(ctx ligo.Context) error {
	users := c.svc.List()
	return ctx.JSON(200, users)
}