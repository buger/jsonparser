package jsonparser

import (
	"bytes"
	"encoding/json"
	"testing"
)

// Verifies: SYS-REQ-009 [example]
// STK-REQ-005:AC-1:acceptance
// SYS-REQ-009:malformed_input:nominal
//   Nominal witness for SYS-REQ-009's malformed_input obligation: a
//   well-formed Set call (valid path, valid value, parent matches the path
//   component kind) returns a parseable mutated JSON document with nil
//   error. The full setTests corpus below covers the happy-path partition
//   symmetric to the KI-3 negative tripwire
//   (TestSetArrayIndexUnderObjectMalformedJSON_KI3).
// MCDC SYS-REQ-009: set_creates_missing_path=F, set_path_is_provided=F, set_returns_not_found_error=F, set_returns_updated_document=F, set_target_exists=F => TRUE
// MCDC SYS-REQ-009: set_creates_missing_path=F, set_path_is_provided=T, set_returns_not_found_error=F, set_returns_updated_document=F, set_target_exists=F => FALSE
// MCDC SYS-REQ-009: set_creates_missing_path=F, set_path_is_provided=T, set_returns_not_found_error=F, set_returns_updated_document=F, set_target_exists=T => TRUE
// MCDC SYS-REQ-009: set_creates_missing_path=F, set_path_is_provided=T, set_returns_not_found_error=F, set_returns_updated_document=T, set_target_exists=F => TRUE
// MCDC SYS-REQ-009: set_creates_missing_path=F, set_path_is_provided=T, set_returns_not_found_error=T, set_returns_updated_document=F, set_target_exists=F => TRUE
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
func TestFuzzSetHarnessCoverage(t *testing.T) {
	if got := FuzzSet([]byte(`{"test":"input"}`)); got != 1 {
		t.Fatalf("expected FuzzSet success path to return 1, got %d", got)
	}
	if got := FuzzSet([]byte(``)); got != 0 {
		t.Fatalf("expected FuzzSet failure path to return 0, got %d", got)
	}
}

// TestSetArrayIndexUnderObjectMalformedJSON_KI3 is the class witness for the
// failure mode tracked as KnownIssue KI-3 / DEFECT-260726-MFPA. Historically,
// when Set received a key path whose [N] component addressed an OBJECT-typed
// parent, the output was malformed JSON (encoding/json rejected it) while Set
// returned nil error. KI-3 is now FIXED via auto-coerce (parser.go:Set): the
// mismatched container is replaced with a fresh container of the type the
// path expects, so the output is always valid JSON.
//
// SYS-REQ-009:malformed_input:negative
//   This witness previously asserted the bug (output was NOT valid JSON). It
//   now asserts the fix: every cross-type (array-index under object / object
//   key under array) Set call returns nil error and parseable JSON output.
//
// Verifies: SYS-REQ-009
//
// Reproduces: KI-3 (fixed)
func TestSetArrayIndexUnderObjectMalformedJSON_KI3(t *testing.T) {
	cases := []struct {
		name string
		in   string
		keys []string
	}{
		{"array_index_under_object_nested", `{"a":{"b":1}}`, []string{"a", "[5]"}},
		{"array_index_under_object_deeper", `{"a":{"b":1}}`, []string{"a", "[0]", "x"}},
		{"array_index_under_object_root", `{"a":1}`, []string{"[0]"}},
		{"empty_brackets_under_object_root", `{"a":1}`, []string{"[]"}},
		{"empty_brackets_after_object_key", `{"a":{"b":1}}`, []string{"a", "[]"}},
		{"empty_brackets_after_array_idx", `{"a":[{"":0}]}`, []string{"a", "[0]", "[]"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := Set([]byte(c.in), []byte(`9`), c.keys...)
			if err != nil {
				t.Fatalf("KI-3 Set(%s,%v) returned error after fix: %v", c.in, c.keys, err)
			}
			// KI-3 fix invariant: the returned bytes MUST be valid JSON.
			if !json.Valid(out) {
				t.Fatalf("KI-3 regressed: Set(%s,%v) -> %s (invalid JSON)", c.in, c.keys, string(out))
			}
			t.Logf("KI-3 fixed: Set(%s,%v) -> %s (valid JSON)", c.in, c.keys, string(out))
		})
	}
}

