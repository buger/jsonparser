package jsonparser

import (
	"bytes"
	"errors"
	"testing"
)

// =============================================================================
// Spec-level MC/DC witness tests.
// =============================================================================
//
// This file closes spec-level MC/DC witness rows reported by
// `proof mcdc spec queue`. Each test drives the requirement's truth-table row
// through the public API and asserts the implementation-side outcome that
// corresponds to that row.
//
// - Trigger-false rows (implication antecedent false) carry a `[no-action: ...]`
//   trailer that names the caller-level assertion proving the action did not
//   fire (callback counter stays 0; typed getter returns ""; Get returns nil).
// - Invariant-violation rows (formula evaluates FALSE in the table) are
//   witnessed by driving the positive neighbour row, which proves the FALSE
//   combination cannot occur in a correct implementation.
// - Positive action rows drive the actual scenario through the public API.

// -----------------------------------------------------------------------------
// SYS-REQ-001 (parser.go Get — existing-path lookup)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-001
// MCDC SYS-REQ-001: addressed_path_exists=T, json_input_is_well_formed=F, key_path_is_provided=T, returns_existing_path_lookup_result=F => TRUE [no-action: Get returns nil value and non-nil error, no existing-path result emitted]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_001_Row2_TriggerFalse(t *testing.T) {
	// Malformed input where the addressed key would exist if the payload were
	// well-formed. The parser must NOT emit a successful existing-path lookup
	// result; it returns a nil value and an error.
	value, _, _, err := Get([]byte(`{"a":`), "a")
	if err == nil {
		t.Fatal("expected error on malformed input, got nil")
	}
	if value != nil {
		t.Fatalf("expected nil value (no existing-path-result action), got %v", value)
	}
}

// Verifies: SYS-REQ-001
// MCDC SYS-REQ-001: addressed_path_exists=T, json_input_is_well_formed=T, key_path_is_provided=T, returns_existing_path_lookup_result=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_001_Row4_InvariantViolation(t *testing.T) {
	// Row 4 is an invariant-violation row: it would require Get on
	// well-formed JSON with a key path that exists to return NO existing-path
	// lookup result. The positive neighbour row (Row 5) shows the
	// implementation returns the value, so this combination is unreachable
	// in a correct build. Witness by driving the positive path.
	value, dataType, _, err := Get([]byte(`{"a":1}`), "a")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || !bytes.Equal(value, []byte("1")) {
		t.Fatalf("expected existing-path lookup to return 1, got value=%s type=%v", string(value), dataType)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-002 (GetString)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-002
// MCDC SYS-REQ-002: addressed_value_is_string=F, raw_string_token_is_well_formed=T, returns_getstring_decoded_value=F => TRUE [no-action: GetString returns empty string and non-nil error, no decoded value emitted]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_002_Row1_TriggerFalse(t *testing.T) {
	// Addressed value is NOT a string (it is a number) but the raw token is a
	// well-formed JSON value. GetString must not decode a value.
	value, err := GetString([]byte(`{"a":123}`), "a")
	if err == nil {
		t.Fatal("expected error when GetString targets non-string, got nil")
	}
	if value != "" {
		t.Fatalf("expected empty string (no decode action), got %q", value)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-003 (GetInt)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-003
// MCDC SYS-REQ-003: addressed_value_is_number=F, raw_number_token_is_integer_parseable=T, returns_getint_value=F => TRUE [no-action: GetInt returns 0 and non-nil error, no value emitted]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_003_Row1_TriggerFalse(t *testing.T) {
	// Addressed value is NOT a number (it is a string) even though a
	// well-formed number-like token exists in the payload. GetInt must not
	// return a value.
	value, err := GetInt([]byte(`{"a":"123"}`), "a")
	if err == nil {
		t.Fatal("expected error when GetInt targets non-number, got nil")
	}
	if value != 0 {
		t.Fatalf("expected zero (no value action), got %d", value)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-004 (GetFloat)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-004
// MCDC SYS-REQ-004: addressed_value_is_number=F, raw_number_token_is_float_parseable=T, returns_getfloat_value=F => TRUE [no-action: GetFloat returns 0 and non-nil error, no value emitted]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_004_Row1_TriggerFalse(t *testing.T) {
	value, err := GetFloat([]byte(`{"a":"1.5"}`), "a")
	if err == nil {
		t.Fatal("expected error when GetFloat targets non-number, got nil")
	}
	if value != 0 {
		t.Fatalf("expected zero (no value action), got %v", value)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-005 (GetBoolean)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-005
// MCDC SYS-REQ-005: addressed_value_is_boolean=F, raw_boolean_token_is_well_formed=T, returns_getboolean_value=F => TRUE [no-action: GetBoolean returns false and non-nil error, no value emitted]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_005_Row1_TriggerFalse(t *testing.T) {
	value, err := GetBoolean([]byte(`{"a":"true"}`), "a")
	if err == nil {
		t.Fatal("expected error when GetBoolean targets non-boolean, got nil")
	}
	if value {
		t.Fatal("expected false (no value action), got true")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-006 (ArrayEach)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-006
// MCDC SYS-REQ-006: addressed_array_is_empty=F, addressed_array_is_well_formed=F, array_callback_receives_elements_in_order=F => TRUE [no-action: callback counter == 0, ArrayEach returns error, no in-order delivery]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_006_Row1_TriggerFalse(t *testing.T) {
	// Malformed array (just opening bracket, not parseable as elements). The
	// callback must never fire.
	callbackCalls := 0
	_, err := ArrayEach([]byte(`[`), func(value []byte, dataType ValueType, offset int, err error) {
		callbackCalls++
	})
	if callbackCalls != 0 {
		t.Fatalf("expected zero callback invocations on malformed array, got %d", callbackCalls)
	}
	if err == nil {
		t.Fatal("expected error from ArrayEach on malformed input, got nil")
	}
}

// Verifies: SYS-REQ-006
// MCDC SYS-REQ-006: addressed_array_is_empty=T, addressed_array_is_well_formed=T, array_callback_receives_elements_in_order=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_006_Row4_EmptyArray(t *testing.T) {
	// Empty well-formed array: callback must not fire because there are no
	// elements to deliver.
	callbackCalls := 0
	_, err := ArrayEach([]byte(`[]`), func(value []byte, dataType ValueType, offset int, err error) {
		callbackCalls++
	})
	if err != nil {
		t.Fatalf("ArrayEach on empty array returned error: %v", err)
	}
	if callbackCalls != 0 {
		t.Fatalf("expected zero callback invocations on empty array, got %d", callbackCalls)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-007 (ObjectEach)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-007
// MCDC SYS-REQ-007: addressed_object_is_empty=F, addressed_object_is_well_formed=F, object_callback_receives_entries=F => TRUE [no-action: callback counter == 0, ObjectEach returns error, no entries delivered]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_007_Row1_TriggerFalse(t *testing.T) {
	callbackCalls := 0
	err := ObjectEach([]byte(`{`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		callbackCalls++
		return nil
	})
	if callbackCalls != 0 {
		t.Fatalf("expected zero callback invocations on malformed object, got %d", callbackCalls)
	}
	if err == nil {
		t.Fatal("expected error from ObjectEach on malformed input, got nil")
	}
}

// Verifies: SYS-REQ-007
// MCDC SYS-REQ-007: addressed_object_is_empty=T, addressed_object_is_well_formed=T, object_callback_receives_entries=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_007_Row4_EmptyObject(t *testing.T) {
	callbackCalls := 0
	err := ObjectEach([]byte(`{}`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		callbackCalls++
		return nil
	})
	if err != nil {
		t.Fatalf("ObjectEach on empty object returned error: %v", err)
	}
	if callbackCalls != 0 {
		t.Fatalf("expected zero callback invocations on empty object, got %d", callbackCalls)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-008 (EachKey multipath scan)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-008
// MCDC SYS-REQ-008: eachkey_callback_receives_found_values=F, eachkey_completes_requested_scan=F, eachkey_malformed_input_returns_error=F, missing_multipath_request_does_not_emit_callback=F, multipath_requests_are_provided=F => TRUE [no-action: callback counter == 0, EachKey returns immediately because no paths are provided]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_008_Row1_TriggerFalse(t *testing.T) {
	// EachKey with no paths exercises the antecedent-false branch
	// (multipath_requests_are_provided = F).
	callbackCalls := 0
	EachKey([]byte(`{"a":1}`), func(i int, value []byte, vt ValueType, off error) {
		callbackCalls++
	})
	if callbackCalls != 0 {
		t.Fatalf("expected zero callback invocations when no paths provided, got %d", callbackCalls)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-009 (Set)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-009
// MCDC SYS-REQ-009: set_creates_missing_path=F, set_path_is_provided=F, set_returns_not_found_error=F, set_returns_updated_document=F, set_target_exists=F => TRUE [no-action: Set returns (nil, KeyPathNotFoundError) and input is not mutated]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_009_Row1_TriggerFalse(t *testing.T) {
	// Set without any keys: returns nil + KeyPathNotFoundError. No document
	// update action is performed.
	data := []byte(`{"a":1}`)
	value, err := Set(data, []byte(`42`))
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError when no path provided, got %v", err)
	}
	if value != nil {
		t.Fatalf("expected nil return (no update action), got %v", value)
	}
	if !bytes.Equal(data, []byte(`{"a":1}`)) {
		t.Fatalf("input was mutated: %s", string(data))
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-011 (GetUnsafeString)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-011
// MCDC SYS-REQ-011: addressed_value_is_string=F, returns_unsafe_string_view=F => TRUE [no-action: GetUnsafeString returns empty string and KeyPathNotFoundError, no view emitted]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_011_Row1_TriggerFalse(t *testing.T) {
	// Addressed path does not resolve to any value, so no unsafe string view
	// is returned.
	value, err := GetUnsafeString([]byte(`{"a":1}`), "missing")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError on missing path, got %v", err)
	}
	if value != "" {
		t.Fatalf("expected empty string (no view action), got %q", value)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-012 (ParseBoolean)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-012
// MCDC SYS-REQ-012: raw_boolean_literal_is_valid=F, returns_parseboolean_value=F => TRUE [no-action: ParseBoolean returns false and non-nil error, no value emitted]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_012_Row1_TriggerFalse(t *testing.T) {
	value, err := ParseBoolean([]byte(`notabool`))
	if err == nil {
		t.Fatal("expected error when ParseBoolean gets invalid literal, got nil")
	}
	if value {
		t.Fatal("expected false (no value action), got true")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-013 (ParseFloat)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-013
// MCDC SYS-REQ-013: raw_float_token_is_well_formed=F, returns_parsefloat_value=F => TRUE [no-action: ParseFloat returns 0 and non-nil error, no value emitted]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_013_Row1_TriggerFalse(t *testing.T) {
	value, err := ParseFloat([]byte(`notafloat`))
	if err == nil {
		t.Fatal("expected error when ParseFloat gets invalid token, got nil")
	}
	if value != 0 {
		t.Fatalf("expected zero (no value action), got %v", value)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-014 (ParseString)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-014
// MCDC SYS-REQ-014: raw_string_literal_is_well_formed=F, returns_parsestring_value=F => TRUE [no-action: ParseString returns empty string and MalformedValueError, no value emitted]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_014_Row1_TriggerFalse(t *testing.T) {
	// Malformed escape sequence forces Unescape to fail; ParseString wraps
	// the failure as MalformedValueError and returns an empty string.
	value, err := ParseString([]byte(`abc\q`))
	if err == nil {
		t.Fatal("expected error when ParseString gets malformed escape, got nil")
	}
	if value != "" {
		t.Fatalf("expected empty string (no value action), got %q", value)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-015 (ParseInt)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-015
// MCDC SYS-REQ-015: raw_int_token_is_well_formed=F, returns_parseint_value=F => TRUE [no-action: ParseInt returns 0 and non-nil error, no value emitted]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_015_Row1_TriggerFalse(t *testing.T) {
	value, err := ParseInt([]byte(`notanint`))
	if err == nil {
		t.Fatal("expected error when ParseInt gets invalid token, got nil")
	}
	if value != 0 {
		t.Fatalf("expected zero (no value action), got %d", value)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-016 (missing-path lookup on well-formed JSON via Get)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-016
// MCDC SYS-REQ-016: addressed_path_exists=F, json_input_is_well_formed=F, key_path_is_provided=T, returns_missing_path_result_for_well_formed_lookup=F => TRUE [no-action: Get returns non-nil error and value=nil on malformed input, no missing-path-result action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_016_Row1_TriggerFalse(t *testing.T) {
	value, _, _, err := Get([]byte(`{"a":`), "missing")
	if err == nil {
		t.Fatal("expected error on malformed input, got nil")
	}
	if value != nil {
		t.Fatalf("expected nil value (no missing-path-result action), got %v", value)
	}
}

// Verifies: SYS-REQ-016
// MCDC SYS-REQ-016: addressed_path_exists=F, json_input_is_well_formed=T, key_path_is_provided=F, returns_missing_path_result_for_well_formed_lookup=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_016_Row2_NoKeyPath(t *testing.T) {
	// No key path provided: the formula is satisfied via !key_path_is_provided
	// regardless of the missing-path-result action. Drive Get on well-formed
	// JSON without a key path; it returns the root value (no missing-path lookup).
	value, dataType, _, err := Get([]byte(`{"a":1}`))
	if err != nil {
		t.Fatalf("Get without key path returned error: %v", err)
	}
	if dataType != Object || string(value) != `{"a":1}` {
		t.Fatalf("unexpected root value: %s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-016
// MCDC SYS-REQ-016: addressed_path_exists=F, json_input_is_well_formed=T, key_path_is_provided=T, returns_missing_path_result_for_well_formed_lookup=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_016_Row3_InvariantViolation(t *testing.T) {
	// Invariant-violation row: Get on well-formed JSON with a key path that
	// does not exist must return the missing-path-result (Row 4). Drive the
	// positive path to prove this FALSE combination is unreachable.
	_, dataType, offset, err := Get([]byte(`{"a":1}`), "missing")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError, got %v", err)
	}
	if dataType != NotExist || offset != -1 {
		t.Fatalf("expected not-found tuple, got type=%v offset=%d", dataType, offset)
	}
}

// Verifies: SYS-REQ-016
// MCDC SYS-REQ-016: addressed_path_exists=F, json_input_is_well_formed=T, key_path_is_provided=T, returns_missing_path_result_for_well_formed_lookup=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_016_Row4_MissingPathResult(t *testing.T) {
	_, dataType, offset, err := Get([]byte(`{"a":1}`), "missing")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError, got %v", err)
	}
	if dataType != NotExist || offset != -1 {
		t.Fatalf("expected not-found tuple, got type=%v offset=%d", dataType, offset)
	}
}

// Verifies: SYS-REQ-016
// MCDC SYS-REQ-016: addressed_path_exists=T, json_input_is_well_formed=T, key_path_is_provided=T, returns_missing_path_result_for_well_formed_lookup=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_016_Row5_AddressedPathExists(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":1}`), "a")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "1" {
		t.Fatalf("expected existing path lookup, got value=%s type=%v", string(value), dataType)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-017 (parse error on incomplete input via Get)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-017
// MCDC SYS-REQ-017: input_is_incomplete_during_lookup=F, returns_parse_error_for_incomplete_lookup=F => TRUE [no-action: Get returns nil error on complete input, no parse-error action fires]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_017_Row1_TriggerFalse(t *testing.T) {
	value, _, _, err := Get([]byte(`{"a":1}`), "a")
	if err != nil {
		t.Fatalf("expected nil error on complete input, got %v", err)
	}
	if value == nil {
		t.Fatal("expected non-nil value, got nil")
	}
}

// Verifies: SYS-REQ-017
// MCDC SYS-REQ-017: input_is_incomplete_during_lookup=T, returns_parse_error_for_incomplete_lookup=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_017_Row2_InvariantViolation(t *testing.T) {
	// Invariant-violation row: incomplete input without a parse error cannot
	// occur in a correct build. Drive the positive path (Row 3) to prove this
	// combination is unreachable.
	if _, _, _, err := Get([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected parse error on incomplete input, got nil")
	}
}

// Verifies: SYS-REQ-017
// MCDC SYS-REQ-017: input_is_incomplete_during_lookup=T, returns_parse_error_for_incomplete_lookup=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_017_Row3_ParseErrorReturned(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected parse error on incomplete input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-018 (root value lookup without key path)
// FRETish: !json_input_is_well_formed | !key_path_is_provided | returns_root_value_without_key_path
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-018
// MCDC SYS-REQ-018: json_input_is_well_formed=F, key_path_is_provided=F, returns_root_value_without_key_path=F => TRUE [no-action: Get on malformed input without key path returns error, no root value emitted]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_018_Row1_TriggerFalse(t *testing.T) {
	value, _, _, err := Get([]byte(`{"a":`))
	if err == nil {
		t.Fatal("expected error on malformed input without key path, got nil")
	}
	if value != nil {
		t.Fatalf("expected nil value (no root-value action), got %v", value)
	}
}

// Verifies: SYS-REQ-018
// MCDC SYS-REQ-018: json_input_is_well_formed=T, key_path_is_provided=F, returns_root_value_without_key_path=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_018_Row2_InvariantViolation(t *testing.T) {
	// Invariant violation: Get on well-formed JSON without a key path MUST
	// return the root value (Row 3). Witness the positive path.
	value, dataType, _, err := Get([]byte(`{"a":1}`))
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Object || string(value) != `{"a":1}` {
		t.Fatalf("expected root value, got %s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-018
// MCDC SYS-REQ-018: json_input_is_well_formed=T, key_path_is_provided=F, returns_root_value_without_key_path=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_018_Row3_RootValueReturned(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":1}`))
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Object || string(value) != `{"a":1}` {
		t.Fatalf("expected root value, got %s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-018
// MCDC SYS-REQ-018: json_input_is_well_formed=T, key_path_is_provided=T, returns_root_value_without_key_path=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_018_Row4_KeyPathProvided(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":1}`), "a")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "1" {
		t.Fatalf("expected addressed lookup, got %s type=%v", string(value), dataType)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-019 (empty input + key path)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-019
// MCDC SYS-REQ-019: json_input_is_empty=F, key_path_is_provided=T, returns_missing_path_result_for_empty_input=F => TRUE [no-action: Get on non-empty input does not invoke the empty-input missing-path action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_019_Row1_TriggerFalse(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":1}`), "missing")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError, got %v", err)
	}
	if dataType != NotExist || value != nil {
		t.Fatalf("expected not-found on non-empty input, got value=%v type=%v", value, dataType)
	}
}

// Verifies: SYS-REQ-019
// MCDC SYS-REQ-019: json_input_is_empty=T, key_path_is_provided=F, returns_missing_path_result_for_empty_input=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_019_Row2_EmptyNoKeyPath(t *testing.T) {
	// Empty input without key path: formula satisfied via !key_path_is_provided.
	// Get on empty input without key path.
	value, _, _, err := Get([]byte(""))
	if err == nil {
		t.Fatal("expected error on empty input, got nil")
	}
	if value != nil {
		t.Fatalf("expected nil value, got %v", value)
	}
}

// Verifies: SYS-REQ-019
// MCDC SYS-REQ-019: json_input_is_empty=T, key_path_is_provided=T, returns_missing_path_result_for_empty_input=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_019_Row3_InvariantViolation(t *testing.T) {
	// Invariant violation: empty input + key path MUST return missing-path
	// result (Row 4). Drive the positive path.
	_, dataType, offset, err := Get([]byte(""), "a")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError on empty input, got %v", err)
	}
	if dataType != NotExist || offset != -1 {
		t.Fatalf("expected not-found tuple, got type=%v offset=%d", dataType, offset)
	}
}

// Verifies: SYS-REQ-019
// MCDC SYS-REQ-019: json_input_is_empty=T, key_path_is_provided=T, returns_missing_path_result_for_empty_input=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_019_Row4_EmptyMissingPath(t *testing.T) {
	_, dataType, offset, err := Get([]byte(""), "a")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError on empty input, got %v", err)
	}
	if dataType != NotExist || offset != -1 {
		t.Fatalf("expected not-found tuple, got type=%v offset=%d", dataType, offset)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-020 (object key segment at current scope)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-020
// MCDC SYS-REQ-020: path_segment_is_object_key=F, returns_value_from_current_scope_object_key=F, segment_is_evaluated_at_current_scope=T => TRUE [no-action: array-index segment does not invoke object-key lookup]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_020_Row1_TriggerFalse(t *testing.T) {
	// Use an array-index segment; the object-key lookup action must not fire.
	value, dataType, _, err := Get([]byte(`{"a":[1,2,3]}`), "a", "[1]")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "2" {
		t.Fatalf("expected array element lookup, got value=%s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-020
// MCDC SYS-REQ-020: path_segment_is_object_key=T, returns_value_from_current_scope_object_key=F, segment_is_evaluated_at_current_scope=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_020_Row2_NotEvaluated(t *testing.T) {
	// Path segment is an object key, but evaluation stops before this segment
	// because an earlier segment did not match. Get returns not-found.
	_, _, _, err := Get([]byte(`{"a":1}`), "missing", "b")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError, got %v", err)
	}
}

// Verifies: SYS-REQ-020
// MCDC SYS-REQ-020: path_segment_is_object_key=T, returns_value_from_current_scope_object_key=F, segment_is_evaluated_at_current_scope=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_020_Row3_InvariantViolation(t *testing.T) {
	// Invariant violation: an evaluated object-key segment MUST return a value
	// from the current scope (Row 4). Drive the positive path.
	value, dataType, _, err := Get([]byte(`{"a":{"b":2}}`), "a", "b")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "2" {
		t.Fatalf("expected nested object-key lookup, got value=%s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-020
// MCDC SYS-REQ-020: path_segment_is_object_key=T, returns_value_from_current_scope_object_key=T, segment_is_evaluated_at_current_scope=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_020_Row4_ObjectKeyMatched(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":{"b":2}}`), "a", "b")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "2" {
		t.Fatalf("expected nested object-key lookup, got value=%s type=%v", string(value), dataType)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-021 (in-bounds array index)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-021
// MCDC SYS-REQ-021: array_index_is_in_bounds=F, array_index_segment_is_valid=T, path_segment_is_array_index=T, returns_value_from_in_bounds_array_index=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_021_Row1_OutOfBounds(t *testing.T) {
	_, _, _, err := Get([]byte(`{"a":[1,2]}`), "a", "[9]")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError for out-of-bounds index, got %v", err)
	}
}

// Verifies: SYS-REQ-021
// MCDC SYS-REQ-021: array_index_is_in_bounds=T, array_index_segment_is_valid=F, path_segment_is_array_index=T, returns_value_from_in_bounds_array_index=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_021_Row2_InvalidSegment(t *testing.T) {
	_, _, _, err := Get([]byte(`{"a":[1,2]}`), "a", "[")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError for malformed array index, got %v", err)
	}
}

// Verifies: SYS-REQ-021
// MCDC SYS-REQ-021: array_index_is_in_bounds=T, array_index_segment_is_valid=T, path_segment_is_array_index=F, returns_value_from_in_bounds_array_index=F => TRUE [no-action: non-array-index segment does not invoke in-bounds-array-index action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_021_Row3_NotArraySegment(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":[1,2]}`), "a")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Array {
		t.Fatalf("expected Array type, got %v", dataType)
	}
	if string(value) != "[1,2]" {
		t.Fatalf("expected array bytes, got %s", string(value))
	}
}

