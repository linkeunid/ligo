package errors

import (
	stderrors "errors"
	"fmt"
	"strings"
)

// ChainableError represents an error with a dependency chain.
// This is useful for tracking the full path of errors through a call stack.
type ChainableError struct {
	Type       string
	RequiredBy string
	Cause      error
}

// Error returns a formatted error message showing the full chain.
func (e *ChainableError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "ligo: cannot resolve %s", e.Type)
	if e.RequiredBy != "" {
		fmt.Fprintf(&b, " (required by %s)", e.RequiredBy)
	}
	if e.Cause != nil {
		fmt.Fprintf(&b, ": %v", e.Cause)
	}
	return b.String()
}

// Unwrap returns the underlying cause for use with errors.Is/As.
func (e *ChainableError) Unwrap() error {
	return e.Cause
}

// NewChainableError creates a new chainable error with the given type and cause.
func NewChainableError(typ string, cause error) *ChainableError {
	return &ChainableError{
		Type:  typ,
		Cause: cause,
	}
}

// WithRequiredBy adds the required-by context to a chainable error.
func (e *ChainableError) WithRequiredBy(requiredBy string) *ChainableError {
	e.RequiredBy = requiredBy
	return e
}

// maxFormatChainDepth bounds FormatChain recursion. Cyclic error chains
// (legal under errors.Unwrap — A.Unwrap()=B, B.Unwrap()=A) would otherwise
// stack-overflow the formatter; pathological deep chains would DoS it.
const maxFormatChainDepth = 32

// FormatChain formats a dependency chain for error messages.
// This is useful for displaying the full chain of missing dependencies.
// Depth is capped at maxFormatChainDepth to prevent stack overflow on
// cyclic chains; subsequent links are replaced with "<truncated>".
func FormatChain(dep, requiredBy string, cause error, indent string) string {
	return formatChainDepth(dep, requiredBy, cause, indent, 0)
}

func formatChainDepth(dep, requiredBy string, cause error, indent string, depth int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s%s", indent, dep)
	if requiredBy != "" {
		fmt.Fprintf(&b, "  <- required by %s", requiredBy)
	}
	b.WriteString("\n")

	if depth >= maxFormatChainDepth {
		fmt.Fprintf(&b, "%s  <truncated>", indent)
		return b.String()
	}

	// Continue unwrapping if there's a chainable cause
	var chainable *ChainableError
	if stderrors.As(cause, &chainable) {
		b.WriteString(formatChainDepth(chainable.Type, chainable.RequiredBy, chainable.Cause, indent+"  ", depth+1))
	} else if cause != nil {
		fmt.Fprintf(&b, "%s  %s", indent, cause.Error())
	}

	return b.String()
}
