package jsonparser

import (
	"bytes"
	"testing"
)

// Verifies: SYS-REQ-009 [example]
// STK-REQ-005:AC-1:acceptance
// MCDC SYS-REQ-009: set_creates_missing_path=F, set_path_is_provided=F, set_returns_not_found_error=F, set_returns_updated_document=F, set_target_exists=F => TRUE
// MCDC SYS-REQ-009: set_creates_missing_path=F, set_path_is_provided=T, set_returns_not_found_error=F, set_returns_updated_document=F, set_target_exists=F => FALSE
// MCDC SYS-REQ-009: set_creates_missing_path=F, set_path_is_provided=T, set_returns_not_found_error=F, set_returns_updated_document=F, set_target_exists=T => TRUE
// MCDC SYS-REQ-009: set_creates_missing_path=F, set_path_is_provided=T, set_returns_not_found_error=F, set_returns_updated_document=T, set_target_exists=F => TRUE
// MCDC SYS-REQ-009: set_creates_missing_path=F, set_path_is_provided=T, set_returns_not_found_error=T, set_returns_updated_document=F, set_target_exists=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestSet(t *testing.T) {
	runSetTests(t, "Set()", setTests,
		func(test SetTest) (value interface{}, dataType ValueType, err error) {
			value, err = Set([]byte(test.json), []byte(test.setData), test.path...)
			return
		},
		func(test SetTest, value interface{}) (bool, interface{}) {
			expected := []byte(test.data.(string))
			return bytes.Equal(expected, value.([]byte)), expected
		},
	)
}

// Verifies: SYS-REQ-009 [boundary]
// MCDC SYS-REQ-009: set_creates_missing_path=T, set_path_is_provided=T, set_returns_not_found_error=F, set_returns_updated_document=F, set_target_exists=F => TRUE
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestSetCreatesMissingEntryInExistingArray(t *testing.T) {
	value, err := Set(
		[]byte(`{"top":[{"middle":[{"present":true}]}]}`),
		[]byte(`{"bottom":"value"}`),
		"top", "[0]", "middle", "[1]",
	)
	if err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	expected := `{"top":[{"middle":[{"present":true},{"bottom":"value"}]}]}`
	if string(value) != expected {
		t.Fatalf("Set result mismatch: expected %s, got %s", expected, string(value))
	}
}

// Verifies: SYS-REQ-009 [fuzz]
// MCDC SYS-REQ-009: N/A
// reqproof:proptest:skip test-case harness function; is itself a unit/integration test, not a pure function amenable to property-based testing
func TestFuzzSetHarnessCoverage(t *testing.T) {
	if got := FuzzSet([]byte(`{"test":"input"}`)); got != 1 {
		t.Fatalf("expected FuzzSet success path to return 1, got %d", got)
	}
	if got := FuzzSet([]byte(``)); got != 0 {
		t.Fatalf("expected FuzzSet failure path to return 0, got %d", got)
	}
}