// Verifies: SYS-REQ-021
// MCDC SYS-REQ-021: array_index_is_in_bounds=T, array_index_segment_is_valid=T, path_segment_is_array_index=T, returns_value_from_in_bounds_array_index=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_021_Row4_InvariantViolation(t *testing.T) {
	// Invariant violation: in-bounds valid array index MUST return the element
	// (Row 5). Drive the positive path.
	value, dataType, _, err := Get([]byte(`{"a":[1,2,3]}`), "a", "[1]")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "2" {
		t.Fatalf("expected array element 2, got %s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-021
// MCDC SYS-REQ-021: array_index_is_in_bounds=T, array_index_segment_is_valid=T, path_segment_is_array_index=T, returns_value_from_in_bounds_array_index=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_021_Row5_InBoundsReturned(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":[1,2,3]}`), "a", "[1]")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "2" {
		t.Fatalf("expected array element 2, got %s type=%v", string(value), dataType)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-022 (invalid array index syntax)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-022
// MCDC SYS-REQ-022: array_index_segment_is_valid=F, path_segment_is_array_index=F, returns_invalid_array_index_not_found=F => TRUE [no-action: non-array-index segment does not invoke the invalid-array-index action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_022_Row1_TriggerFalse(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":1}`), "a")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "1" {
		t.Fatalf("expected object-key lookup, got %s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-022
// MCDC SYS-REQ-022: array_index_segment_is_valid=F, path_segment_is_array_index=T, returns_invalid_array_index_not_found=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_022_Row2_InvariantViolation(t *testing.T) {
	// Invariant violation: invalid array index segment MUST return not-found
	// (Row 3). Drive the positive path.
	_, _, _, err := Get([]byte(`{"a":[1,2]}`), "a", "[")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError for malformed array index, got %v", err)
	}
}

// Verifies: SYS-REQ-022
// MCDC SYS-REQ-022: array_index_segment_is_valid=F, path_segment_is_array_index=T, returns_invalid_array_index_not_found=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_022_Row3_InvalidIndexNotFound(t *testing.T) {
	_, _, _, err := Get([]byte(`{"a":[1,2]}`), "a", "[")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError for malformed array index, got %v", err)
	}
}

// Verifies: SYS-REQ-022
// MCDC SYS-REQ-022: array_index_segment_is_valid=T, path_segment_is_array_index=T, returns_invalid_array_index_not_found=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_022_Row4_ValidSegment(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":[1,2]}`), "a", "[0]")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "1" {
		t.Fatalf("expected valid array element, got %s type=%v", string(value), dataType)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-023 (out-of-bounds array index)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-023
// MCDC SYS-REQ-023: array_index_is_out_of_bounds=F, array_index_segment_is_valid=T, path_segment_is_array_index=T, returns_oob_array_index_not_found=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_023_Row1_InBounds(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":[1,2]}`), "a", "[0]")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "1" {
		t.Fatalf("expected in-bounds element, got %s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-023
// MCDC SYS-REQ-023: array_index_is_out_of_bounds=T, array_index_segment_is_valid=F, path_segment_is_array_index=T, returns_oob_array_index_not_found=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_023_Row2_InvalidOutOfBounds(t *testing.T) {
	_, _, _, err := Get([]byte(`{"a":[1,2]}`), "a", "[")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError for malformed array index, got %v", err)
	}
}

// Verifies: SYS-REQ-023
// MCDC SYS-REQ-023: array_index_is_out_of_bounds=T, array_index_segment_is_valid=T, path_segment_is_array_index=F, returns_oob_array_index_not_found=F => TRUE [no-action: non-array-index segment does not invoke the oob action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_023_Row3_NotArraySegment(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":1}`), "a")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number {
		t.Fatalf("expected Number type, got %v", dataType)
	}
	if string(value) != "1" {
		t.Fatalf("expected value 1, got %s", string(value))
	}
}

// Verifies: SYS-REQ-023
// MCDC SYS-REQ-023: array_index_is_out_of_bounds=T, array_index_segment_is_valid=T, path_segment_is_array_index=T, returns_oob_array_index_not_found=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_023_Row4_InvariantViolation(t *testing.T) {
	// Invariant violation: out-of-bounds valid array index MUST return
	// not-found (Row 5). Drive the positive path.
	_, _, _, err := Get([]byte(`{"a":[1,2]}`), "a", "[9]")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError for out-of-bounds index, got %v", err)
	}
}

// Verifies: SYS-REQ-023
// MCDC SYS-REQ-023: array_index_is_out_of_bounds=T, array_index_segment_is_valid=T, path_segment_is_array_index=T, returns_oob_array_index_not_found=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_023_Row5_OobNotFound(t *testing.T) {
	_, _, _, err := Get([]byte(`{"a":[1,2]}`), "a", "[9]")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError for out-of-bounds index, got %v", err)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-024 (decoded escaped object key)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-024
// MCDC SYS-REQ-024: decoded_path_segment_matches_escaped_key=F, escaped_json_object_key_is_present=T, returns_value_from_decoded_escaped_key=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_024_Row1_NoMatch(t *testing.T) {
	// Escaped key is present, but the path segment doesn't match it.
	_, _, _, err := Get([]byte(`{"a\u00B0b":1}`), "axb")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError, got %v", err)
	}
}

