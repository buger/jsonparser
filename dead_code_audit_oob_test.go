package jsonparser

import (
	"testing"
)

// Test that ObjectEach doesn't panic with out-of-bounds access
// after removing the `offset < len(data)` loop guard.

// Verifies: SYS-REQ-007 [boundary]
func TestObjectEach_OOB_TruncatedAfterComma(t *testing.T) {
	// {"a":1, — truncated right after comma, no more data
	// After parsing "a":1, finds comma at step 4, increments offset past comma.
	// Then step "skip to next token after comma" calls nextToken on remaining data.
	// If remaining is empty → nextToken returns -1 → returns MalformedArrayError.
	// No panic.
	err := ObjectEach([]byte(`{"a":1,`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for truncated object after comma")
	}
	t.Logf("Correctly got error: %v", err)
}

// Verifies: SYS-REQ-007 [boundary]
func TestObjectEach_OOB_TruncatedAfterColon(t *testing.T) {
	// {"a": — truncated after colon
	err := ObjectEach([]byte(`{"a":`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for truncated object after colon")
	}
	t.Logf("Correctly got error: %v", err)
}

// Verifies: SYS-REQ-007 [boundary]
func TestObjectEach_OOB_TruncatedAfterKey(t *testing.T) {
	// {"a" — truncated after key string
	err := ObjectEach([]byte(`{"a"`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for truncated object after key")
	}
	t.Logf("Correctly got error: %v", err)
}

// Verifies: SYS-REQ-007 [boundary]
func TestObjectEach_OOB_TruncatedMidKey(t *testing.T) {
	// {"a — unterminated string
	err := ObjectEach([]byte(`{"a`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for unterminated key string")
	}
	t.Logf("Correctly got error: %v", err)
}

// Verifies: SYS-REQ-007 [boundary]
func TestObjectEach_OOB_JustOpenBrace(t *testing.T) {
	// { — only opening brace, then nothing
	err := ObjectEach([]byte(`{`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for just opening brace")
	}
	t.Logf("Correctly got error: %v", err)
}

// Verifies: SYS-REQ-007 [boundary]
func TestObjectEach_OOB_BraceAndWhitespace(t *testing.T) {
	// {    — opening brace then only whitespace
	err := ObjectEach([]byte(`{   `), func(key []byte, value []byte, dataType ValueType, offset int) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for brace+whitespace")
	}
	t.Logf("Correctly got error: %v", err)
}

// ArrayEach infinite loop guard: verify o==0 catches all no-progress cases
// Verifies: SYS-REQ-006 [boundary]
func TestArrayEach_OOB_MalformedElements(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"bare_comma", `[,]`},
		{"double_comma", `[1,,2]`},
		{"just_bracket", `[`},
		{"bracket_space", `[   `},
		{"unclosed_string", `["abc`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := 0
			_, err := ArrayEach([]byte(tt.json), func(value []byte, dataType ValueType, offset int, err error) {
				count++
				if count > 100 {
					t.Fatal("possible infinite loop detected")
				}
			})
			// We don't care whether it errors; we care that it terminates
			_ = err
			t.Logf("Terminated with count=%d, err=%v", count, err)
		})
	}
}
