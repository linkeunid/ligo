package errors

import (
	stderrors "errors"
	"strings"
	"testing"
)

func TestChainableError_ErrorIncludesRequiredBy(t *testing.T) {
	e := &ChainableError{Type: "Foo", RequiredBy: "Bar"}
	if !strings.Contains(e.Error(), "required by Bar") {
		t.Errorf("missing required-by, got %q", e.Error())
	}
}

func TestChainableError_ErrorOmitsRequiredByWhenEmpty(t *testing.T) {
	e := &ChainableError{Type: "Foo"}
	if strings.Contains(e.Error(), "required by") {
		t.Errorf("should omit required-by, got %q", e.Error())
	}
}

func TestChainableError_ErrorWithCause(t *testing.T) {
	inner := stderrors.New("boom")
	e := &ChainableError{Type: "Foo", Cause: inner}
	if !strings.Contains(e.Error(), "boom") {
		t.Errorf("missing cause, got %q", e.Error())
	}
}

func TestChainableError_UnwrapTraversesChain(t *testing.T) {
	inner := stderrors.New("root")
	e := NewChainableError("Foo", inner).WithRequiredBy("Bar")
	if !stderrors.Is(e, inner) {
		t.Error("errors.Is did not traverse Unwrap chain")
	}
	if e.RequiredBy != "Bar" {
		t.Errorf("WithRequiredBy did not set field, got %q", e.RequiredBy)
	}
}

func TestFormatChain_CyclicChainTerminates(t *testing.T) {
	// Build a cyclic chain: A -> B -> A -> B ...
	a := &ChainableError{Type: "A", RequiredBy: ""}
	b := &ChainableError{Type: "B", RequiredBy: "A"}
	a.Cause = b
	b.Cause = a

	// Must not stack-overflow. Result must be bounded.
	out := FormatChain(a.Type, a.RequiredBy, a.Cause, "")
	if !strings.Contains(out, "<truncated>") {
		t.Errorf("expected <truncated> marker in cyclic output, got: %q", out)
	}
}

func TestFormatChain_DeepChainTruncates(t *testing.T) {
	// Build a chain 100 levels deep.
	var head *ChainableError
	for range 100 {
		head = &ChainableError{Type: "T", Cause: head}
	}

	out := FormatChain(head.Type, "", head.Cause, "")
	if !strings.Contains(out, "<truncated>") {
		t.Errorf("expected <truncated> in deep-chain output")
	}
	// And reasonable line count — at most maxFormatChainDepth+1 levels worth.
	lines := strings.Count(out, "\n")
	if lines > maxFormatChainDepth+5 {
		t.Errorf("output has %d lines, expected <= %d", lines, maxFormatChainDepth+5)
	}
}

func TestFormatChain_ShortChainStillWorks(t *testing.T) {
	inner := &ChainableError{Type: "Inner", RequiredBy: "Outer"}
	out := FormatChain("Outer", "", inner, "")
	if !strings.Contains(out, "Outer") || !strings.Contains(out, "Inner") {
		t.Errorf("expected Outer + Inner in output, got: %q", out)
	}
	if strings.Contains(out, "<truncated>") {
		t.Errorf("unexpected truncation on short chain: %q", out)
	}
}