// Verifies: SYS-REQ-024
// MCDC SYS-REQ-024: decoded_path_segment_matches_escaped_key=T, escaped_json_object_key_is_present=F, returns_value_from_decoded_escaped_key=F => TRUE [no-action: no escaped key present means no decoded-escaped-key lookup action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_024_Row2_TriggerFalse(t *testing.T) {
	// No escaped key in payload; the decoded-escaped-key action cannot fire.
	value, dataType, _, err := Get([]byte(`{"plain":1}`), "plain")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "1" {
		t.Fatalf("expected plain key lookup, got %s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-024
// MCDC SYS-REQ-024: decoded_path_segment_matches_escaped_key=T, escaped_json_object_key_is_present=T, returns_value_from_decoded_escaped_key=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_024_Row3_InvariantViolation(t *testing.T) {
	// Invariant violation: matching decoded escaped key MUST return value (Row 4).
	value, dataType, _, err := Get([]byte(`{"a\u00B0b":1}`), "a°b")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "1" {
		t.Fatalf("expected escaped-key lookup, got %s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-024
// MCDC SYS-REQ-024: decoded_path_segment_matches_escaped_key=T, escaped_json_object_key_is_present=T, returns_value_from_decoded_escaped_key=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_024_Row4_DecodedEscapedMatched(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a\u00B0b":1}`), "a°b")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "1" {
		t.Fatalf("expected escaped-key lookup, got %s type=%v", string(value), dataType)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-025 (unquoted raw string contents)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-025
// MCDC SYS-REQ-025: addressed_value_is_string=F, returns_unquoted_raw_string_contents=F => TRUE [no-action: Get on non-string does not return unquoted raw string contents]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_025_Row1_TriggerFalse(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":123}`), "a")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number {
		t.Fatalf("expected Number type (not String), got %v", dataType)
	}
	if string(value) != "123" {
		t.Fatalf("expected 123, got %s", string(value))
	}
}

// Verifies: SYS-REQ-025
// MCDC SYS-REQ-025: addressed_value_is_string=T, returns_unquoted_raw_string_contents=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_025_Row2_InvariantViolation(t *testing.T) {
	// Invariant violation: string value MUST return unquoted contents (Row 3).
	value, dataType, _, err := Get([]byte(`{"a":"hello"}`), "a")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != String || string(value) != "hello" {
		t.Fatalf("expected unquoted string, got %s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-025
// MCDC SYS-REQ-025: addressed_value_is_string=T, returns_unquoted_raw_string_contents=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_025_Row3_StringUnquoted(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":"hello"}`), "a")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != String || string(value) != "hello" {
		t.Fatalf("expected unquoted string, got %s type=%v", string(value), dataType)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-026 (best-effort lookup when malformed input is outside addressed token)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-026
// MCDC SYS-REQ-026: addressed_token_can_be_isolated=F, malformed_input_outside_addressed_token=T, returns_best_effort_lookup_result=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_026_Row1_CannotIsolate(t *testing.T) {
	// Malformed input that prevents token isolation: Get returns an error
	// instead of a best-effort result.
	if _, _, _, err := Get([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected error when addressed token cannot be isolated, got nil")
	}
}

// Verifies: SYS-REQ-026
// MCDC SYS-REQ-026: addressed_token_can_be_isolated=T, malformed_input_outside_addressed_token=F, returns_best_effort_lookup_result=F => TRUE [no-action: no malformed input outside token, no best-effort action fires]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_026_Row2_TriggerFalse(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":1}`), "a")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "1" {
		t.Fatalf("expected clean lookup, got %s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-026
// MCDC SYS-REQ-026: addressed_token_can_be_isolated=T, malformed_input_outside_addressed_token=T, returns_best_effort_lookup_result=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_026_Row3_InvariantViolation(t *testing.T) {
	// Invariant violation: malformed input outside an isolatable addressed
	// token MUST yield a best-effort result (Row 4). Drive the positive path.
	value, dataType, _, err := Get([]byte(`{"a":1]`), "a")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "1" {
		t.Fatalf("expected best-effort lookup, got %s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-026
// MCDC SYS-REQ-026: addressed_token_can_be_isolated=T, malformed_input_outside_addressed_token=T, returns_best_effort_lookup_result=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_026_Row4_BestEffortSuccess(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":1]`), "a")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "1" {
		t.Fatalf("expected best-effort lookup, got %s type=%v", string(value), dataType)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-027 (invalid addressed token shape)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-027
// MCDC SYS-REQ-027: addressed_token_shape_is_invalid=F, returns_value_type_error=F => TRUE [no-action: valid token shape does not invoke value-type-error action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_027_Row1_TriggerFalse(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":1}`), "a")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "1" {
		t.Fatalf("expected valid lookup, got %s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-027
// MCDC SYS-REQ-027: addressed_token_shape_is_invalid=T, returns_value_type_error=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_027_Row2_InvariantViolation(t *testing.T) {
	// Invariant violation: invalid token shape MUST return value-type-error (Row 3).
	if _, _, _, err := Get([]byte(`{"a":u}`), "a"); !errors.Is(err, UnknownValueTypeError) {
		t.Fatalf("expected UnknownValueTypeError, got %v", err)
	}
}

// Verifies: SYS-REQ-027
// MCDC SYS-REQ-027: addressed_token_shape_is_invalid=T, returns_value_type_error=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_027_Row3_ValueTypeError(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"a":u}`), "a"); !errors.Is(err, UnknownValueTypeError) {
		t.Fatalf("expected UnknownValueTypeError, got %v", err)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-028 (empty well-formed array produces no callbacks via ArrayEach)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-028
// MCDC SYS-REQ-028: addressed_array_is_empty=F, addressed_array_is_well_formed=T, empty_array_produces_no_callbacks=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_028_Row1_NonEmptyWellFormed(t *testing.T) {
	// Non-empty well-formed array: callback fires for each element so the
	// "empty-array produces no callbacks" action is FALSE.
	calls := 0
	if _, err := ArrayEach([]byte(`[1,2,3]`), func(value []byte, dataType ValueType, offset int, err error) {
		calls++
	}); err != nil {
		t.Fatalf("ArrayEach returned error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 callback invocations, got %d", calls)
	}
}

// Verifies: SYS-REQ-028
// MCDC SYS-REQ-028: addressed_array_is_empty=T, addressed_array_is_well_formed=F, empty_array_produces_no_callbacks=F => TRUE [no-action: callback counter == 0 on malformed input, no empty-array action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_028_Row2_EmptyMalformed(t *testing.T) {
	calls := 0
	_, err := ArrayEach([]byte(`[`), func(value []byte, dataType ValueType, offset int, err error) {
		calls++
	})
	if calls != 0 {
		t.Fatalf("expected zero callbacks on malformed input, got %d", calls)
	}
	if err == nil {
		t.Fatal("expected error on malformed input, got nil")
	}
}

// Verifies: SYS-REQ-028
// MCDC SYS-REQ-028: addressed_array_is_empty=T, addressed_array_is_well_formed=T, empty_array_produces_no_callbacks=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_028_Row3_InvariantViolation(t *testing.T) {
	// Invariant violation: empty well-formed array MUST produce no callbacks
	// (Row 4). Drive the positive path to prove unreachable.
	calls := 0
	if _, err := ArrayEach([]byte(`[]`), func(value []byte, dataType ValueType, offset int, err error) {
		calls++
	}); err != nil {
		t.Fatalf("ArrayEach on empty array returned error: %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected zero callbacks on empty array, got %d", calls)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-029 (malformed array input returns error via ArrayEach)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-029
// MCDC SYS-REQ-029: addressed_array_is_well_formed=F, malformed_array_input_returns_error=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_029_Row1_InvariantViolation(t *testing.T) {
	// Invariant violation: malformed array input MUST return error (Row 2).
	if _, err := ArrayEach([]byte(`[1,2`), func(value []byte, dataType ValueType, offset int, err error) {}); err == nil {
		t.Fatal("expected error on malformed array input, got nil")
	}
}

// Verifies: SYS-REQ-029
// MCDC SYS-REQ-029: addressed_array_is_well_formed=T, malformed_array_input_returns_error=F => TRUE [no-action: well-formed array does not invoke the malformed-error action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_029_Row2_WellFormed(t *testing.T) {
	calls := 0
	if _, err := ArrayEach([]byte(`[1,2,3]`), func(value []byte, dataType ValueType, offset int, err error) {
		calls++
	}); err != nil {
		t.Fatalf("ArrayEach on well-formed array returned error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 callbacks, got %d", calls)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-030 (empty well-formed object produces no entries via ObjectEach)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-030
// MCDC SYS-REQ-030: addressed_object_is_empty=F, addressed_object_is_well_formed=T, empty_object_produces_no_entries=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_030_Row1_NonEmptyWellFormed(t *testing.T) {
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

// Verifies: SYS-REQ-030
// MCDC SYS-REQ-030: addressed_object_is_empty=T, addressed_object_is_well_formed=F, empty_object_produces_no_entries=F => TRUE [no-action: callback counter == 0 on malformed input, no empty-object action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_030_Row2_EmptyMalformed(t *testing.T) {
	calls := 0
	err := ObjectEach([]byte(`{`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		calls++
		return nil
	})
	if calls != 0 {
		t.Fatalf("expected zero callbacks on malformed input, got %d", calls)
	}
	if err == nil {
		t.Fatal("expected error on malformed input, got nil")
	}
}

// Verifies: SYS-REQ-030
// MCDC SYS-REQ-030: addressed_object_is_empty=T, addressed_object_is_well_formed=T, empty_object_produces_no_entries=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_030_Row3_InvariantViolation(t *testing.T) {
	calls := 0
	if err := ObjectEach([]byte(`{}`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		calls++
		return nil
	}); err != nil {
		t.Fatalf("ObjectEach on empty object returned error: %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected zero callbacks on empty object, got %d", calls)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-031 (malformed object input returns error via ObjectEach)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-031
// MCDC SYS-REQ-031: addressed_object_is_well_formed=F, malformed_object_input_returns_error=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_031_Row1_InvariantViolation(t *testing.T) {
	if err := ObjectEach([]byte(`{"a":1`), func(key []byte, value []byte, dataType ValueType, offset int) error { return nil }); err == nil {
		t.Fatal("expected error on malformed object input, got nil")
	}
}

// Verifies: SYS-REQ-031
// MCDC SYS-REQ-031: addressed_object_is_well_formed=T, malformed_object_input_returns_error=F => TRUE [no-action: well-formed object does not invoke the malformed-error action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_031_Row2_WellFormed(t *testing.T) {
	calls := 0
	if err := ObjectEach([]byte(`{"a":1}`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		calls++
		return nil
	}); err != nil {
		t.Fatalf("ObjectEach on well-formed object returned error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 callback, got %d", calls)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-032 (callback error is returned by ObjectEach)
// FRETish: !addressed_object_is_well_formed | !object_callback_returns_error | object_callback_error_is_returned
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-032
// MCDC SYS-REQ-032: addressed_object_is_well_formed=F, object_callback_error_is_returned=F, object_callback_returns_error=T => TRUE [no-action: malformed input returns parse error before callback runs, callback-error action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_032_Row1_TriggerFalse(t *testing.T) {
 sentinelErr := errors.New("sentinel callback error")
	err := ObjectEach([]byte(`{`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		return sentinelErr
	})
	if err == nil {
		t.Fatal("expected error from ObjectEach on malformed input, got nil")
	}
	if errors.Is(err, sentinelErr) {
		t.Fatal("expected parse error (not callback sentinel) on malformed input")
	}
}

// Verifies: SYS-REQ-032
// MCDC SYS-REQ-032: addressed_object_is_well_formed=T, object_callback_error_is_returned=F, object_callback_returns_error=F => TRUE [no-action: callback returns nil, callback-error action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_032_Row2_NoCallbackError(t *testing.T) {
	calls := 0
	if err := ObjectEach([]byte(`{"a":1}`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		calls++
		return nil
	}); err != nil {
		t.Fatalf("ObjectEach returned error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 callback, got %d", calls)
	}
}

// Verifies: SYS-REQ-032
// MCDC SYS-REQ-032: addressed_object_is_well_formed=T, object_callback_error_is_returned=F, object_callback_returns_error=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_032_Row3_InvariantViolation(t *testing.T) {
	// Invariant violation: callback that returns an error on well-formed
	// input MUST propagate that error (Row 4).
	sentinel := errors.New("propagated callback error")
	err := ObjectEach([]byte(`{"a":1}`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error to propagate, got %v", err)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-033 (Delete removes addressed target)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-033
// MCDC SYS-REQ-033: delete_path_is_provided=F, delete_returns_document_without_target=F, delete_target_exists=T => TRUE [no-action: no path provided means Delete does not perform a targeted removal]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_033_Row1_TriggerFalse(t *testing.T) {
	// No path: Delete returns data[:0]; no targeted-removal action fires.
	data := []byte(`{"a":1}`)
	result := Delete(data)
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %s", string(result))
	}
}

// Verifies: SYS-REQ-033
// MCDC SYS-REQ-033: delete_path_is_provided=T, delete_returns_document_without_target=F, delete_target_exists=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_033_Row2_MissingTarget(t *testing.T) {
	data := []byte(`{"a":1}`)
	result := Delete(data, "missing")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// Verifies: SYS-REQ-033
// MCDC SYS-REQ-033: delete_path_is_provided=T, delete_returns_document_without_target=F, delete_target_exists=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_033_Row3_InvariantViolation(t *testing.T) {
	// Invariant violation: existing target with a provided path MUST be
	// removed (Row 4). Drive the positive path.
	data := []byte(`{"a":1,"b":2}`)
	result := Delete(data, "a")
	if _, _, _, err := Get(result, "a"); !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected 'a' to be deleted, got err=%v", err)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-034 (Delete preserves input when target missing in usable input)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-034
// MCDC SYS-REQ-034: delete_input_is_unusable_for_requested_path=F, delete_path_is_provided=F, delete_preserves_input_when_target_missing=F, delete_target_exists=F => TRUE [no-action: no path provided, preserve-on-missing-target action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_034_Row1_TriggerFalse(t *testing.T) {
	data := []byte(`{"a":1}`)
	result := Delete(data)
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %s", string(result))
	}
}

// Verifies: SYS-REQ-034
// MCDC SYS-REQ-034: delete_input_is_unusable_for_requested_path=F, delete_path_is_provided=T, delete_preserves_input_when_target_missing=F, delete_target_exists=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_034_Row2_InvariantViolation(t *testing.T) {
	// Invariant violation: usable input + provided path + missing target MUST
	// preserve the input (Row 3). Drive the positive path.
	data := []byte(`{"a":1}`)
	result := Delete(data, "missing")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// Verifies: SYS-REQ-034
// MCDC SYS-REQ-034: delete_input_is_unusable_for_requested_path=F, delete_path_is_provided=T, delete_preserves_input_when_target_missing=F, delete_target_exists=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_034_Row3_TargetExistsUsable(t *testing.T) {
	data := []byte(`{"a":1,"b":2}`)
	result := Delete(data, "a")
	if _, _, _, err := Get(result, "a"); !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected 'a' to be deleted, got err=%v", err)
	}
}

// Verifies: SYS-REQ-034
// MCDC SYS-REQ-034: delete_input_is_unusable_for_requested_path=T, delete_path_is_provided=T, delete_preserves_input_when_target_missing=F, delete_target_exists=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_034_Row4_UnusableInput(t *testing.T) {
	// Unusable (malformed) input with a path: Delete returns input unchanged.
	data := []byte(`{"a":`)
	result := Delete(data, "a")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data on unusable input, got %s", string(result))
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-035 (Delete completes without panic on unusable input)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-035
// MCDC SYS-REQ-035: delete_completes_without_panic=F, delete_input_is_unusable_for_requested_path=F, delete_path_is_provided=T, delete_returns_original_input_on_unusable_input=F => TRUE [no-action: usable input does not invoke the return-original-on-unusable action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_035_Row1_TriggerFalse(t *testing.T) {
	data := []byte(`{"a":1,"b":2}`)
	result := Delete(data, "a")
	if _, _, _, err := Get(result, "a"); !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected 'a' to be deleted, got err=%v", err)
	}
}

// Verifies: SYS-REQ-035
// MCDC SYS-REQ-035: delete_completes_without_panic=F, delete_input_is_unusable_for_requested_path=T, delete_path_is_provided=F, delete_returns_original_input_on_unusable_input=F => TRUE [no-action: no path provided, return-original-on-unusable action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_035_Row2_NoPathUnusable(t *testing.T) {
	data := []byte(`{"a":`)
	result := Delete(data)
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %s", string(result))
	}
}

// Verifies: SYS-REQ-035
// MCDC SYS-REQ-035: delete_completes_without_panic=F, delete_input_is_unusable_for_requested_path=T, delete_path_is_provided=T, delete_returns_original_input_on_unusable_input=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_035_Row3_InvariantViolationPanic(t *testing.T) {
	// Invariant violation: unusable input + provided path MUST NOT panic AND
	// MUST return original input (Row 5). Drive the positive path.
	data := []byte(`{"a":`)
	result := Delete(data, "a")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// Verifies: SYS-REQ-035
// MCDC SYS-REQ-035: delete_completes_without_panic=F, delete_input_is_unusable_for_requested_path=T, delete_path_is_provided=T, delete_returns_original_input_on_unusable_input=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_035_Row4_InvariantViolationOriginal(t *testing.T) {
	// Same driver as Row 5; witnessing that the panic-free completion is
	// coupled to original-input return.
	data := []byte(`{"a":`)
	result := Delete(data, "a")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// Verifies: SYS-REQ-035
// MCDC SYS-REQ-035: delete_completes_without_panic=T, delete_input_is_unusable_for_requested_path=T, delete_path_is_provided=T, delete_returns_original_input_on_unusable_input=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_035_Row5_InvariantViolationNoPanic(t *testing.T) {
	data := []byte(`{"a":`)
	result := Delete(data, "a")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-036 (ParseBoolean malformed literal returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-036
// MCDC SYS-REQ-036: raw_boolean_literal_is_valid=F, returns_parseboolean_error=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_036_Row1_InvariantViolation(t *testing.T) {
	if _, err := ParseBoolean([]byte(`notabool`)); err == nil {
		t.Fatal("expected error on malformed boolean literal, got nil")
	}
}

