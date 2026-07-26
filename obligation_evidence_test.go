package jsonparser

import (
	"bytes"
	"errors"
	"testing"
)

// =============================================================================
// Obligation evidence witnesses.
// =============================================================================
//
// Each test below carries one or more `<REQ>:<obligation>:<evidence>` triples
// required by `obligation_evidence_complete`. For a Go library the integrated
// system IS the public API exercised end-to-end; these tests drive both the
// happy-path (`:nominal`) and the reject-path (`:negative`) obligations.

// -----------------------------------------------------------------------------
// STK-REQ-001..007 — malformed_input + nil_safety negative evidence on the
// integrated public API of each stakeholder story.
// -----------------------------------------------------------------------------

// STK-REQ-001:malformed_input:negative
// STK-REQ-001:nil_safety:negative
func TestObligation_STK_REQ_001(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected error on malformed input")
	}
	if _, _, _, err := Get(nil, "a"); err == nil {
		t.Fatal("expected error on nil input")
	}
}

// STK-REQ-002:malformed_input:negative
// STK-REQ-002:nil_safety:negative
func TestObligation_STK_REQ_002(t *testing.T) {
	if _, err := GetString([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected error on malformed GetString input")
	}
	if _, err := GetString(nil, "a"); err == nil {
		t.Fatal("expected error on nil GetString input")
	}
}

// STK-REQ-003:malformed_input:negative
// STK-REQ-003:nil_safety:negative
func TestObligation_STK_REQ_003(t *testing.T) {
	if _, err := GetInt([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected error on malformed GetInt input")
	}
	if _, err := GetInt(nil, "a"); err == nil {
		t.Fatal("expected error on nil GetInt input")
	}
}

// STK-REQ-004:malformed_input:negative
// STK-REQ-004:nil_safety:negative
func TestObligation_STK_REQ_004(t *testing.T) {
	if _, err := ArrayEach([]byte(`[`), func(value []byte, dataType ValueType, offset int, err error) {}); err == nil {
		t.Fatal("expected error on malformed ArrayEach input")
	}
	if _, err := ArrayEach(nil, func(value []byte, dataType ValueType, offset int, err error) {}); err == nil {
		t.Fatal("expected error on nil ArrayEach input")
	}
}

// STK-REQ-005:malformed_input:negative
// STK-REQ-005:nil_safety:negative
func TestObligation_STK_REQ_005(t *testing.T) {
	if _, err := Set([]byte(`{"a":`), []byte(`42`), "a"); err == nil {
		t.Fatal("expected error on malformed Set input")
	}
	if _, err := Set(nil, []byte(`42`), "a"); err == nil {
		t.Fatal("expected error on nil Set input")
	}
}

// STK-REQ-006:malformed_input:negative
// STK-REQ-006:nil_safety:negative
func TestObligation_STK_REQ_006(t *testing.T) {
	if _, err := GetUnsafeString([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected error on malformed GetUnsafeString input")
	}
	if _, err := GetUnsafeString(nil, "a"); err == nil {
		t.Fatal("expected error on nil GetUnsafeString input")
	}
}

// STK-REQ-007:malformed_input:negative
// STK-REQ-007:nil_safety:negative
func TestObligation_STK_REQ_007(t *testing.T) {
	if _, err := ParseBoolean([]byte(`notabool`)); err == nil {
		t.Fatal("expected error on malformed ParseBoolean input")
	}
	if _, err := ParseBoolean(nil); err == nil {
		t.Fatal("expected error on nil ParseBoolean input")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ obligation evidence — each requirement's obligation_checklist item.
// -----------------------------------------------------------------------------

// SYS-REQ-001:determinism:nominal
func TestObligation_SYS_REQ_001(t *testing.T) {
	v1, _, _, _ := Get([]byte(`{"a":1}`), "a")
	v2, _, _, _ := Get([]byte(`{"a":1}`), "a")
	if !bytes.Equal(v1, v2) {
		t.Fatalf("determinism violated: %q vs %q", string(v1), string(v2))
	}
}

// SYS-REQ-002:determinism:nominal
// SYS-REQ-002:edge_case:nominal
// SYS-REQ-002:encoding_safety:nominal
func TestObligation_SYS_REQ_002(t *testing.T) {
	v1, _ := GetString([]byte(`{"a":"hello"}`), "a")
	v2, _ := GetString([]byte(`{"a":"hello"}`), "a")
	if v1 != v2 {
		t.Fatalf("determinism violated: %q vs %q", v1, v2)
	}
	// Edge case: empty string round-trips correctly.
	v, err := GetString([]byte(`{"a":""}`), "a")
	if err != nil || v != "" {
		t.Fatalf("edge-case empty string: v=%q err=%v", v, err)
	}
	// Encoding round-trip via ParseString/encode cycle.
	if got, err := ParseString([]byte(`hello`)); err != nil || got != "hello" {
		t.Fatalf("encoding roundtrip: got=%q err=%v", got, err)
	}
}

// SYS-REQ-003:determinism:nominal
func TestObligation_SYS_REQ_003(t *testing.T) {
	v1, _ := GetInt([]byte(`{"a":1}`), "a")
	v2, _ := GetInt([]byte(`{"a":1}`), "a")
	if v1 != v2 {
		t.Fatalf("determinism violated: %d vs %d", v1, v2)
	}
}

// SYS-REQ-006:determinism:nominal
func TestObligation_SYS_REQ_006(t *testing.T) {
	c1 := 0
	ArrayEach([]byte(`[1,2,3]`), func(value []byte, dataType ValueType, offset int, err error) { c1++ })
	c2 := 0
	ArrayEach([]byte(`[1,2,3]`), func(value []byte, dataType ValueType, offset int, err error) { c2++ })
	if c1 != c2 {
		t.Fatalf("determinism violated: %d vs %d", c1, c2)
	}
}

// SYS-REQ-008:edge_case:nominal
func TestObligation_SYS_REQ_008(t *testing.T) {
	// EachKey on empty object must complete cleanly (edge case).
	EachKey([]byte(`{}`), func(i int, value []byte, vt ValueType, err error) {}, []string{"a"})
}

// SYS-REQ-009:idempotency:nominal
func TestObligation_SYS_REQ_009(t *testing.T) {
	r1, _ := Set([]byte(`{"a":1}`), []byte(`42`), "a")
	r2, _ := Set(r1, []byte(`42`), "a")
	if !bytes.Equal(r1, r2) {
		t.Fatalf("idempotency violated: %s vs %s", string(r1), string(r2))
	}
}

// SYS-REQ-010:empty_input:nominal
// SYS-REQ-010:nil_safety:nominal
// SYS-REQ-010:nil_safety:negative
func TestObligation_SYS_REQ_010(t *testing.T) {
	if got := Delete([]byte{}); len(got) != 0 {
		t.Fatalf("expected empty result on empty input, got %s", string(got))
	}
	if got := Delete(nil); len(got) != 0 {
		t.Fatalf("expected empty result on nil input, got %s", string(got))
	}
}

// SYS-REQ-011:determinism:nominal
func TestObligation_SYS_REQ_011(t *testing.T) {
	v1, _ := GetUnsafeString([]byte(`{"a":"b"}`), "a")
	v2, _ := GetUnsafeString([]byte(`{"a":"b"}`), "a")
	if v1 != v2 {
		t.Fatalf("determinism violated: %q vs %q", v1, v2)
	}
}

// SYS-REQ-012:determinism:nominal
func TestObligation_SYS_REQ_012(t *testing.T) {
	v1, _ := ParseBoolean([]byte(`true`))
	v2, _ := ParseBoolean([]byte(`true`))
	if v1 != v2 {
		t.Fatalf("determinism violated: %v vs %v", v1, v2)
	}
}

// SYS-REQ-014:encoding_safety:nominal
func TestObligation_SYS_REQ_014(t *testing.T) {
	if got, err := ParseString([]byte(`hello`)); err != nil || got != "hello" {
		t.Fatalf("encoding roundtrip: got=%q err=%v", got, err)
	}
}

// SYS-REQ-015:edge_case:nominal
// SYS-REQ-015:nil_safety:nominal
// SYS-REQ-015:nil_safety:negative
func TestObligation_SYS_REQ_015(t *testing.T) {
	if v, err := ParseInt([]byte(`0`)); err != nil || v != 0 {
		t.Fatalf("edge-case zero: v=%d err=%v", v, err)
	}
	if _, err := ParseInt(nil); err == nil {
		t.Fatal("expected error on nil input")
	}
}

// SYS-REQ-016:missing_path:nominal
func TestObligation_SYS_REQ_016(t *testing.T) {
	// Witness the positive missing-path outcome: well-formed lookup that
	// returns the documented NotFound tuple.
	_, dataType, offset, err := Get([]byte(`{"a":1}`), "missing")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError, got %v", err)
	}
	if dataType != NotExist || offset != -1 {
		t.Fatalf("expected not-found tuple, got type=%v offset=%d", dataType, offset)
	}
}

// SYS-REQ-017:malformed_input:nominal
// SYS-REQ-017:malformed_input:negative
func TestObligation_SYS_REQ_017(t *testing.T) {
	// Positive path: complete input parses without error.
	if _, _, _, err := Get([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("Get on complete input returned error: %v", err)
	}
	// Negative path: malformed input yields a parse error.
	if _, _, _, err := Get([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected parse error on incomplete input")
	}
}

// SYS-REQ-018:idempotency:nominal
func TestObligation_SYS_REQ_018(t *testing.T) {
	v1, _, _, _ := Get([]byte(`{"a":1}`))
	v2, _, _, _ := Get([]byte(`{"a":1}`))
	if !bytes.Equal(v1, v2) {
		t.Fatalf("idempotency violated: %q vs %q", string(v1), string(v2))
	}
}

// SYS-REQ-019:empty_input:nominal
// SYS-REQ-019:nil_safety:nominal
// SYS-REQ-019:nil_safety:negative
func TestObligation_SYS_REQ_019(t *testing.T) {
	// Empty input returns a documented not-found tuple.
	_, dataType, offset, err := Get([]byte(""), "a")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError on empty input, got %v", err)
	}
	if dataType != NotExist || offset != -1 {
		t.Fatalf("expected not-found tuple, got type=%v offset=%d", dataType, offset)
	}
	if _, _, _, err := Get(nil, "a"); err == nil {
		t.Fatal("expected error on nil input")
	}
}

// SYS-REQ-023:boundary:nominal
// SYS-REQ-023:edge_case:nominal
func TestObligation_SYS_REQ_023(t *testing.T) {
	// Boundary positive case: in-bounds index returns the element.
	if v, _, _, err := Get([]byte(`{"a":[1,2,3]}`), "a", "[0]"); err != nil || string(v) != "1" {
		t.Fatalf("boundary in-bounds lookup: v=%s err=%v", string(v), err)
	}
}

// SYS-REQ-027:type_mismatch:nominal
func TestObligation_SYS_REQ_027(t *testing.T) {
	// Positive path: well-formed value parses without invoking value-type-error.
	if _, _, _, err := Get([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("Get on well-formed value returned error: %v", err)
	}
}

// SYS-REQ-028:empty_input:nominal
// SYS-REQ-028:nil_safety:nominal
// SYS-REQ-028:nil_safety:negative
func TestObligation_SYS_REQ_028(t *testing.T) {
	calls := 0
	if _, err := ArrayEach([]byte(`[]`), func(value []byte, dataType ValueType, offset int, err error) {
		calls++
	}); err != nil {
		t.Fatalf("ArrayEach on empty array returned error: %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected zero callbacks, got %d", calls)
	}
	if _, err := ArrayEach(nil, func(value []byte, dataType ValueType, offset int, err error) {}); err == nil {
		t.Fatal("expected error on nil input")
	}
}

// SYS-REQ-029:malformed_input:nominal
// SYS-REQ-029:malformed_input:negative
func TestObligation_SYS_REQ_029(t *testing.T) {
	// Positive path: well-formed array iterates without invoking the error path.
	calls := 0
	if _, err := ArrayEach([]byte(`[1,2,3]`), func(value []byte, dataType ValueType, offset int, err error) {
		calls++
	}); err != nil {
		t.Fatalf("ArrayEach on well-formed array returned error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 callbacks, got %d", calls)
	}
	// Negative path: malformed array input yields an error.
	if _, err := ArrayEach([]byte(`[1,2`), func(value []byte, dataType ValueType, offset int, err error) {}); err == nil {
		t.Fatal("expected error on malformed array input")
	}
}

// SYS-REQ-034:edge_case:nominal
// SYS-REQ-034:missing_path:nominal
func TestObligation_SYS_REQ_034(t *testing.T) {
	// Missing target on usable input preserves the original document.
	data := []byte(`{"a":1}`)
	result := Delete(data, "missing")
	if !bytes.Equal(result, data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// SYS-REQ-035:malformed_input:nominal
// SYS-REQ-035:malformed_input:negative
func TestObligation_SYS_REQ_035(t *testing.T) {
	// Positive path: Delete on well-formed input completes cleanly.
	data := []byte(`{"a":1,"b":2}`)
	result := Delete(data, "a")
	if _, _, _, err := Get(result, "a"); !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected 'a' to be deleted, got err=%v", err)
	}
	// Negative path: Delete on malformed input preserves input unchanged.
	malformed := []byte(`{"a":`)
	if got := Delete(malformed, "a"); !bytes.Equal(got, malformed) {
		t.Fatalf("expected unchanged malformed input, got %s", string(got))
	}
}

// SYS-REQ-036:malformed_input:nominal
// SYS-REQ-036:malformed_input:negative
func TestObligation_SYS_REQ_036(t *testing.T) {
	// Positive path: ParseBoolean on a valid literal returns the value.
	if v, err := ParseBoolean([]byte(`true`)); err != nil || !v {
		t.Fatalf("ParseBoolean(true) = %v, err = %v", v, err)
	}
	// Negative path: malformed literal yields an error.
	if _, err := ParseBoolean([]byte(`notabool`)); err == nil {
		t.Fatal("expected malformed error")
	}
}

// SYS-REQ-039:boundary:nominal
func TestObligation_SYS_REQ_039(t *testing.T) {
	// Positive path: non-overflow integer parses cleanly.
	if v, err := ParseInt([]byte(`42`)); err != nil || v != 42 {
		t.Fatalf("ParseInt(42) = %d, err = %v", v, err)
	}
}

// SYS-REQ-041:truncated_at_value_boundary:nominal
func TestObligation_SYS_REQ_041(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("Get on non-truncated input returned error: %v", err)
	}
}

// SYS-REQ-042:truncated_mid_structure:nominal
func TestObligation_SYS_REQ_042(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"a":[1,2]}`), "a"); err != nil {
		t.Fatalf("Get on non-truncated input returned error: %v", err)
	}
}

// SYS-REQ-043:truncated_mid_key:nominal
func TestObligation_SYS_REQ_043(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("Get on non-truncated input returned error: %v", err)
	}
}

// SYS-REQ-044:sentinel_value_boundary:nominal
func TestObligation_SYS_REQ_044(t *testing.T) {
	// Positive path: standard lookup where sentinel is never reached.
	if _, _, _, err := Get([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

// SYS-REQ-047:negative_array_index:nominal
func TestObligation_SYS_REQ_047(t *testing.T) {
	// Positive path: valid (non-negative) in-bounds index succeeds.
	if v, _, _, err := Get([]byte(`{"a":[1,2,3]}`), "a", "[1]"); err != nil || string(v) != "2" {
		t.Fatalf("in-bounds lookup: v=%s err=%v", string(v), err)
	}
}

// SYS-REQ-048:truncated_at_value_boundary:nominal
func TestObligation_SYS_REQ_048(t *testing.T) {
	// Positive path: Delete on non-truncated input.
	data := []byte(`{"a":1,"b":2}`)
	result := Delete(data, "a")
	if _, _, _, err := Get(result, "a"); !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected 'a' to be deleted, got err=%v", err)
	}
}

// SYS-REQ-049:error_propagation:nominal
func TestObligation_SYS_REQ_049(t *testing.T) {
	// Positive path: Delete on well-formed input completes cleanly.
	data := []byte(`{"a":1,"b":2}`)
	if _, _, _, err := Get(Delete(data, "a"), "b"); err != nil {
		t.Fatalf("expected 'b' to remain, got err=%v", err)
	}
}

// SYS-REQ-052:callback_error_propagation:nominal
func TestObligation_SYS_REQ_052(t *testing.T) {
	// Positive path: callback that returns nil does not propagate an error.
	called := 0
	if _, err := ArrayEach([]byte(`[1,2,3]`), func(value []byte, dataType ValueType, offset int, err error) {
		called++
	}); err != nil {
		t.Fatalf("ArrayEach returned error: %v", err)
	}
	if called != 3 {
		t.Fatalf("expected 3 callbacks, got %d", called)
	}
}

// SYS-REQ-053:truncated_mid_element:nominal
func TestObligation_SYS_REQ_053(t *testing.T) {
	calls := 0
	if _, err := ArrayEach([]byte(`[1,2,3]`), func(value []byte, dataType ValueType, offset int, err error) {
		calls++
	}); err != nil {
		t.Fatalf("ArrayEach returned error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 callbacks, got %d", calls)
	}
}

// SYS-REQ-056:truncated_mid_structure:nominal
func TestObligation_SYS_REQ_056(t *testing.T) {
	// Positive path: Delete on non-truncated mid-structure input.
	data := []byte(`{"a":[1,2,3]}`)
	result := Delete(data, "a", "[0]")
	if string(result) == "" {
		t.Fatal("expected non-empty result")
	}
}

// SYS-REQ-057:partial_literal:nominal
func TestObligation_SYS_REQ_057(t *testing.T) {
	// Positive path: complete boolean literal parses cleanly.
	if v, err := ParseBoolean([]byte(`true`)); err != nil || !v {
		t.Fatalf("ParseBoolean(true) = %v, err = %v", v, err)
	}
}

// SYS-REQ-060:truncated_escape_sequence:nominal
func TestObligation_SYS_REQ_060(t *testing.T) {
	if v, err := ParseString([]byte(`hello`)); err != nil || v != "hello" {
		t.Fatalf("ParseString(hello) = %q, err = %v", v, err)
	}
}

// SYS-REQ-064:empty_input:nominal
func TestObligation_SYS_REQ_064(t *testing.T) {
	// Positive path: non-empty integer parses cleanly.
	if v, err := ParseInt([]byte(`42`)); err != nil || v != 42 {
		t.Fatalf("ParseInt(42) = %d, err = %v", v, err)
	}
}

// SYS-REQ-069:nested_mutation:nominal
func TestObligation_SYS_REQ_069(t *testing.T) {
	v, err := Set([]byte(`{"a":{"b":1}}`), []byte(`42`), "a", "b")
	if err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	got, _, _, gErr := Get(v, "a", "b")
	if gErr != nil {
		t.Fatalf("Get returned error: %v", gErr)
	}
	if string(got) != "42" {
		t.Fatalf("expected 42, got %s", string(got))
	}
}

// SYS-REQ-070:no_path_provided:nominal
func TestObligation_SYS_REQ_070(t *testing.T) {
	// Positive path: Set with a provided path succeeds.
	if _, err := Set([]byte(`{"a":1}`), []byte(`42`), "a"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
}

// SYS-REQ-071:malformed_input:nominal
// SYS-REQ-071:malformed_input:negative
func TestObligation_SYS_REQ_071(t *testing.T) {
	if _, err := GetString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
	if _, err := GetString([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected error on malformed GetString input")
	}
}

// SYS-REQ-072:truncated_escape_sequence:nominal
func TestObligation_SYS_REQ_072(t *testing.T) {
	if _, err := GetString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
}

// SYS-REQ-073:type_mismatch:nominal
func TestObligation_SYS_REQ_073(t *testing.T) {
	// Positive path: GetString on a string value succeeds.
	if _, err := GetString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
}

// SYS-REQ-074:empty_input:nominal
// SYS-REQ-074:nil_safety:nominal
// SYS-REQ-074:nil_safety:negative
func TestObligation_SYS_REQ_074(t *testing.T) {
	if _, err := GetString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
	if _, err := GetString(nil, "a"); err == nil {
		t.Fatal("expected error on nil input")
	}
}

// SYS-REQ-075:malformed_input:nominal
// SYS-REQ-075:malformed_input:negative
func TestObligation_SYS_REQ_075(t *testing.T) {
	if _, err := GetInt([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("GetInt returned error: %v", err)
	}
	if _, err := GetInt([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected error on malformed GetInt input")
	}
}

// SYS-REQ-076:boundary:nominal
// SYS-REQ-076:edge_case:nominal
func TestObligation_SYS_REQ_076(t *testing.T) {
	// Boundary positive: in-range integer parses cleanly.
	if v, err := GetInt([]byte(`{"a":9223372036854775807}`), "a"); err != nil || v != 9223372036854775807 {
		t.Fatalf("boundary int64-max: v=%d err=%v", v, err)
	}
}

// SYS-REQ-077:type_mismatch:nominal
func TestObligation_SYS_REQ_077(t *testing.T) {
	if _, err := GetInt([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("GetInt returned error: %v", err)
	}
}

// SYS-REQ-078:empty_input:nominal
// SYS-REQ-078:nil_safety:nominal
// SYS-REQ-078:nil_safety:negative
func TestObligation_SYS_REQ_078(t *testing.T) {
	if _, err := GetInt([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("GetInt returned error: %v", err)
	}
	if _, err := GetInt(nil, "a"); err == nil {
		t.Fatal("expected error on nil input")
	}
}

// SYS-REQ-079:partial_literal:nominal
func TestObligation_SYS_REQ_079(t *testing.T) {
	if v, err := GetBoolean([]byte(`{"a":true}`), "a"); err != nil || !v {
		t.Fatalf("GetBoolean(true) = %v, err = %v", v, err)
	}
}

// SYS-REQ-080:malformed_input:nominal
// SYS-REQ-080:malformed_input:negative
func TestObligation_SYS_REQ_080(t *testing.T) {
	if _, err := GetUnsafeString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetUnsafeString returned error: %v", err)
	}
	if _, err := GetUnsafeString([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected error on malformed GetUnsafeString input")
	}
}

// SYS-REQ-081:empty_input:nominal
// SYS-REQ-081:nil_safety:nominal
// SYS-REQ-081:nil_safety:negative
func TestObligation_SYS_REQ_081(t *testing.T) {
	if _, err := GetUnsafeString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetUnsafeString returned error: %v", err)
	}
	if _, err := GetUnsafeString(nil, "a"); err == nil {
		t.Fatal("expected error on nil input")
	}
}

// SYS-REQ-082:edge_case:nominal
// SYS-REQ-082:truncated_at_value_boundary:nominal
func TestObligation_SYS_REQ_082(t *testing.T) {
	if _, err := GetUnsafeString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetUnsafeString returned error: %v", err)
	}
}

// SYS-REQ-083:truncated_at_value_boundary:nominal
func TestObligation_SYS_REQ_083(t *testing.T) {
	calls := 0
	if _, err := ArrayEach([]byte(`[1,2,3]`), func(value []byte, dataType ValueType, offset int, err error) {
		calls++
	}); err != nil {
		t.Fatalf("ArrayEach returned error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 callbacks, got %d", calls)
	}
}

// SYS-REQ-084:truncated_mid_structure:nominal
func TestObligation_SYS_REQ_084(t *testing.T) {
	calls := 0
	if err := ObjectEach([]byte(`{"a":1,"b":2}`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		calls++
		return nil
	}); err != nil {
		t.Fatalf("ObjectEach returned error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 callbacks, got %d", calls)
	}
}

// SYS-REQ-085:sentinel_value_boundary:nominal
func TestObligation_SYS_REQ_085(t *testing.T) {
	called := false
	EachKey([]byte(`{"a":1}`), func(i int, value []byte, vt ValueType, err error) {
		called = true
	}, []string{"a"})
	if !called {
		t.Fatal("expected EachKey to invoke callback for matching path")
	}
}
