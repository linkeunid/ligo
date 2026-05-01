package ligo

import (
	"testing"
)

func TestErrAppAlreadyStarted(t *testing.T) {
	err := &ErrAppAlreadyStarted{}
	if err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestErrMissingDependency(t *testing.T) {
	err := &ErrMissingDependency{
		Type:       "*test.Service",
		RequiredBy: "*test.Controller",
	}
	if err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestErrCircularDependency(t *testing.T) {
	err := &ErrCircularDependency{
		Chain: []string{"UserService", "UserRepo", "UserService"},
	}
	if err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestErrDuplicateProvider(t *testing.T) {
	err := &ErrDuplicateProvider{Type: "*test.Service"}
	if err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestDIError(t *testing.T) {
	err := &DIError{
		Type:       "*test.Service",
		RequiredBy: "*test.Controller",
		Cause:      &ErrMissingDependency{Type: "test", RequiredBy: "test"},
	}
	if err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
}