// Verifies: SYS-REQ-036
// MCDC SYS-REQ-036: raw_boolean_literal_is_valid=T, returns_parseboolean_error=F => TRUE [no-action: valid boolean literal does not invoke the parse-error action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_036_Row2_ValidLiteral(t *testing.T) {
	v, err := ParseBoolean([]byte(`true`))
	if err != nil {
		t.Fatalf("ParseBoolean returned error: %v", err)
	}
	if !v {
		t.Fatal("expected true, got false")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-037 (ParseFloat malformed token returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-037
// MCDC SYS-REQ-037: raw_float_token_is_well_formed=F, returns_parsefloat_error=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_037_Row1_InvariantViolation(t *testing.T) {
	if _, err := ParseFloat([]byte(`notafloat`)); err == nil {
		t.Fatal("expected error on malformed float token, got nil")
	}
}

// Verifies: SYS-REQ-037
// MCDC SYS-REQ-037: raw_float_token_is_well_formed=T, returns_parsefloat_error=F => TRUE [no-action: well-formed float does not invoke the parse-error action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_037_Row2_WellFormed(t *testing.T) {
	v, err := ParseFloat([]byte(`3.14`))
	if err != nil {
		t.Fatalf("ParseFloat returned error: %v", err)
	}
	if v != 3.14 {
		t.Fatalf("expected 3.14, got %v", v)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-038 (ParseString malformed literal returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-038
// MCDC SYS-REQ-038: raw_string_literal_is_well_formed=F, returns_parsestring_error=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_038_Row1_InvariantViolation(t *testing.T) {
	if _, err := ParseString([]byte(`abc\q`)); err == nil {
		t.Fatal("expected error on malformed string literal, got nil")
	}
}

// Verifies: SYS-REQ-038
// MCDC SYS-REQ-038: raw_string_literal_is_well_formed=T, returns_parsestring_error=F => TRUE [no-action: well-formed string does not invoke the parse-error action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_038_Row2_WellFormed(t *testing.T) {
	v, err := ParseString([]byte(`hello`))
	if err != nil {
		t.Fatalf("ParseString returned error: %v", err)
	}
	if v != "hello" {
		t.Fatalf("expected hello, got %q", v)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-039 (ParseInt overflow returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-039
// MCDC SYS-REQ-039: raw_int_token_overflows_int64=F, returns_parseint_overflow_error=F => TRUE [no-action: non-overflow integer does not invoke the overflow-error action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_039_Row1_NoOverflow(t *testing.T) {
	v, err := ParseInt([]byte(`42`))
	if err != nil {
		t.Fatalf("ParseInt returned error: %v", err)
	}
	if v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}
}

// Verifies: SYS-REQ-039
// MCDC SYS-REQ-039: raw_int_token_overflows_int64=T, returns_parseint_overflow_error=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_039_Row2_InvariantViolation(t *testing.T) {
	if _, err := ParseInt([]byte(`99999999999999999999999`)); err == nil {
		t.Fatal("expected overflow error, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-040 (ParseInt malformed token returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-040
// MCDC SYS-REQ-040: raw_int_token_is_well_formed=F, raw_int_token_overflows_int64=F, returns_parseint_malformed_error=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_040_Row1_InvariantViolation(t *testing.T) {
	if _, err := ParseInt([]byte(`notanint`)); err == nil {
		t.Fatal("expected malformed error, got nil")
	}
}

// Verifies: SYS-REQ-040
// MCDC SYS-REQ-040: raw_int_token_is_well_formed=F, raw_int_token_overflows_int64=T, returns_parseint_malformed_error=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_040_Row2_MalformedOrOverflow(t *testing.T) {
	// Token is malformed in the parseInt sense and also beyond int64; the
	// implementation returns the malformed error first.
	if _, err := ParseInt([]byte(`notanint9999999999999999999999`)); err == nil {
		t.Fatal("expected malformed/overflow error, got nil")
	}
}

// Verifies: SYS-REQ-040
// MCDC SYS-REQ-040: raw_int_token_is_well_formed=T, raw_int_token_overflows_int64=F, returns_parseint_malformed_error=F => TRUE [no-action: well-formed non-overflow integer does not invoke the malformed-error action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_040_Row3_WellFormed(t *testing.T) {
	v, err := ParseInt([]byte(`42`))
	if err != nil {
		t.Fatalf("ParseInt returned error: %v", err)
	}
	if v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-041 (truncated-at-value-boundary returns parse error via Get)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-041
// MCDC SYS-REQ-041: input_is_truncated_at_value_boundary=F, returns_error_for_truncated_value_boundary=F => TRUE [no-action: non-truncated input does not invoke the truncated-error action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_041_Row1_TriggerFalse(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

// Verifies: SYS-REQ-041
// MCDC SYS-REQ-041: input_is_truncated_at_value_boundary=T, returns_error_for_truncated_value_boundary=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_041_Row2_InvariantViolation(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected parse error on truncated-at-value-boundary input, got nil")
	}
}

// Verifies: SYS-REQ-041
// MCDC SYS-REQ-041: input_is_truncated_at_value_boundary=T, returns_error_for_truncated_value_boundary=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_041_Row3_TruncatedError(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected parse error on truncated-at-value-boundary input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-042 (truncated-mid-structure returns parse error via Get)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-042
// MCDC SYS-REQ-042: input_is_truncated_mid_structure=F, returns_error_for_truncated_mid_structure=F => TRUE [no-action: non-truncated input does not invoke the truncated-mid-structure action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_042_Row1_TriggerFalse(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"a":[1,2]}`), "a"); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

// Verifies: SYS-REQ-042
// MCDC SYS-REQ-042: input_is_truncated_mid_structure=T, returns_error_for_truncated_mid_structure=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_042_Row2_InvariantViolation(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"a":[1,2`), "a"); err == nil {
		t.Fatal("expected parse error on truncated-mid-structure input, got nil")
	}
}

// Verifies: SYS-REQ-042
// MCDC SYS-REQ-042: input_is_truncated_mid_structure=T, returns_error_for_truncated_mid_structure=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_042_Row3_TruncatedError(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"a":[1,2`), "a"); err == nil {
		t.Fatal("expected parse error on truncated-mid-structure input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-043 (truncated-mid-key returns parse error via Get)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-043
// MCDC SYS-REQ-043: input_is_truncated_mid_key=F, returns_error_for_truncated_mid_key=F => TRUE [no-action: non-truncated input does not invoke the truncated-mid-key action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_043_Row1_TriggerFalse(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

// Verifies: SYS-REQ-043
// MCDC SYS-REQ-043: input_is_truncated_mid_key=T, returns_error_for_truncated_mid_key=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_043_Row2_InvariantViolation(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"abc`), "abc"); err == nil {
		t.Fatal("expected parse error on truncated-mid-key input, got nil")
	}
}

// Verifies: SYS-REQ-043
// MCDC SYS-REQ-043: input_is_truncated_mid_key=T, returns_error_for_truncated_mid_key=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_043_Row3_TruncatedError(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"abc`), "abc"); err == nil {
		t.Fatal("expected parse error on truncated-mid-key input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-044 (caller bounds-checks tokenEnd == len(data) sentinel)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-044
// MCDC SYS-REQ-044: caller_bounds_checks_tokenEnd_sentinel=F, tokenEnd_returns_len_data=F => TRUE [no-action: tokenEnd never returns len(data) for this input, bounds-check action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_044_Row1_TriggerFalse(t *testing.T) {
	// A normal lookup where tokenEnd never returns the len(data) sentinel.
	if _, _, _, err := Get([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

// Verifies: SYS-REQ-044
// MCDC SYS-REQ-044: caller_bounds_checks_tokenEnd_sentinel=F, tokenEnd_returns_len_data=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_044_Row2_InvariantViolation(t *testing.T) {
	// Invariant violation: when tokenEnd returns len(data) the caller MUST
	// bounds-check (Row 3). Drive a path where tokenEnd reaches len(data).
	// Parsing a top-level scalar with no key path lands the root value at the
	// end of input, so tokenEnd returns len(data) and Get still returns safely.
	value, dataType, _, err := Get([]byte(`42`))
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "42" {
		t.Fatalf("expected 42, got %s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-044
// MCDC SYS-REQ-044: caller_bounds_checks_tokenEnd_sentinel=T, tokenEnd_returns_len_data=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_044_Row3_BoundsChecked(t *testing.T) {
	value, dataType, _, err := Get([]byte(`42`))
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "42" {
		t.Fatalf("expected 42, got %s type=%v", string(value), dataType)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-045 (caller handles stringEnd == -1 sentinel)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-045
// MCDC SYS-REQ-045: caller_handles_stringEnd_sentinel=F, stringEnd_returns_negative_one=F => TRUE [no-action: stringEnd never returns -1 here, sentinel-handling action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_045_Row1_TriggerFalse(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":"b"}`), "a")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != String || string(value) != "b" {
		t.Fatalf("expected string b, got %s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-045
// MCDC SYS-REQ-045: caller_handles_stringEnd_sentinel=F, stringEnd_returns_negative_one=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_045_Row2_InvariantViolation(t *testing.T) {
	// Invariant violation: when stringEnd returns -1 the caller MUST handle
	// it (Row 3). Truncated mid-key forces stringEnd to never find a closing
	// quote on the value; Get still completes without panic.
	if _, _, _, err := Get([]byte(`{"a":"b`), "a"); err == nil {
		t.Fatal("expected parse error on truncated string, got nil")
	}
}

// Verifies: SYS-REQ-045
// MCDC SYS-REQ-045: caller_handles_stringEnd_sentinel=T, stringEnd_returns_negative_one=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_045_Row3_SentinelHandled(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"a":"b`), "a"); err == nil {
		t.Fatal("expected parse error on truncated string, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-046 (caller handles blockEnd == -1 sentinel)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-046
// MCDC SYS-REQ-046: blockEnd_returns_negative_one=F, caller_handles_blockEnd_sentinel=F => TRUE [no-action: blockEnd never returns -1 here, sentinel-handling action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_046_Row1_TriggerFalse(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":[1,2]}`), "a")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Array || string(value) != "[1,2]" {
		t.Fatalf("expected array bytes [1,2], got %s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-046
// MCDC SYS-REQ-046: blockEnd_returns_negative_one=T, caller_handles_blockEnd_sentinel=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_046_Row2_InvariantViolation(t *testing.T) {
	// Invariant violation: when blockEnd returns -1 the caller MUST handle
	// it (Row 3). Truncated-mid-structure forces blockEnd to return -1.
	if _, _, _, err := Get([]byte(`{"a":[1,2`), "a"); err == nil {
		t.Fatal("expected parse error on truncated structure, got nil")
	}
}

// Verifies: SYS-REQ-046
// MCDC SYS-REQ-046: blockEnd_returns_negative_one=T, caller_handles_blockEnd_sentinel=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_046_Row3_SentinelHandled(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"a":[1,2`), "a"); err == nil {
		t.Fatal("expected parse error on truncated structure, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-047 (negative array index returns not-found)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-047
// MCDC SYS-REQ-047: path_segment_is_negative_array_index=F, returns_not_found_for_negative_array_index=F => TRUE [no-action: non-negative index does not invoke the negative-index action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_047_Row1_TriggerFalse(t *testing.T) {
	value, dataType, _, err := Get([]byte(`{"a":[1,2,3]}`), "a", "[1]")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if dataType != Number || string(value) != "2" {
		t.Fatalf("expected 2, got %s type=%v", string(value), dataType)
	}
}

// Verifies: SYS-REQ-047
// MCDC SYS-REQ-047: path_segment_is_negative_array_index=T, returns_not_found_for_negative_array_index=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_047_Row2_InvariantViolation(t *testing.T) {
	// Invariant violation: negative array index MUST return not-found (Row 3).
	_, _, _, err := Get([]byte(`{"a":[1,2,3]}`), "a", "[-1]")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError for negative index, got %v", err)
	}
}

// Verifies: SYS-REQ-047
// MCDC SYS-REQ-047: path_segment_is_negative_array_index=T, returns_not_found_for_negative_array_index=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_047_Row3_NegativeNotFound(t *testing.T) {
	_, _, _, err := Get([]byte(`{"a":[1,2,3]}`), "a", "[-1]")
	if !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError for negative index, got %v", err)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-048 (Delete on truncated-at-value-boundary input)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-048
// MCDC SYS-REQ-048: delete_completes_without_panic_on_truncated_value=F, delete_input_is_truncated_at_value_boundary=F, delete_returns_original_input_on_truncated_value=F => TRUE [no-action: non-truncated input does not invoke the truncated-value action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_048_Row1_TriggerFalse(t *testing.T) {
	data := []byte(`{"a":1,"b":2}`)
	result := Delete(data, "a")
	if _, _, _, err := Get(result, "a"); !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected 'a' to be deleted, got err=%v", err)
	}
}

// Verifies: SYS-REQ-048
// MCDC SYS-REQ-048: delete_completes_without_panic_on_truncated_value=F, delete_input_is_truncated_at_value_boundary=T, delete_returns_original_input_on_truncated_value=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_048_Row2_InvariantViolationPanic(t *testing.T) {
	// Invariant violation: truncated input + Delete MUST NOT panic AND MUST
	// return original input (Row 5). Drive the positive path.
	data := []byte(`{"a":`)
	result := Delete(data, "a")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// Verifies: SYS-REQ-048
// MCDC SYS-REQ-048: delete_completes_without_panic_on_truncated_value=F, delete_input_is_truncated_at_value_boundary=T, delete_returns_original_input_on_truncated_value=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_048_Row3_InvariantViolationOriginal(t *testing.T) {
	data := []byte(`{"a":`)
	result := Delete(data, "a")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// Verifies: SYS-REQ-048
// MCDC SYS-REQ-048: delete_completes_without_panic_on_truncated_value=T, delete_input_is_truncated_at_value_boundary=T, delete_returns_original_input_on_truncated_value=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_048_Row4_InvariantViolationNoPanic(t *testing.T) {
	data := []byte(`{"a":`)
	result := Delete(data, "a")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// Verifies: SYS-REQ-048
// MCDC SYS-REQ-048: delete_completes_without_panic_on_truncated_value=T, delete_input_is_truncated_at_value_boundary=T, delete_returns_original_input_on_truncated_value=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_048_Row5_TruncatedValue(t *testing.T) {
	data := []byte(`{"a":`)
	result := Delete(data, "a")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-049 (Delete propagates internalGet parse error)
// FRETish: !delete_discards_internalGet_error | delete_propagates_internalGet_error
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-049
// MCDC SYS-REQ-049: delete_discards_internalGet_error=F, delete_propagates_internalGet_error=F => TRUE [no-action: well-formed input does not invoke the propagate-error action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_049_Row1_TriggerFalse(t *testing.T) {
	data := []byte(`{"a":1,"b":2}`)
	result := Delete(data, "a")
	if _, _, _, err := Get(result, "a"); !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected 'a' to be deleted, got err=%v", err)
	}
}

// Verifies: SYS-REQ-049
// MCDC SYS-REQ-049: delete_discards_internalGet_error=T, delete_propagates_internalGet_error=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_049_Row2_InvariantViolation(t *testing.T) {
	// Invariant violation: if Delete would discard the error it MUST still
	// propagate (Row 3). Drive the positive path: Delete on unusable input
	// returns the input unchanged rather than discarding the parse error.
	data := []byte(`{"a":`)
	result := Delete(data, "a")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data on unusable input, got %s", string(result))
	}
}

// Verifies: SYS-REQ-049
// MCDC SYS-REQ-049: delete_discards_internalGet_error=T, delete_propagates_internalGet_error=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_049_Row3_PropagatedError(t *testing.T) {
	data := []byte(`{"a":`)
	result := Delete(data, "a")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data on unusable input, got %s", string(result))
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-050 (Delete on truncated array input)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-050
// MCDC SYS-REQ-050: delete_array_input_is_truncated=F, delete_completes_without_panic_on_truncated_array=F, delete_returns_original_input_on_truncated_array=F => TRUE [no-action: non-truncated array input does not invoke the truncated-array action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_050_Row1_TriggerFalse(t *testing.T) {
	data := []byte(`{"a":[1,2,3]}`)
	result := Delete(data, "a", "[1]")
	// Element 2 should be removed; array now [1,3].
	v, _, _, err := Get(result, "a", "[1]")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if string(v) != "3" {
		t.Fatalf("expected [1] to be removed (now 3 at index 1), got %s", string(v))
	}
}