// TestSetAutoCoerce_KI3 locks the auto-coerce semantics that fix KI-3. When a
// path component expects one container type but the existing structure is the
// other, Set replaces the mismatched container with a fresh container of the
// path's expected type and then performs a normal insertion. Each case asserts
// no error, json.Valid(result), and that re-Get yields the set value.
//
// Verifies: SYS-REQ-009
//
// Reproduces: KI-3 (fixed)
func TestSetAutoCoerce_KI3(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		setValue string
		keys     []string
		getPath  []string
		expected string
	}{
		{"object_root_to_array", `{}`, `9`, []string{"[5]"}, []string{"[0]"}, `[9]`},
		{"array_root_to_object", `[]`, `9`, []string{"key"}, []string{"key"}, `{"key":9}`},
		{"nested_object_to_array", `{"a":{}}`, `9`, []string{"a", "[5]"}, []string{"a", "[0]"}, `{"a":[9]}`},
		{"nested_array_to_object", `{"a":[]}`, `9`, []string{"a", "key"}, []string{"a", "key"}, `{"a":{"key":9}}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := Set([]byte(c.in), []byte(c.setValue), c.keys...)
			if err != nil {
				t.Fatalf("Set(%s,%v) returned error: %v", c.in, c.keys, err)
			}
			if !json.Valid(out) {
				t.Fatalf("Set(%s,%v) -> %s: invalid JSON (KI-3 malformed-output regression)", c.in, c.keys, string(out))
			}
			if string(out) != c.expected {
				t.Fatalf("Set(%s,%v) = %s; expected %s", c.in, c.keys, string(out), c.expected)
			}
			got, _, _, gErr := Get(out, c.getPath...)
			if gErr != nil {
				t.Fatalf("Get(%s,%v) returned error: %v", string(out), c.getPath, gErr)
			}
			if string(got) != c.setValue {
				t.Fatalf("Get(%s,%v) = %q; expected %q", string(out), c.getPath, string(got), c.setValue)
			}
		})
	}
}

// TestDeleteMalformedJSONRegression_SYS_REQ_010 is the Go-level witness for
// SYS-REQ-010's malformed_input obligation. The four FuzzPathMutation corpus
// seeds under testdata/fuzz/FuzzPathMutation/{ed0b39400500dd8f,
// 798e17d6cde9ba4b,7095d73632e4979e,19be37bd98e5d687} exercise the same
// shapes through the fuzz harness; this test makes the obligation-evidence
// triple visible to the audit on stable test names.
//
// SYS-REQ-010:malformed_input:nominal
//   Positive witness: Delete on a well-formed array/object with whitespace
//   in canonical positions returns valid JSON.
//
// SYS-REQ-010:malformed_input:negative
//   Negative witness (now fixed): Delete on the four adversarial whitespace
//   shapes from DEFECT-260726-3PSJ would have produced invalid JSON before
//   the parser.go fix; the same inputs now produce valid JSON (this asserts
//   the fix, NOT the bug — the bug is locked by the fuzz corpus seeds).
func TestDeleteMalformedJSONRegression_SYS_REQ_010(t *testing.T) {
	// Nominal: canonical whitespace, well-formed output.
	t.Run("nominal_canonical_whitespace", func(t *testing.T) {
		out := Delete([]byte(`[0, 1, 2]`), "[1]")
		if !json.Valid(out) {
			t.Fatalf("nominal Delete output invalid: %s", out)
		}
	})

	// Adversarial shapes from DEFECT-260726-3PSJ — all MUST produce valid JSON.
	adversarial := []struct {
		name string
		in   string
	}{
		{"trailing_space_then_close", `[0,0 ]`},
		{"space_comma_middle", `[0,0 ,0]`},
		{"newline_comma_middle", "[0,0\n,0]"},
		{"multi_space_comma_middle", `[0,0  ,0]`},
	}
	for _, c := range adversarial {
		t.Run("adversarial_"+c.name, func(t *testing.T) {
			out := Delete([]byte(c.in), "[1]")
			if !json.Valid(out) {
				t.Errorf("Delete(%q, [1]) -> %q: invalid JSON (DEFECT-260726-3PSJ regressed)", c.in, out)
			}
		})
	}
}
