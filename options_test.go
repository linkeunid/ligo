package ligo

import (
	"testing"
)

func TestWithAddr(t *testing.T) {
	opts := defaultOptions()
	WithAddr(":3000")(&opts)
	if opts.addr != ":3000" {
		t.Fatalf("expected addr :3000, got %s", opts.addr)
	}
}

func TestWithDebug(t *testing.T) {
	opts := defaultOptions()
	WithDebug(true)(&opts)
	if !opts.debug {
		t.Fatal("expected debug to be true")
	}
}