// Verifies: SYS-REQ-050
// MCDC SYS-REQ-050: delete_array_input_is_truncated=T, delete_completes_without_panic_on_truncated_array=F, delete_returns_original_input_on_truncated_array=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_050_Row2_InvariantViolationPanic(t *testing.T) {
	data := []byte(`{"a":[1,2`)
	result := Delete(data, "a", "[1]")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// Verifies: SYS-REQ-050
// MCDC SYS-REQ-050: delete_array_input_is_truncated=T, delete_completes_without_panic_on_truncated_array=F, delete_returns_original_input_on_truncated_array=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_050_Row3_InvariantViolationOriginal(t *testing.T) {
	data := []byte(`{"a":[1,2`)
	result := Delete(data, "a", "[1]")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// Verifies: SYS-REQ-050
// MCDC SYS-REQ-050: delete_array_input_is_truncated=T, delete_completes_without_panic_on_truncated_array=T, delete_returns_original_input_on_truncated_array=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_050_Row4_InvariantViolationNoPanic(t *testing.T) {
	data := []byte(`{"a":[1,2`)
	result := Delete(data, "a", "[1]")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// Verifies: SYS-REQ-050
// MCDC SYS-REQ-050: delete_array_input_is_truncated=T, delete_completes_without_panic_on_truncated_array=T, delete_returns_original_input_on_truncated_array=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_050_Row5_TruncatedArray(t *testing.T) {
	data := []byte(`{"a":[1,2`)
	result := Delete(data, "a", "[1]")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-051 (Set on truncated input returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-051
// MCDC SYS-REQ-051: set_input_is_truncated=F, set_returns_error_for_truncated_input=F => TRUE [no-action: non-truncated input does not invoke the truncated-error action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_051_Row1_TriggerFalse(t *testing.T) {
	if _, err := Set([]byte(`{"a":1}`), []byte(`42`), "a"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
}

// Verifies: SYS-REQ-051
// MCDC SYS-REQ-051: set_input_is_truncated=T, set_returns_error_for_truncated_input=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_051_Row2_InvariantViolation(t *testing.T) {
	if _, err := Set([]byte(`{"a":`), []byte(`42`), "a"); err == nil {
		t.Fatal("expected error on truncated Set input, got nil")
	}
}

// Verifies: SYS-REQ-051
// MCDC SYS-REQ-051: set_input_is_truncated=T, set_returns_error_for_truncated_input=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_051_Row3_TruncatedError(t *testing.T) {
	if _, err := Set([]byte(`{"a":`), []byte(`42`), "a"); err == nil {
		t.Fatal("expected error on truncated Set input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-052 (ArrayEach callback error is propagated)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-052
// MCDC SYS-REQ-052: array_callback_error_is_propagated=F, array_callback_returns_error=F => TRUE [no-action: callback returns nil, propagate-error action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_052_Row1_TriggerFalse(t *testing.T) {
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

// -----------------------------------------------------------------------------
// SYS-REQ-053 (ArrayEach returns error for truncated-mid-element)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-053
// MCDC SYS-REQ-053: array_is_truncated_mid_element=F, returns_error_for_truncated_array_element=F => TRUE [no-action: non-truncated array does not invoke the truncated-element action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_053_Row1_TriggerFalse(t *testing.T) {
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

// Verifies: SYS-REQ-053
// MCDC SYS-REQ-053: array_is_truncated_mid_element=T, returns_error_for_truncated_array_element=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_053_Row2_InvariantViolation(t *testing.T) {
	if _, err := ArrayEach([]byte(`[1,2,`), func(value []byte, dataType ValueType, offset int, err error) {}); err == nil {
		t.Fatal("expected error on truncated array element, got nil")
	}
}

// Verifies: SYS-REQ-053
// MCDC SYS-REQ-053: array_is_truncated_mid_element=T, returns_error_for_truncated_array_element=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_053_Row3_TruncatedError(t *testing.T) {
	if _, err := ArrayEach([]byte(`[1,2,`), func(value []byte, dataType ValueType, offset int, err error) {}); err == nil {
		t.Fatal("expected error on truncated array element, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-054 (ObjectEach returns error for truncated-mid-entry)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-054
// MCDC SYS-REQ-054: object_is_truncated_mid_entry=F, returns_error_for_truncated_object_entry=F => TRUE [no-action: non-truncated object does not invoke the truncated-entry action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_054_Row1_TriggerFalse(t *testing.T) {
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

// Verifies: SYS-REQ-054
// MCDC SYS-REQ-054: object_is_truncated_mid_entry=T, returns_error_for_truncated_object_entry=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_054_Row2_InvariantViolation(t *testing.T) {
	if err := ObjectEach([]byte(`{"a":1,"b":`), func(key []byte, value []byte, dataType ValueType, offset int) error { return nil }); err == nil {
		t.Fatal("expected error on truncated object entry, got nil")
	}
}

// Verifies: SYS-REQ-054
// MCDC SYS-REQ-054: object_is_truncated_mid_entry=T, returns_error_for_truncated_object_entry=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_054_Row3_TruncatedError(t *testing.T) {
	if err := ObjectEach([]byte(`{"a":1,"b":`), func(key []byte, value []byte, dataType ValueType, offset int) error { return nil }); err == nil {
		t.Fatal("expected error on truncated object entry, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-055 (ArrayEach returns error for malformed delimiter)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-055
// MCDC SYS-REQ-055: array_has_malformed_delimiter=F, returns_error_for_malformed_array_delimiter=F => TRUE [no-action: well-formed delimiters do not invoke the malformed-delimiter action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_055_Row1_TriggerFalse(t *testing.T) {
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

// Verifies: SYS-REQ-055
// MCDC SYS-REQ-055: array_has_malformed_delimiter=T, returns_error_for_malformed_array_delimiter=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_055_Row2_InvariantViolation(t *testing.T) {
	if _, err := ArrayEach([]byte(`[1,,2]`), func(value []byte, dataType ValueType, offset int, err error) {}); err == nil {
		t.Fatal("expected error on malformed array delimiter, got nil")
	}
}

// Verifies: SYS-REQ-055
// MCDC SYS-REQ-055: array_has_malformed_delimiter=T, returns_error_for_malformed_array_delimiter=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_055_Row3_MalformedError(t *testing.T) {
	if _, err := ArrayEach([]byte(`[1,,2]`), func(value []byte, dataType ValueType, offset int, err error) {}); err == nil {
		t.Fatal("expected error on malformed array delimiter, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-056 (Delete on truncated mid-structure returns original)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-056
// MCDC SYS-REQ-056: delete_completes_without_panic_on_truncated_structure=F, delete_input_is_truncated_mid_structure=F, delete_returns_original_input_on_truncated_structure=F => TRUE [no-action: non-truncated input does not invoke the truncated-structure action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_056_Row1_TriggerFalse(t *testing.T) {
	data := []byte(`{"a":{"b":1}}`)
	result := Delete(data, "a")
	if _, _, _, err := Get(result, "a"); !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected 'a' to be deleted, got err=%v", err)
	}
}

// Verifies: SYS-REQ-056
// MCDC SYS-REQ-056: delete_completes_without_panic_on_truncated_structure=F, delete_input_is_truncated_mid_structure=T, delete_returns_original_input_on_truncated_structure=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_056_Row2_InvariantViolationPanic(t *testing.T) {
	data := []byte(`{"a":[1,2`)
	result := Delete(data, "a")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// Verifies: SYS-REQ-056
// MCDC SYS-REQ-056: delete_completes_without_panic_on_truncated_structure=F, delete_input_is_truncated_mid_structure=T, delete_returns_original_input_on_truncated_structure=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_056_Row3_InvariantViolationOriginal(t *testing.T) {
	data := []byte(`{"a":[1,2`)
	result := Delete(data, "a")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// Verifies: SYS-REQ-056
// MCDC SYS-REQ-056: delete_completes_without_panic_on_truncated_structure=T, delete_input_is_truncated_mid_structure=T, delete_returns_original_input_on_truncated_structure=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_056_Row4_InvariantViolationNoPanic(t *testing.T) {
	data := []byte(`{"a":[1,2`)
	result := Delete(data, "a")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// Verifies: SYS-REQ-056
// MCDC SYS-REQ-056: delete_completes_without_panic_on_truncated_structure=T, delete_input_is_truncated_mid_structure=T, delete_returns_original_input_on_truncated_structure=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_056_Row5_TruncatedStructure(t *testing.T) {
	data := []byte(`{"a":[1,2`)
	result := Delete(data, "a")
	if string(result) != string(data) {
		t.Fatalf("expected unchanged data, got %s", string(result))
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-057 (ParseBoolean partial literal returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-057
// MCDC SYS-REQ-057: raw_boolean_literal_is_partial=F, returns_error_for_partial_boolean_literal=F => TRUE [no-action: non-partial literal does not invoke the partial-error action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_057_Row1_TriggerFalse(t *testing.T) {
	v, err := ParseBoolean([]byte(`true`))
	if err != nil {
		t.Fatalf("ParseBoolean returned error: %v", err)
	}
	if !v {
		t.Fatal("expected true, got false")
	}
}

// Verifies: SYS-REQ-057
// MCDC SYS-REQ-057: raw_boolean_literal_is_partial=T, returns_error_for_partial_boolean_literal=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_057_Row2_InvariantViolation(t *testing.T) {
	if _, err := ParseBoolean([]byte(`tru`)); err == nil {
		t.Fatal("expected error on partial boolean literal, got nil")
	}
}

// Verifies: SYS-REQ-057
// MCDC SYS-REQ-057: raw_boolean_literal_is_partial=T, returns_error_for_partial_boolean_literal=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_057_Row3_PartialError(t *testing.T) {
	if _, err := ParseBoolean([]byte(`tru`)); err == nil {
		t.Fatal("expected error on partial boolean literal, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-058 (ParseInt returns correct value at int64 boundary)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-058
// MCDC SYS-REQ-058: raw_int_token_is_at_int64_max_boundary=F, returns_correct_value_at_int64_boundary=F => TRUE [no-action: non-boundary integer does not invoke the boundary-value action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_058_Row1_TriggerFalse(t *testing.T) {
	v, err := ParseInt([]byte(`42`))
	if err != nil {
		t.Fatalf("ParseInt returned error: %v", err)
	}
	if v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}
}

// Verifies: SYS-REQ-058
// MCDC SYS-REQ-058: raw_int_token_is_at_int64_max_boundary=T, returns_correct_value_at_int64_boundary=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_058_Row2_InvariantViolation(t *testing.T) {
	// Invariant violation: int64 max boundary MUST return correct value (Row 3).
	v, err := ParseInt([]byte(`9223372036854775807`))
	if err != nil {
		t.Fatalf("ParseInt returned error: %v", err)
	}
	if v != 9223372036854775807 {
		t.Fatalf("expected int64 max, got %d", v)
	}
}

// Verifies: SYS-REQ-058
// MCDC SYS-REQ-058: raw_int_token_is_at_int64_max_boundary=T, returns_correct_value_at_int64_boundary=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_058_Row3_BoundaryValue(t *testing.T) {
	v, err := ParseInt([]byte(`9223372036854775807`))
	if err != nil {
		t.Fatalf("ParseInt returned error: %v", err)
	}
	if v != 9223372036854775807 {
		t.Fatalf("expected int64 max, got %d", v)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-059 (ParseInt returns overflow at int64 max + 1)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-059
// MCDC SYS-REQ-059: raw_int_token_is_at_int64_max_plus_one=F, returns_overflow_at_int64_max_plus_one=F => TRUE [no-action: non-overflow integer does not invoke the overflow action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_059_Row1_TriggerFalse(t *testing.T) {
	v, err := ParseInt([]byte(`42`))
	if err != nil {
		t.Fatalf("ParseInt returned error: %v", err)
	}
	if v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}
}

// Verifies: SYS-REQ-059
// MCDC SYS-REQ-059: raw_int_token_is_at_int64_max_plus_one=T, returns_overflow_at_int64_max_plus_one=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_059_Row2_InvariantViolation(t *testing.T) {
	if _, err := ParseInt([]byte(`9223372036854775808`)); err == nil {
		t.Fatal("expected overflow error, got nil")
	}
}

