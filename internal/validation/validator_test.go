package validation

import (
	"errors"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

type emailRequired struct {
	Email string `validate:"required,email"`
	Name  string `validate:"required,min=2"`
}

// MAJOR-020: empty Email used to surface both "required" AND "must be valid
// email" because the 2nd pass substituted "x" — which itself fails email.
// We now suppress format-tag errors for fields that already failed required.
func TestValidateExhaustive_SuppressesFormatTagsForRequired(t *testing.T) {
	v := validator.New()
	err := ValidateExhaustive(v, &emailRequired{Email: "", Name: ""})
	if err == nil {
		t.Fatal("expected validation error")
	}

	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}

	for _, fe := range verrs {
		// No "email" tag error on Email — Email is required, the contradictory
		// format error must be suppressed.
		if fe.Field() == "Email" && fe.Tag() == "email" {
			t.Errorf("Email had email-tag error in addition to required: %+v", fe)
		}
	}

	msg := err.Error()
	if !strings.Contains(msg, "Email") {
		t.Errorf("expected Email-required message, got %q", msg)
	}
	// Name has min=2 which is NOT format-sensitive — second-pass should
	// still surface it.
	hasNameMin := false
	for _, fe := range verrs {
		if fe.Field() == "Name" && fe.Tag() == "min" {
			hasNameMin = true
		}
	}
	if !hasNameMin {
		t.Errorf("expected Name min=2 to surface in 2nd pass (not format-sensitive)")
	}
}

type minOnly struct {
	Comment string `validate:"required,min=10"`
}

func TestValidateExhaustive_MinSurfacesForRequiredEmpty(t *testing.T) {
	v := validator.New()
	err := ValidateExhaustive(v, &minOnly{})
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}

	tags := map[string]bool{}
	for _, fe := range verrs {
		tags[fe.Tag()] = true
	}
	if !tags["required"] {
		t.Errorf("expected 'required' tag, got %v", tags)
	}
	if !tags["min"] {
		t.Errorf("expected 'min' tag to surface (non-format), got %v", tags)
	}
}
