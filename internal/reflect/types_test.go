package reflectutil

import (
	"reflect"
	"testing"
)

type Widget struct{}

func TestExtractTypeName_Nil(t *testing.T) {
	if got := ExtractTypeName(nil); got != "unknown" {
		t.Errorf("nil = %q, want unknown", got)
	}
}

func TestExtractTypeName_Struct(t *testing.T) {
	if got := ExtractTypeName(Widget{}); got != "Widget" {
		t.Errorf("Widget = %q", got)
	}
}

func TestExtractTypeName_PointerToStruct(t *testing.T) {
	if got := ExtractTypeName(&Widget{}); got != "Widget" {
		t.Errorf("&Widget = %q", got)
	}
}

func TestExtractTypeName_FuncReturningPointer(t *testing.T) {
	fn := func() *Widget { return nil }
	if got := ExtractTypeName(fn); got != "Widget" {
		t.Errorf("func()*Widget = %q", got)
	}
}

func TestExtractTypeName_FuncReturningValue(t *testing.T) {
	fn := func() Widget { return Widget{} }
	if got := ExtractTypeName(fn); got != "Widget" {
		t.Errorf("func()Widget = %q", got)
	}
}

func TestIsPointerType(t *testing.T) {
	cases := []struct {
		in   any
		want bool
	}{
		{nil, false},
		{Widget{}, false},
		{&Widget{}, true},
		{42, false},
	}
	for _, c := range cases {
		if got := IsPointerType(c.in); got != c.want {
			t.Errorf("IsPointerType(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestGetElementType_Pointer(t *testing.T) {
	got := GetElementType(&Widget{})
	if got != reflect.TypeFor[Widget]() {
		t.Errorf("element type = %v", got)
	}
}

func TestGetElementType_NonPointer(t *testing.T) {
	got := GetElementType(Widget{})
	if got != reflect.TypeFor[Widget]() {
		t.Errorf("element type = %v", got)
	}
}

func TestGetElementType_Nil(t *testing.T) {
	if got := GetElementType(nil); got != nil {
		t.Errorf("nil element type = %v", got)
	}
}