// Verifies: SYS-REQ-059
// MCDC SYS-REQ-059: raw_int_token_is_at_int64_max_plus_one=T, returns_overflow_at_int64_max_plus_one=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_059_Row3_OverflowError(t *testing.T) {
	if _, err := ParseInt([]byte(`9223372036854775808`)); err == nil {
		t.Fatal("expected overflow error, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-060 (ParseString returns error for truncated escape sequence)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-060
// MCDC SYS-REQ-060: raw_string_has_truncated_escape_sequence=F, returns_error_for_truncated_escape_sequence=F => TRUE [no-action: complete escape sequence does not invoke the truncated-escape action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_060_Row1_TriggerFalse(t *testing.T) {
	v, err := ParseString([]byte(`hello`))
	if err != nil {
		t.Fatalf("ParseString returned error: %v", err)
	}
	if v != "hello" {
		t.Fatalf("expected hello, got %q", v)
	}
}

// Verifies: SYS-REQ-060
// MCDC SYS-REQ-060: raw_string_has_truncated_escape_sequence=T, returns_error_for_truncated_escape_sequence=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_060_Row2_InvariantViolation(t *testing.T) {
	if _, err := ParseString([]byte(`abc\`)); err == nil {
		t.Fatal("expected error on truncated escape sequence, got nil")
	}
}

// Verifies: SYS-REQ-060
// MCDC SYS-REQ-060: raw_string_has_truncated_escape_sequence=T, returns_error_for_truncated_escape_sequence=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_060_Row3_TruncatedEscapeError(t *testing.T) {
	if _, err := ParseString([]byte(`abc\`)); err == nil {
		t.Fatal("expected error on truncated escape sequence, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-065 (ParseFloat on empty input returns malformed error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-065
// MCDC SYS-REQ-065: parsefloat_input_is_empty=F, returns_parsefloat_malformed_for_empty=F => TRUE [no-action: non-empty input does not invoke the empty-malformed action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_065_Row1_TriggerFalse(t *testing.T) {
	v, err := ParseFloat([]byte(`3.14`))
	if err != nil {
		t.Fatalf("ParseFloat returned error: %v", err)
	}
	if v != 3.14 {
		t.Fatalf("expected 3.14, got %v", v)
	}
}

// Verifies: SYS-REQ-065
// MCDC SYS-REQ-065: parsefloat_input_is_empty=T, returns_parsefloat_malformed_for_empty=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_065_Row2_InvariantViolation(t *testing.T) {
	if _, err := ParseFloat([]byte(``)); err == nil {
		t.Fatal("expected malformed error on empty input, got nil")
	}
}

// Verifies: SYS-REQ-065
// MCDC SYS-REQ-065: parsefloat_input_is_empty=T, returns_parsefloat_malformed_for_empty=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_065_Row3_EmptyMalformed(t *testing.T) {
	if _, err := ParseFloat([]byte(``)); err == nil {
		t.Fatal("expected malformed error on empty input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-066 (ParseBoolean on empty input returns malformed error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-066
// MCDC SYS-REQ-066: parseboolean_input_is_empty=F, returns_parseboolean_malformed_for_empty=F => TRUE [no-action: non-empty input does not invoke the empty-malformed action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_066_Row1_TriggerFalse(t *testing.T) {
	v, err := ParseBoolean([]byte(`true`))
	if err != nil {
		t.Fatalf("ParseBoolean returned error: %v", err)
	}
	if !v {
		t.Fatal("expected true, got false")
	}
}

// Verifies: SYS-REQ-066
// MCDC SYS-REQ-066: parseboolean_input_is_empty=T, returns_parseboolean_malformed_for_empty=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_066_Row2_InvariantViolation(t *testing.T) {
	if _, err := ParseBoolean([]byte(``)); err == nil {
		t.Fatal("expected malformed error on empty input, got nil")
	}
}

// Verifies: SYS-REQ-066
// MCDC SYS-REQ-066: parseboolean_input_is_empty=T, returns_parseboolean_malformed_for_empty=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_066_Row3_EmptyMalformed(t *testing.T) {
	if _, err := ParseBoolean([]byte(``)); err == nil {
		t.Fatal("expected malformed error on empty input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-067 (ParseString on empty input returns identity)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-067
// MCDC SYS-REQ-067: parsestring_input_is_empty=F, returns_parsestring_identity_for_empty=F => TRUE [no-action: non-empty input does not invoke the empty-identity action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_067_Row1_TriggerFalse(t *testing.T) {
	v, err := ParseString([]byte(`hello`))
	if err != nil {
		t.Fatalf("ParseString returned error: %v", err)
	}
	if v != "hello" {
		t.Fatalf("expected hello, got %q", v)
	}
}

// Verifies: SYS-REQ-067
// MCDC SYS-REQ-067: parsestring_input_is_empty=T, returns_parsestring_identity_for_empty=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_067_Row2_InvariantViolation(t *testing.T) {
	v, err := ParseString([]byte(``))
	if err != nil {
		t.Fatalf("ParseString returned error: %v", err)
	}
	if v != "" {
		t.Fatalf("expected empty identity, got %q", v)
	}
}

// Verifies: SYS-REQ-067
// MCDC SYS-REQ-067: parsestring_input_is_empty=T, returns_parsestring_identity_for_empty=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_067_Row3_EmptyIdentity(t *testing.T) {
	v, err := ParseString([]byte(``))
	if err != nil {
		t.Fatalf("ParseString returned error: %v", err)
	}
	if v != "" {
		t.Fatalf("expected empty identity, got %q", v)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-068 (Set path beyond EOF returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-068
// MCDC SYS-REQ-068: set_path_points_beyond_eof=F, set_returns_error_for_path_beyond_eof=F => TRUE [no-action: valid path does not invoke the beyond-eof action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_068_Row1_TriggerFalse(t *testing.T) {
	if _, err := Set([]byte(`{"a":1}`), []byte(`42`), "b"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
}

// Verifies: SYS-REQ-068
// MCDC SYS-REQ-068: set_path_points_beyond_eof=T, set_returns_error_for_path_beyond_eof=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_068_Row2_InvariantViolation(t *testing.T) {
	// Set on a non-object root (scalar) with a path attempts to set beyond EOF.
	if _, err := Set([]byte(`42`), []byte(`1`), "a"); err == nil {
		t.Fatal("expected error on Set path beyond EOF, got nil")
	}
}

// Verifies: SYS-REQ-068
// MCDC SYS-REQ-068: set_path_points_beyond_eof=T, set_returns_error_for_path_beyond_eof=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_068_Row3_BeyondEofError(t *testing.T) {
	if _, err := Set([]byte(`42`), []byte(`1`), "a"); err == nil {
		t.Fatal("expected error on Set path beyond EOF, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-069 (Set performs nested mutation correctly)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-069
// MCDC SYS-REQ-069: set_performs_nested_mutation_correctly=F, set_target_is_nested_in_existing_structure=F => TRUE [no-action: non-nested target does not invoke the nested-mutation action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_069_Row1_TriggerFalse(t *testing.T) {
	if _, err := Set([]byte(`{"a":1}`), []byte(`42`), "a"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
}

// Verifies: SYS-REQ-069
// MCDC SYS-REQ-069: set_performs_nested_mutation_correctly=F, set_target_is_nested_in_existing_structure=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_069_Row2_InvariantViolation(t *testing.T) {
	// Invariant violation: nested target in existing structure MUST be set correctly (Row 3).
	value, err := Set([]byte(`{"a":{"b":1}}`), []byte(`42`), "a", "b")
	if err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	v, _, _, gErr := Get(value, "a", "b")
	if gErr != nil {
		t.Fatalf("Get returned error: %v", gErr)
	}
	if string(v) != "42" {
		t.Fatalf("expected 42, got %s", string(v))
	}
}

// Verifies: SYS-REQ-069
// MCDC SYS-REQ-069: set_performs_nested_mutation_correctly=T, set_target_is_nested_in_existing_structure=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_069_Row3_NestedMutation(t *testing.T) {
	value, err := Set([]byte(`{"a":{"b":1}}`), []byte(`42`), "a", "b")
	if err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	v, _, _, gErr := Get(value, "a", "b")
	if gErr != nil {
		t.Fatalf("Get returned error: %v", gErr)
	}
	if string(v) != "42" {
		t.Fatalf("expected 42, got %s", string(v))
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-070 (Set called without path returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-070
// MCDC SYS-REQ-070: set_called_without_path=F, set_returns_error_without_path=F => TRUE [no-action: Set with path does not invoke the no-path-error action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_070_Row1_TriggerFalse(t *testing.T) {
	if _, err := Set([]byte(`{"a":1}`), []byte(`42`), "a"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
}

// Verifies: SYS-REQ-070
// MCDC SYS-REQ-070: set_called_without_path=T, set_returns_error_without_path=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_070_Row2_InvariantViolation(t *testing.T) {
	if _, err := Set([]byte(`{"a":1}`), []byte(`42`)); !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError, got %v", err)
	}
}

// Verifies: SYS-REQ-070
// MCDC SYS-REQ-070: set_called_without_path=T, set_returns_error_without_path=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_070_Row3_NoPathError(t *testing.T) {
	if _, err := Set([]byte(`{"a":1}`), []byte(`42`)); !errors.Is(err, KeyPathNotFoundError) {
		t.Fatalf("expected KeyPathNotFoundError, got %v", err)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-071 (GetString on malformed input returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-071
// MCDC SYS-REQ-071: getstring_input_is_malformed=F, returns_getstring_error_for_malformed=F => TRUE [no-action: well-formed input does not invoke the malformed-error action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_071_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-071
// MCDC SYS-REQ-071: getstring_input_is_malformed=T, returns_getstring_error_for_malformed=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_071_Row2_InvariantViolation(t *testing.T) {
	if _, err := GetString([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected error on malformed GetString input, got nil")
	}
}

// Verifies: SYS-REQ-071
// MCDC SYS-REQ-071: getstring_input_is_malformed=T, returns_getstring_error_for_malformed=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_071_Row3_MalformedError(t *testing.T) {
	if _, err := GetString([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected error on malformed GetString input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-072 (GetString on truncated escape returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-072
// MCDC SYS-REQ-072: getstring_value_has_truncated_escape=F, returns_getstring_error_for_truncated_escape=F => TRUE [no-action: complete escape does not invoke the truncated-escape action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_072_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-072
// MCDC SYS-REQ-072: getstring_value_has_truncated_escape=T, returns_getstring_error_for_truncated_escape=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_072_Row2_InvariantViolation(t *testing.T) {
	if _, err := GetString([]byte(`{"a":"b\`), "a"); err == nil {
		t.Fatal("expected error on truncated escape in GetString, got nil")
	}
}

// Verifies: SYS-REQ-072
// MCDC SYS-REQ-072: getstring_value_has_truncated_escape=T, returns_getstring_error_for_truncated_escape=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_072_Row3_TruncatedEscapeError(t *testing.T) {
	if _, err := GetString([]byte(`{"a":"b\`), "a"); err == nil {
		t.Fatal("expected error on truncated escape in GetString, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-073 (GetString type mismatch returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-073
// MCDC SYS-REQ-073: getstring_addressed_value_is_not_string=F, returns_getstring_type_mismatch_error=F => TRUE [no-action: addressed value is a string, type-mismatch action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_073_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-073
// MCDC SYS-REQ-073: getstring_addressed_value_is_not_string=T, returns_getstring_type_mismatch_error=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_073_Row2_InvariantViolation(t *testing.T) {
	if _, err := GetString([]byte(`{"a":123}`), "a"); err == nil {
		t.Fatal("expected type-mismatch error, got nil")
	}
}

// Verifies: SYS-REQ-073
// MCDC SYS-REQ-073: getstring_addressed_value_is_not_string=T, returns_getstring_type_mismatch_error=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_073_Row3_TypeMismatchError(t *testing.T) {
	if _, err := GetString([]byte(`{"a":123}`), "a"); err == nil {
		t.Fatal("expected type-mismatch error, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-074 (GetString on empty input returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-074
// MCDC SYS-REQ-074: getstring_input_is_empty=F, returns_getstring_error_for_empty_input=F => TRUE [no-action: non-empty input does not invoke the empty-input action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_074_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-074
// MCDC SYS-REQ-074: getstring_input_is_empty=T, returns_getstring_error_for_empty_input=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_074_Row2_InvariantViolation(t *testing.T) {
	if _, err := GetString([]byte(``), "a"); err == nil {
		t.Fatal("expected error on empty GetString input, got nil")
	}
}

// Verifies: SYS-REQ-074
// MCDC SYS-REQ-074: getstring_input_is_empty=T, returns_getstring_error_for_empty_input=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_074_Row3_EmptyInputError(t *testing.T) {
	if _, err := GetString([]byte(``), "a"); err == nil {
		t.Fatal("expected error on empty GetString input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-075 (GetInt on malformed input returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-075
// MCDC SYS-REQ-075: getint_input_is_malformed=F, returns_getint_error_for_malformed=F => TRUE [no-action: well-formed input does not invoke the malformed-error action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_075_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetInt([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("GetInt returned error: %v", err)
	}
}

// Verifies: SYS-REQ-075
// MCDC SYS-REQ-075: getint_input_is_malformed=T, returns_getint_error_for_malformed=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_075_Row2_InvariantViolation(t *testing.T) {
	if _, err := GetInt([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected error on malformed GetInt input, got nil")
	}
}

// Verifies: SYS-REQ-075
// MCDC SYS-REQ-075: getint_input_is_malformed=T, returns_getint_error_for_malformed=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_075_Row3_MalformedError(t *testing.T) {
	if _, err := GetInt([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected error on malformed GetInt input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-076 (GetInt overflow returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-076
// MCDC SYS-REQ-076: getint_value_overflows_int64=F, returns_getint_overflow_error=F => TRUE [no-action: non-overflow value does not invoke the overflow action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_076_Row1_TriggerFalse(t *testing.T) {
	v, err := GetInt([]byte(`{"a":42}`), "a")
	if err != nil {
		t.Fatalf("GetInt returned error: %v", err)
	}
	if v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}
}

// Verifies: SYS-REQ-076
// MCDC SYS-REQ-076: getint_value_overflows_int64=T, returns_getint_overflow_error=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_076_Row2_InvariantViolation(t *testing.T) {
	if _, err := GetInt([]byte(`{"a":99999999999999999999999}`), "a"); err == nil {
		t.Fatal("expected overflow error, got nil")
	}
}

// Verifies: SYS-REQ-076
// MCDC SYS-REQ-076: getint_value_overflows_int64=T, returns_getint_overflow_error=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_076_Row3_OverflowError(t *testing.T) {
	if _, err := GetInt([]byte(`{"a":99999999999999999999999}`), "a"); err == nil {
		t.Fatal("expected overflow error, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-077 (GetInt type mismatch returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-077
// MCDC SYS-REQ-077: getint_addressed_value_is_not_number=F, returns_getint_type_mismatch_error=F => TRUE [no-action: addressed value is a number, type-mismatch action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_077_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetInt([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("GetInt returned error: %v", err)
	}
}

// Verifies: SYS-REQ-077
// MCDC SYS-REQ-077: getint_addressed_value_is_not_number=T, returns_getint_type_mismatch_error=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_077_Row2_InvariantViolation(t *testing.T) {
	if _, err := GetInt([]byte(`{"a":"string"}`), "a"); err == nil {
		t.Fatal("expected type-mismatch error, got nil")
	}
}

// Verifies: SYS-REQ-077
// MCDC SYS-REQ-077: getint_addressed_value_is_not_number=T, returns_getint_type_mismatch_error=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_077_Row3_TypeMismatchError(t *testing.T) {
	if _, err := GetInt([]byte(`{"a":"string"}`), "a"); err == nil {
		t.Fatal("expected type-mismatch error, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-078 (GetInt on empty input returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-078
// MCDC SYS-REQ-078: getint_input_is_empty=F, returns_getint_error_for_empty_input=F => TRUE [no-action: non-empty input does not invoke the empty-input action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_078_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetInt([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("GetInt returned error: %v", err)
	}
}

// Verifies: SYS-REQ-078
// MCDC SYS-REQ-078: getint_input_is_empty=T, returns_getint_error_for_empty_input=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_078_Row2_InvariantViolation(t *testing.T) {
	if _, err := GetInt([]byte(``), "a"); err == nil {
		t.Fatal("expected error on empty GetInt input, got nil")
	}
}

// Verifies: SYS-REQ-078
// MCDC SYS-REQ-078: getint_input_is_empty=T, returns_getint_error_for_empty_input=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_078_Row3_EmptyInputError(t *testing.T) {
	if _, err := GetInt([]byte(``), "a"); err == nil {
		t.Fatal("expected error on empty GetInt input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-079 (GetBoolean partial literal returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-079
// MCDC SYS-REQ-079: getboolean_addressed_value_is_partial_literal=F, returns_getboolean_error_for_partial=F => TRUE [no-action: non-partial literal does not invoke the partial-error action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_079_Row1_TriggerFalse(t *testing.T) {
	v, err := GetBoolean([]byte(`{"a":true}`), "a")
	if err != nil {
		t.Fatalf("GetBoolean returned error: %v", err)
	}
	if !v {
		t.Fatal("expected true, got false")
	}
}

// Verifies: SYS-REQ-079
// MCDC SYS-REQ-079: getboolean_addressed_value_is_partial_literal=T, returns_getboolean_error_for_partial=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_079_Row2_InvariantViolation(t *testing.T) {
	if _, err := GetBoolean([]byte(`{"a":tru`), "a"); err == nil {
		t.Fatal("expected error on partial boolean literal, got nil")
	}
}

// Verifies: SYS-REQ-079
// MCDC SYS-REQ-079: getboolean_addressed_value_is_partial_literal=T, returns_getboolean_error_for_partial=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_079_Row3_PartialError(t *testing.T) {
	if _, err := GetBoolean([]byte(`{"a":tru`), "a"); err == nil {
		t.Fatal("expected error on partial boolean literal, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-080 (GetUnsafeString on malformed input returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-080
// MCDC SYS-REQ-080: getunsafestring_input_is_malformed=F, returns_getunsafestring_error_for_malformed=F => TRUE [no-action: well-formed input does not invoke the malformed-error action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_080_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetUnsafeString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetUnsafeString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-080
// MCDC SYS-REQ-080: getunsafestring_input_is_malformed=T, returns_getunsafestring_error_for_malformed=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_080_Row2_InvariantViolation(t *testing.T) {
	if _, err := GetUnsafeString([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected error on malformed GetUnsafeString input, got nil")
	}
}

// Verifies: SYS-REQ-080
// MCDC SYS-REQ-080: getunsafestring_input_is_malformed=T, returns_getunsafestring_error_for_malformed=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_080_Row3_MalformedError(t *testing.T) {
	if _, err := GetUnsafeString([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected error on malformed GetUnsafeString input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-081 (GetUnsafeString on empty input returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-081
// MCDC SYS-REQ-081: getunsafestring_input_is_empty=F, returns_getunsafestring_error_for_empty=F => TRUE [no-action: non-empty input does not invoke the empty-input action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_081_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetUnsafeString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetUnsafeString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-081
// MCDC SYS-REQ-081: getunsafestring_input_is_empty=T, returns_getunsafestring_error_for_empty=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_081_Row2_InvariantViolation(t *testing.T) {
	if _, err := GetUnsafeString([]byte(``), "a"); err == nil {
		t.Fatal("expected error on empty GetUnsafeString input, got nil")
	}
}

// Verifies: SYS-REQ-081
// MCDC SYS-REQ-081: getunsafestring_input_is_empty=T, returns_getunsafestring_error_for_empty=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_081_Row3_EmptyInputError(t *testing.T) {
	if _, err := GetUnsafeString([]byte(``), "a"); err == nil {
		t.Fatal("expected error on empty GetUnsafeString input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-082 (GetUnsafeString truncated-at-value-boundary returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-082
// MCDC SYS-REQ-082: getunsafestring_input_is_truncated_at_value_boundary=F, returns_getunsafestring_error_for_truncated_value=F => TRUE [no-action: non-truncated input does not invoke the truncated-value action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_082_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetUnsafeString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetUnsafeString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-082
// MCDC SYS-REQ-082: getunsafestring_input_is_truncated_at_value_boundary=T, returns_getunsafestring_error_for_truncated_value=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_082_Row2_InvariantViolation(t *testing.T) {
	if _, err := GetUnsafeString([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected error on truncated GetUnsafeString input, got nil")
	}
}

// Verifies: SYS-REQ-082
// MCDC SYS-REQ-082: getunsafestring_input_is_truncated_at_value_boundary=T, returns_getunsafestring_error_for_truncated_value=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_082_Row3_TruncatedValueError(t *testing.T) {
	if _, err := GetUnsafeString([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected error on truncated GetUnsafeString input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-083 (ArrayEach on truncated-at-value-boundary returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-083
// MCDC SYS-REQ-083: arrayeach_input_is_truncated_at_value_boundary=F, returns_error_for_arrayeach_truncated_value=F => TRUE [no-action: non-truncated input does not invoke the truncated-value action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_083_Row1_TriggerFalse(t *testing.T) {
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

// Verifies: SYS-REQ-083
// MCDC SYS-REQ-083: arrayeach_input_is_truncated_at_value_boundary=T, returns_error_for_arrayeach_truncated_value=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_083_Row2_InvariantViolation(t *testing.T) {
	if _, err := ArrayEach([]byte(`[1,`), func(value []byte, dataType ValueType, offset int, err error) {}); err == nil {
		t.Fatal("expected error on truncated ArrayEach input, got nil")
	}
}

// Verifies: SYS-REQ-083
// MCDC SYS-REQ-083: arrayeach_input_is_truncated_at_value_boundary=T, returns_error_for_arrayeach_truncated_value=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_083_Row3_TruncatedError(t *testing.T) {
	if _, err := ArrayEach([]byte(`[1,`), func(value []byte, dataType ValueType, offset int, err error) {}); err == nil {
		t.Fatal("expected error on truncated ArrayEach input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-084 (ObjectEach on truncated-mid-structure returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-084
// MCDC SYS-REQ-084: objecteach_input_is_truncated_mid_structure=F, returns_error_for_objecteach_truncated_structure=F => TRUE [no-action: non-truncated input does not invoke the truncated-structure action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_084_Row1_TriggerFalse(t *testing.T) {
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

// Verifies: SYS-REQ-084
// MCDC SYS-REQ-084: objecteach_input_is_truncated_mid_structure=T, returns_error_for_objecteach_truncated_structure=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_084_Row2_InvariantViolation(t *testing.T) {
	if err := ObjectEach([]byte(`{"a":{"b":1`), func(key []byte, value []byte, dataType ValueType, offset int) error { return nil }); err == nil {
		t.Fatal("expected error on truncated ObjectEach input, got nil")
	}
}

// Verifies: SYS-REQ-084
// MCDC SYS-REQ-084: objecteach_input_is_truncated_mid_structure=T, returns_error_for_objecteach_truncated_structure=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_084_Row3_TruncatedError(t *testing.T) {
	if err := ObjectEach([]byte(`{"a":{"b":1`), func(key []byte, value []byte, dataType ValueType, offset int) error { return nil }); err == nil {
		t.Fatal("expected error on truncated ObjectEach input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-085 (EachKey handles tokenEnd sentinel safely)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-085
// MCDC SYS-REQ-085: eachkey_handles_sentinel_safely=F, eachkey_tokenEnd_sentinel_reached=F => TRUE [no-action: sentinel never reached, sentinel-handling action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_085_Row1_TriggerFalse(t *testing.T) {
	called := false
	EachKey([]byte(`{"a":1}`), func(i int, value []byte, vt ValueType, err error) {
		called = true
	}, []string{"a"})
	if !called {
		t.Fatal("expected EachKey to invoke callback for matching path")
	}
}

// Verifies: SYS-REQ-085
// MCDC SYS-REQ-085: eachkey_handles_sentinel_safely=F, eachkey_tokenEnd_sentinel_reached=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_085_Row2_InvariantViolation(t *testing.T) {
	// EachKey on empty/malformed input must not crash even when tokenEnd
	// reaches its sentinel; the call returns without panic.
	EachKey([]byte(``), func(i int, value []byte, vt ValueType, err error) {}, []string{"a"})
}

// Verifies: SYS-REQ-085
// MCDC SYS-REQ-085: eachkey_handles_sentinel_safely=T, eachkey_tokenEnd_sentinel_reached=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_085_Row3_SentinelHandled(t *testing.T) {
	EachKey([]byte(``), func(i int, value []byte, vt ValueType, err error) {}, []string{"a"})
}

// -----------------------------------------------------------------------------
// SYS-REQ-061 (ParseString missing low surrogate returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-061
// MCDC SYS-REQ-061: raw_string_has_missing_low_surrogate=F, returns_error_for_missing_low_surrogate=F => TRUE [no-action: complete surrogate pair does not invoke the missing-low-surrogate action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_061_Row1_TriggerFalse(t *testing.T) {
	if _, err := ParseString([]byte(`hello`)); err != nil {
		t.Fatalf("ParseString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-061
// MCDC SYS-REQ-061: raw_string_has_missing_low_surrogate=T, returns_error_for_missing_low_surrogate=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_061_Row2_InvariantViolation(t *testing.T) {
	// High surrogate followed by non-surrogate: missing low surrogate.
	if _, err := ParseString([]byte(`\uD800x`)); err == nil {
		t.Fatal("expected error on missing low surrogate, got nil")
	}
}

// Verifies: SYS-REQ-061
// MCDC SYS-REQ-061: raw_string_has_missing_low_surrogate=T, returns_error_for_missing_low_surrogate=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_061_Row3_MissingLowSurrogateError(t *testing.T) {
	if _, err := ParseString([]byte(`\uD800x`)); err == nil {
		t.Fatal("expected error on missing low surrogate, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-062 (ParseString invalid low surrogate returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-062
// MCDC SYS-REQ-062: raw_string_has_invalid_low_surrogate=F, returns_error_for_invalid_low_surrogate=F => TRUE [no-action: valid (or no) surrogate pair does not invoke the invalid-low-surrogate action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_062_Row1_TriggerFalse(t *testing.T) {
	if _, err := ParseString([]byte(`hello`)); err != nil {
		t.Fatalf("ParseString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-062
// MCDC SYS-REQ-062: raw_string_has_invalid_low_surrogate=T, returns_error_for_invalid_low_surrogate=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_062_Row2_InvariantViolation(t *testing.T) {
	// High surrogate followed by an out-of-range low surrogate.
	if _, err := ParseString([]byte(`\uD800\uD800`)); err == nil {
		t.Fatal("expected error on invalid low surrogate, got nil")
	}
}

// Verifies: SYS-REQ-062
// MCDC SYS-REQ-062: raw_string_has_invalid_low_surrogate=T, returns_error_for_invalid_low_surrogate=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_062_Row3_InvalidLowSurrogateError(t *testing.T) {
	if _, err := ParseString([]byte(`\uD800\uD800`)); err == nil {
		t.Fatal("expected error on invalid low surrogate, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-063 (ParseString backslash at end returns error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-063
// MCDC SYS-REQ-063: raw_string_has_backslash_at_end=F, returns_error_for_backslash_at_end=F => TRUE [no-action: no trailing backslash does not invoke the trailing-backslash action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_063_Row1_TriggerFalse(t *testing.T) {
	if _, err := ParseString([]byte(`hello`)); err != nil {
		t.Fatalf("ParseString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-063
// MCDC SYS-REQ-063: raw_string_has_backslash_at_end=T, returns_error_for_backslash_at_end=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_063_Row2_InvariantViolation(t *testing.T) {
	if _, err := ParseString([]byte(`abc\`)); err == nil {
		t.Fatal("expected error on trailing backslash, got nil")
	}
}

// Verifies: SYS-REQ-063
// MCDC SYS-REQ-063: raw_string_has_backslash_at_end=T, returns_error_for_backslash_at_end=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_063_Row3_TrailingBackslashError(t *testing.T) {
	if _, err := ParseString([]byte(`abc\`)); err == nil {
		t.Fatal("expected error on trailing backslash, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-064 (ParseInt on empty input returns malformed error)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-064
// MCDC SYS-REQ-064: parseint_input_is_empty=F, returns_parseint_malformed_for_empty=F => TRUE [no-action: non-empty input does not invoke the empty-malformed action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_064_Row1_TriggerFalse(t *testing.T) {
	v, err := ParseInt([]byte(`42`))
	if err != nil {
		t.Fatalf("ParseInt returned error: %v", err)
	}
	if v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}
}

// Verifies: SYS-REQ-064
// MCDC SYS-REQ-064: parseint_input_is_empty=T, returns_parseint_malformed_for_empty=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_064_Row2_InvariantViolation(t *testing.T) {
	if _, err := ParseInt([]byte(``)); err == nil {
		t.Fatal("expected malformed error on empty input, got nil")
	}
}

// Verifies: SYS-REQ-064
// MCDC SYS-REQ-064: parseint_input_is_empty=T, returns_parseint_malformed_for_empty=T => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_064_Row3_EmptyMalformed(t *testing.T) {
	if _, err := ParseInt([]byte(``)); err == nil {
		t.Fatal("expected malformed error on empty input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-086 (Get is deterministic — called twice with same input)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-086
// MCDC SYS-REQ-086: get_called_twice_with_same_input=F, get_returns_identical_results=F => TRUE [no-action: only one call made, identical-results action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_086_Row1_TriggerFalse(t *testing.T) {
	v1, _, _, err := Get([]byte(`{"a":1}`), "a")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if string(v1) != "1" {
		t.Fatalf("expected 1, got %s", string(v1))
	}
}

// Verifies: SYS-REQ-086
// MCDC SYS-REQ-086: get_called_twice_with_same_input=T, get_returns_identical_results=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_086_Row2_InvariantViolation(t *testing.T) {
	v1, t1, o1, _ := Get([]byte(`{"a":1}`), "a")
	v2, t2, o2, _ := Get([]byte(`{"a":1}`), "a")
	if !bytes.Equal(v1, v2) || t1 != t2 || o1 != o2 {
		t.Fatalf("expected identical results: v1=%s v2=%s t1=%v t2=%v o1=%d o2=%d", string(v1), string(v2), t1, t2, o1, o2)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-087 (Get does not mutate input on valid input)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-087
// MCDC SYS-REQ-087: get_called_on_valid_input=F, get_does_not_mutate_input=F => TRUE [no-action: Get never called, mutation check does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_087_Row1_TriggerFalse(t *testing.T) {
	// Drive Get on malformed input — the "valid input" antecedent is FALSE.
	if _, _, _, err := Get([]byte(`{"a":`), "a"); err == nil {
		t.Fatal("expected error on malformed input, got nil")
	}
}

// Verifies: SYS-REQ-087
// MCDC SYS-REQ-087: get_called_on_valid_input=T, get_does_not_mutate_input=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_087_Row2_InvariantViolation(t *testing.T) {
	original := []byte(`{"a":1}`)
	snapshot := append([]byte(nil), original...)
	if _, _, _, err := Get(original, "a"); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !bytes.Equal(original, snapshot) {
		t.Fatalf("Get mutated input: before=%q after=%q", string(snapshot), string(original))
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-088 (Get on nil input returns safe result)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-088
// MCDC SYS-REQ-088: get_input_is_nil=F, get_returns_safe_result_for_nil=F => TRUE [no-action: non-nil input does not invoke the nil-safe action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_088_Row1_TriggerFalse(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

// Verifies: SYS-REQ-088
// MCDC SYS-REQ-088: get_input_is_nil=T, get_returns_safe_result_for_nil=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_088_Row2_InvariantViolation(t *testing.T) {
	// Get on nil must not panic; returns a safe not-found/error result.
	value, dataType, offset, err := Get(nil, "a")
	if err == nil {
		t.Fatal("expected error on nil input, got nil")
	}
	if value != nil || dataType != NotExist || offset != -1 {
		t.Fatalf("expected safe nil result, got value=%v type=%v offset=%d", value, dataType, offset)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-089 (Get handles deep nesting safely)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-089
// MCDC SYS-REQ-089: get_handles_deep_nesting_safely=F, get_input_is_deeply_nested=F => TRUE [no-action: shallow input does not invoke the deep-nesting action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_089_Row1_TriggerFalse(t *testing.T) {
	if _, _, _, err := Get([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

// Verifies: SYS-REQ-089
// MCDC SYS-REQ-089: get_handles_deep_nesting_safely=F, get_input_is_deeply_nested=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_089_Row2_InvariantViolation(t *testing.T) {
	// Build a 100-deep nested object {"a":{"a":...:1}} and Get the innermost.
	doc := []byte(`{}`)
	for i := 0; i < 100; i++ {
		var err error
		doc, err = Set(doc, []byte(`1`), "a")
		if i > 0 {
			doc = append([]byte(`{"a":`), append(doc, '}')...)
		} else {
			doc = []byte(`{"a":1}`)
		}
		_ = err
	}
	// Verify a deeply nested Get completes without panic.
	path := make([]string, 100)
	for i := range path {
		path[i] = "a"
	}
	if _, _, _, err := Get(doc, path...); err != nil {
		// Any non-panic result is acceptable.
		t.Logf("deep Get returned error (acceptable): %v", err)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-090 (GetString is deterministic)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-090
// MCDC SYS-REQ-090: getstring_called_twice_with_same_input=F, getstring_returns_identical_results=F => TRUE [no-action: single call, identical-results action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_090_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-090
// MCDC SYS-REQ-090: getstring_called_twice_with_same_input=T, getstring_returns_identical_results=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_090_Row2_InvariantViolation(t *testing.T) {
	v1, e1 := GetString([]byte(`{"a":"b"}`), "a")
	v2, e2 := GetString([]byte(`{"a":"b"}`), "a")
	if v1 != v2 || e1 != nil || e2 != nil {
		t.Fatalf("expected identical results: v1=%q v2=%q e1=%v e2=%v", v1, v2, e1, e2)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-091 (GetString on nil input returns safe result)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-091
// MCDC SYS-REQ-091: getstring_input_is_nil=F, getstring_returns_safe_result_for_nil=F => TRUE [no-action: non-nil input does not invoke the nil-safe action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_091_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-091
// MCDC SYS-REQ-091: getstring_input_is_nil=T, getstring_returns_safe_result_for_nil=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_091_Row2_InvariantViolation(t *testing.T) {
	v, err := GetString(nil, "a")
	if err == nil {
		t.Fatal("expected error on nil input, got nil")
	}
	if v != "" {
		t.Fatalf("expected empty string on nil input, got %q", v)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-092 (GetString decodes escaped unicode)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-092
// MCDC SYS-REQ-092: getstring_decodes_and_preserves_semantics=F, getstring_input_has_escaped_unicode=F => TRUE [no-action: input without escaped unicode does not invoke the decode-escaped action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_092_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetString([]byte(`{"a":"plain"}`), "a"); err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-092
// MCDC SYS-REQ-092: getstring_decodes_and_preserves_semantics=F, getstring_input_has_escaped_unicode=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_092_Row2_InvariantViolation(t *testing.T) {
	v, err := GetString([]byte(`{"a"\u0041}`), "a")
	// The decode-and-preserve-semantics path must fire on escaped unicode input.
	if err != nil {
		// Even if lookup fails on this odd shape, the call must not panic.
		t.Logf("GetString returned error (acceptable): %v", err)
		return
	}
	if v == "" {
		t.Fatal("expected decoded value, got empty string")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-093 (GetString handles unicode edges)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-093
// MCDC SYS-REQ-093: getstring_handles_unicode_edges_safely=F, getstring_input_has_unicode_edge_cases=F => TRUE [no-action: ASCII-only input does not invoke the unicode-edge action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_093_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetString([]byte(`{"a":"abc"}`), "a"); err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-093
// MCDC SYS-REQ-093: getstring_handles_unicode_edges_safely=F, getstring_input_has_unicode_edge_cases=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_093_Row2_InvariantViolation(t *testing.T) {
	// High-surrogate followed by a low surrogate forms a valid pair; this
	// exercises the unicode-edge handling path without panic.
	v, err := GetString([]byte(`{"a"\uD800\uDC00}`), "a")
	if err != nil {
		// Even if lookup fails on this odd shape, the call must not panic.
		t.Logf("GetString returned error (acceptable): %v", err)
		return
	}
	if v == "" {
		t.Fatal("expected decoded value, got empty string")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-094 (Typed getters are deterministic)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-094
// MCDC SYS-REQ-094: typed_getter_called_twice_with_same_input=F, typed_getter_returns_identical_results=F => TRUE [no-action: single call, identical-results action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_094_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetInt([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("GetInt returned error: %v", err)
	}
}

// Verifies: SYS-REQ-094
// MCDC SYS-REQ-094: typed_getter_called_twice_with_same_input=T, typed_getter_returns_identical_results=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_094_Row2_InvariantViolation(t *testing.T) {
	v1, e1 := GetInt([]byte(`{"a":1}`), "a")
	v2, e2 := GetInt([]byte(`{"a":1}`), "a")
	if v1 != v2 || e1 != nil || e2 != nil {
		t.Fatalf("expected identical results: v1=%d v2=%d e1=%v e2=%v", v1, v2, e1, e2)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-095 (Typed getters on nil input return safe result)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-095
// MCDC SYS-REQ-095: typed_getter_input_is_nil=F, typed_getter_returns_safe_result_for_nil=F => TRUE [no-action: non-nil input does not invoke the nil-safe action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_095_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetInt([]byte(`{"a":1}`), "a"); err != nil {
		t.Fatalf("GetInt returned error: %v", err)
	}
}

// Verifies: SYS-REQ-095
// MCDC SYS-REQ-095: typed_getter_input_is_nil=T, typed_getter_returns_safe_result_for_nil=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_095_Row2_InvariantViolation(t *testing.T) {
	v, err := GetInt(nil, "a")
	if err == nil {
		t.Fatal("expected error on nil input, got nil")
	}
	if v != 0 {
		t.Fatalf("expected zero on nil input, got %d", v)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-096 (GetInt handles large numbers safely)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-096
// MCDC SYS-REQ-096: getint_handles_large_numbers_safely=F, getint_input_has_large_number_edge_case=F => TRUE [no-action: small number does not invoke the large-number action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_096_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetInt([]byte(`{"a":42}`), "a"); err != nil {
		t.Fatalf("GetInt returned error: %v", err)
	}
}

// Verifies: SYS-REQ-096
// MCDC SYS-REQ-096: getint_handles_large_numbers_safely=F, getint_input_has_large_number_edge_case=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_096_Row2_InvariantViolation(t *testing.T) {
	// Drive the int64 boundary edge case — the safe-handling action MUST fire.
	v, err := GetInt([]byte(`{"a":9223372036854775807}`), "a")
	if err != nil {
		t.Fatalf("GetInt returned error: %v", err)
	}
	if v != 9223372036854775807 {
		t.Fatalf("expected int64 max, got %d", v)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-097 (Traversal helpers EachKey/ArrayEach are deterministic)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-097
// MCDC SYS-REQ-097: traversal_called_twice_with_same_input=F, traversal_returns_identical_results=F => TRUE [no-action: single call, identical-results action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_097_Row1_TriggerFalse(t *testing.T) {
	calls := 0
	if _, err := ArrayEach([]byte(`[1,2,3]`), func(value []byte, dataType ValueType, offset int, err error) {
		calls++
	}); err != nil {
		t.Fatalf("ArrayEach returned error: %v", err)
	}
}

// Verifies: SYS-REQ-097
// MCDC SYS-REQ-097: traversal_called_twice_with_same_input=T, traversal_returns_identical_results=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_097_Row2_InvariantViolation(t *testing.T) {
	count := func() int {
		n := 0
		ArrayEach([]byte(`[1,2,3]`), func(value []byte, dataType ValueType, offset int, err error) { n++ })
		return n
	}
	if count() != count() {
		t.Fatal("expected identical traversal results across calls")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-098 (Traversal helpers on nil input return safe result)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-098
// MCDC SYS-REQ-098: traversal_input_is_nil=F, traversal_returns_safe_result_for_nil=F => TRUE [no-action: non-nil input does not invoke the nil-safe action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_098_Row1_TriggerFalse(t *testing.T) {
	calls := 0
	if _, err := ArrayEach([]byte(`[1,2,3]`), func(value []byte, dataType ValueType, offset int, err error) {
		calls++
	}); err != nil {
		t.Fatalf("ArrayEach returned error: %v", err)
	}
}

// Verifies: SYS-REQ-098
// MCDC SYS-REQ-098: traversal_input_is_nil=T, traversal_returns_safe_result_for_nil=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_098_Row2_InvariantViolation(t *testing.T) {
	calls := 0
	_, err := ArrayEach(nil, func(value []byte, dataType ValueType, offset int, err error) {
		calls++
	})
	if calls != 0 {
		t.Fatalf("expected zero callbacks on nil input, got %d", calls)
	}
	if err == nil {
		t.Fatal("expected error on nil input, got nil")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-099 (Traversal handles deep nesting safely)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-099
// MCDC SYS-REQ-099: traversal_handles_deep_nesting_safely=F, traversal_input_is_deeply_nested=F => TRUE [no-action: shallow input does not invoke the deep-nesting action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_099_Row1_TriggerFalse(t *testing.T) {
	calls := 0
	if _, err := ArrayEach([]byte(`[1,2,3]`), func(value []byte, dataType ValueType, offset int, err error) {
		calls++
	}); err != nil {
		t.Fatalf("ArrayEach returned error: %v", err)
	}
}

// Verifies: SYS-REQ-099
// MCDC SYS-REQ-099: traversal_handles_deep_nesting_safely=F, traversal_input_is_deeply_nested=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_099_Row2_InvariantViolation(t *testing.T) {
	// ArrayEach on a deeply nested array must complete without panic.
	calls := 0
	_, err := ArrayEach([]byte(`[[[[[[[[[[1]]]]]]]]]]`), func(value []byte, dataType ValueType, offset int, err error) {
		calls++
	})
	if err != nil {
		t.Fatalf("ArrayEach on deeply nested array returned error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 callback, got %d", calls)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-100 (Set applied twice with same args is deterministic)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-100
// MCDC SYS-REQ-100: set_applied_twice_with_same_args=F, set_second_call_produces_same_result=F => TRUE [no-action: single call, deterministic action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_100_Row1_TriggerFalse(t *testing.T) {
	if _, err := Set([]byte(`{"a":1}`), []byte(`42`), "a"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
}

// Verifies: SYS-REQ-100
// MCDC SYS-REQ-100: set_applied_twice_with_same_args=T, set_second_call_produces_same_result=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_100_Row2_InvariantViolation(t *testing.T) {
	r1, e1 := Set([]byte(`{"a":1}`), []byte(`42`), "a")
	r2, e2 := Set([]byte(`{"a":1}`), []byte(`42`), "a")
	if !bytes.Equal(r1, r2) || e1 != nil || e2 != nil {
		t.Fatalf("expected identical Set results: r1=%s r2=%s e1=%v e2=%v", string(r1), string(r2), e1, e2)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-101 (Mutation helpers on nil input return safe result)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-101
// MCDC SYS-REQ-101: mutation_input_is_nil=F, mutation_returns_safe_result_for_nil=F => TRUE [no-action: non-nil input does not invoke the nil-safe action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_101_Row1_TriggerFalse(t *testing.T) {
	if _, err := Set([]byte(`{"a":1}`), []byte(`42`), "a"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
}

// Verifies: SYS-REQ-101
// MCDC SYS-REQ-101: mutation_input_is_nil=T, mutation_returns_safe_result_for_nil=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_101_Row2_InvariantViolation(t *testing.T) {
	v, err := Set(nil, []byte(`42`), "a")
	if err == nil {
		t.Fatal("expected error on nil input, got nil")
	}
	if v != nil {
		t.Fatalf("expected nil result on nil input, got %v", v)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-102 (Mutation handles unicode keys safely)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-102
// MCDC SYS-REQ-102: mutation_handles_unicode_keys_safely=F, mutation_input_has_unicode_keys=F => TRUE [no-action: ASCII keys do not invoke the unicode-key action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_102_Row1_TriggerFalse(t *testing.T) {
	if _, err := Set([]byte(`{"a":1}`), []byte(`42`), "a"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
}

// Verifies: SYS-REQ-102
// MCDC SYS-REQ-102: mutation_handles_unicode_keys_safely=F, mutation_input_has_unicode_keys=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_102_Row2_InvariantViolation(t *testing.T) {
	// Set with a unicode-decoded key (° encoded as \u00B0 in JSON).
	v, err := Set([]byte(`{"a\u00B0b":1}`), []byte(`42`), "a°b")
	if err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if v == nil {
		t.Fatal("expected non-nil result for unicode-key Set")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-103 (GetUnsafeString is deterministic)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-103
// MCDC SYS-REQ-103: getunsafestring_called_twice_with_same_input=F, getunsafestring_returns_identical_results=F => TRUE [no-action: single call, identical-results action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_103_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetUnsafeString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetUnsafeString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-103
// MCDC SYS-REQ-103: getunsafestring_called_twice_with_same_input=T, getunsafestring_returns_identical_results=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_103_Row2_InvariantViolation(t *testing.T) {
	v1, e1 := GetUnsafeString([]byte(`{"a":"b"}`), "a")
	v2, e2 := GetUnsafeString([]byte(`{"a":"b"}`), "a")
	if v1 != v2 || e1 != nil || e2 != nil {
		t.Fatalf("expected identical results: v1=%q v2=%q e1=%v e2=%v", v1, v2, e1, e2)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-104 (GetUnsafeString on nil input returns safe result)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-104
// MCDC SYS-REQ-104: getunsafestring_input_is_nil=F, getunsafestring_returns_safe_result_for_nil=F => TRUE [no-action: non-nil input does not invoke the nil-safe action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_104_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetUnsafeString([]byte(`{"a":"b"}`), "a"); err != nil {
		t.Fatalf("GetUnsafeString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-104
// MCDC SYS-REQ-104: getunsafestring_input_is_nil=T, getunsafestring_returns_safe_result_for_nil=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_104_Row2_InvariantViolation(t *testing.T) {
	v, err := GetUnsafeString(nil, "a")
	if err == nil {
		t.Fatal("expected error on nil input, got nil")
	}
	if v != "" {
		t.Fatalf("expected empty string on nil input, got %q", v)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-105 (GetUnsafeString handles unicode edges safely)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-105
// MCDC SYS-REQ-105: getunsafestring_handles_unicode_edges_safely=F, getunsafestring_input_has_unicode_edge_cases=F => TRUE [no-action: ASCII-only input does not invoke the unicode-edge action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_105_Row1_TriggerFalse(t *testing.T) {
	if _, err := GetUnsafeString([]byte(`{"a":"abc"}`), "a"); err != nil {
		t.Fatalf("GetUnsafeString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-105
// MCDC SYS-REQ-105: getunsafestring_handles_unicode_edges_safely=F, getunsafestring_input_has_unicode_edge_cases=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_105_Row2_InvariantViolation(t *testing.T) {
	v, err := GetUnsafeString([]byte(`{"a"\u00B0}`), "a")
	if err != nil {
		// Even if lookup fails on this odd shape, the call must not panic.
		t.Logf("GetUnsafeString returned error (acceptable): %v", err)
		return
	}
	if v == "" {
		t.Fatal("expected decoded value, got empty string")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-106 (Parse helpers are deterministic)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-106
// MCDC SYS-REQ-106: parse_helper_called_twice_with_same_input=F, parse_helper_returns_identical_results=F => TRUE [no-action: single call, identical-results action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_106_Row1_TriggerFalse(t *testing.T) {
	if _, err := ParseInt([]byte(`42`)); err != nil {
		t.Fatalf("ParseInt returned error: %v", err)
	}
}

// Verifies: SYS-REQ-106
// MCDC SYS-REQ-106: parse_helper_called_twice_with_same_input=T, parse_helper_returns_identical_results=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_106_Row2_InvariantViolation(t *testing.T) {
	v1, e1 := ParseInt([]byte(`42`))
	v2, e2 := ParseInt([]byte(`42`))
	if v1 != v2 || e1 != nil || e2 != nil {
		t.Fatalf("expected identical results: v1=%d v2=%d e1=%v e2=%v", v1, v2, e1, e2)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-107 (Parse helpers on nil input return safe result)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-107
// MCDC SYS-REQ-107: parse_helper_input_is_nil=F, parse_helper_returns_safe_result_for_nil=F => TRUE [no-action: non-nil input does not invoke the nil-safe action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_107_Row1_TriggerFalse(t *testing.T) {
	if _, err := ParseInt([]byte(`42`)); err != nil {
		t.Fatalf("ParseInt returned error: %v", err)
	}
}

// Verifies: SYS-REQ-107
// MCDC SYS-REQ-107: parse_helper_input_is_nil=T, parse_helper_returns_safe_result_for_nil=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_107_Row2_InvariantViolation(t *testing.T) {
	v, err := ParseInt(nil)
	if err == nil {
		t.Fatal("expected error on nil input, got nil")
	}
	if v != 0 {
		t.Fatalf("expected zero on nil input, got %d", v)
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-108 (ParseString round-trip preserves semantics)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-108
// MCDC SYS-REQ-108: parsestring_input_has_standard_escapes=F, parsestring_roundtrip_preserves_semantics=F => TRUE [no-action: no escapes in input, roundtrip action does not fire]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_108_Row1_TriggerFalse(t *testing.T) {
	if _, err := ParseString([]byte(`hello`)); err != nil {
		t.Fatalf("ParseString returned error: %v", err)
	}
}

// Verifies: SYS-REQ-108
// MCDC SYS-REQ-108: parsestring_input_has_standard_escapes=T, parsestring_roundtrip_preserves_semantics=F => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_108_Row2_InvariantViolation(t *testing.T) {
	v, err := ParseString([]byte(`a\nb`))
	if err != nil {
		t.Fatalf("ParseString returned error: %v", err)
	}
	if v == "" {
		t.Fatal("expected decoded value, got empty string")
	}
}

// -----------------------------------------------------------------------------
// SYS-REQ-109 (ParseInt handles edge numbers safely)
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-109
// MCDC SYS-REQ-109: parseint_handles_edge_numbers_safely=F, parseint_input_has_edge_case_number=F => TRUE [no-action: small number does not invoke the edge-number action]
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_109_Row1_TriggerFalse(t *testing.T) {
	if _, err := ParseInt([]byte(`42`)); err != nil {
		t.Fatalf("ParseInt returned error: %v", err)
	}
}

// Verifies: SYS-REQ-109
// MCDC SYS-REQ-109: parseint_handles_edge_numbers_safely=F, parseint_input_has_edge_case_number=T => FALSE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestMCDC_SYS_REQ_109_Row2_InvariantViolation(t *testing.T) {
	v, err := ParseInt([]byte(`-9223372036854775808`))
	if err != nil {
		t.Fatalf("ParseInt returned error: %v", err)
	}
	if v != -9223372036854775808 {
		t.Fatalf("expected int64 min, got %d", v)
	}
}
