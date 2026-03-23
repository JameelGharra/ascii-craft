package utils

import (
	"strings"
	"testing"
)

func TestAssert_NoPanicOnTrue(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Assert panicked unexpectedly on a true condition: %v", r)
		}
	}()

	Assert(true, "This should not panic")
}

// must panic
func TestAssert_PanicOnFalse(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("Assert did not panic on a false condition")
		}

		errStr, ok := r.(string)
		if !ok || !strings.Contains(errStr, "assertion failed") {
			t.Errorf("Expected panic message containing 'assertion failed', got: %v", r)
		}
	}()
	Assert(false, "This should panic", 123, "test_arg")
}

func TestStringify(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"Nil", nil, "nil"},
		{"String", "hello", "hello"},
		{"Byte Slice", []byte("world"), "world"},
		{"Integer", 42, "42"},
		{"Map (JSON)", map[string]int{"a": 1}, `{"a":1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringify(tt.input)
			if result != tt.expected {
				t.Errorf("stringify(%v) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}
