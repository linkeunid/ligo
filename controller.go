package ligo

// Controller defines how HTTP routes are registered for a module.
type Controller interface {
	Routes(r Router)
}
