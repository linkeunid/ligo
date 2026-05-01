package user

import (
	"github.com/linkeunid/ligo"
)

// Module returns the user module with all providers and controllers
// Controllers are auto-injected with dependencies after Run()
func Module() ligo.Module {
	return ligo.NewModule("user",
		ligo.Providers(
			// Repository - singleton
			ligo.Factory[*UserRepo](NewUserRepo),
			// Service - singleton with auto-injected repo
			ligo.Factory[*UserService](NewUserService),
		),
		// Controller constructor - dependencies auto-injected by framework
		ligo.Controllers(func(svc *UserService) ligo.Controller {
			return NewUserController(svc)
		}),
	)
}