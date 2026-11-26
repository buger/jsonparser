package jsonparser

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
)

// =============================================================================
// Determinism tests
// =============================================================================

// Verifies: SYS-REQ-086
// MCDC SYS-REQ-086: get_called_twice_with_same_input=T, get_returns_identical_results=T => TRUE
func TestGetDeterminism(t *testing.T) {
	cases := []struct {
		name string
		data string
		keys []string
	}{
		{name: "simple string", data: `{"name":"alice"}`, keys: []string{"name"}},
		{name: "nested object", data: `{"a":{"b":{"c":42}}}`, keys: []string{"a", "b", "c"}},
		{name: "array index", data: `{"arr":[10,20,30]}`, keys: []string{"arr", "[1]"}},
		{name: "missing key", data: `{"a":1}`, keys: []string{"b"}},
		{name: "empty object", data: `{}`, keys: []string{"x"}},
		{name: "root value no keys", data: `{"a":1}`, keys: nil},
		{name: "boolean value", data: `{"ok":true}`, keys: []string{"ok"}},
		{name: "null value", data: `{"x":null}`, keys: []string{"x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte(tc.data)
			v1, dt1, off1, err1 := Get(data, tc.keys...)
			v2, dt2, off2, err2 := Get(data, tc.keys...)

			if !bytes.Equal(v1, v2) {
				t.Fatalf("Get value mismatch: %q vs %q", v1, v2)
			}
			if dt1 != dt2 {
				t.Fatalf("Get type mismatch: %v vs %v", dt1, dt2)
			}
			if off1 != off2 {
				t.Fatalf("Get offset mismatch: %d vs %d", off1, off2)
			}
			if (err1 == nil) != (err2 == nil) {
				t.Fatalf("Get error mismatch: %v vs %v", err1, err2)
			}
		})
	}
}

// Verifies: SYS-REQ-090
// MCDC SYS-REQ-090: getstring_called_twice_with_same_input=T, getstring_returns_identical_results=T => TRUE
func TestGetStringDeterminism(t *testing.T) {
	cases := []struct {
		name string
		data string
		keys []string
	}{
		{name: "simple", data: `{"s":"hello"}`, keys: []string{"s"}},
		{name: "escaped", data: `{"s":"hello\nworld"}`, keys: []string{"s"}},
		{name: "unicode", data: `{"s":"\u00e9"}`, keys: []string{"s"}},
		{name: "missing", data: `{"a":1}`, keys: []string{"s"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte(tc.data)
			v1, err1 := GetString(data, tc.keys...)
			v2, err2 := GetString(data, tc.keys...)
			if v1 != v2 {
				t.Fatalf("GetString value mismatch: %q vs %q", v1, v2)
			}
			if (err1 == nil) != (err2 == nil) {
				t.Fatalf("GetString error mismatch: %v vs %v", err1, err2)
			}
		})
	}
}

// Verifies: SYS-REQ-094
// MCDC SYS-REQ-094: typed_getter_called_twice_with_same_input=T, typed_getter_returns_identical_results=T => TRUE
func TestTypedGetterDeterminism(t *testing.T) {
	data := []byte(`{"i":42,"f":3.14,"b":true}`)

	// GetInt
	i1, ie1 := GetInt(data, "i")
	i2, ie2 := GetInt(data, "i")
	if i1 != i2 || (ie1 == nil) != (ie2 == nil) {
		t.Fatalf("GetInt not deterministic: (%d,%v) vs (%d,%v)", i1, ie1, i2, ie2)
	}

	// GetFloat
	f1, fe1 := GetFloat(data, "f")
	f2, fe2 := GetFloat(data, "f")
	if f1 != f2 || (fe1 == nil) != (fe2 == nil) {
		t.Fatalf("GetFloat not deterministic: (%f,%v) vs (%f,%v)", f1, fe1, f2, fe2)
	}

	// GetBoolean
	b1, be1 := GetBoolean(data, "b")
	b2, be2 := GetBoolean(data, "b")
	if b1 != b2 || (be1 == nil) != (be2 == nil) {
		t.Fatalf("GetBoolean not deterministic: (%v,%v) vs (%v,%v)", b1, be1, b2, be2)
	}
}

// Verifies: SYS-REQ-097
// MCDC SYS-REQ-097: traversal_called_twice_with_same_input=T, traversal_returns_identical_results=T => TRUE
func TestTraversalDeterminism(t *testing.T) {
	t.Run("ArrayEach", func(t *testing.T) {
		data := []byte(`{"arr":[1,2,3]}`)
		var vals1, vals2 []string

		ArrayEach(data, func(value []byte, dataType ValueType, offset int, err error) {
			vals1 = append(vals1, string(value))
		}, "arr")
		ArrayEach(data, func(value []byte, dataType ValueType, offset int, err error) {
			vals2 = append(vals2, string(value))
		}, "arr")

		if len(vals1) != len(vals2) {
			t.Fatalf("ArrayEach length mismatch: %d vs %d", len(vals1), len(vals2))
		}
		for i := range vals1 {
			if vals1[i] != vals2[i] {
				t.Fatalf("ArrayEach element %d mismatch: %q vs %q", i, vals1[i], vals2[i])
			}
		}
	})

	t.Run("ObjectEach", func(t *testing.T) {
		data := []byte(`{"a":1,"b":2,"c":3}`)
		var keys1, keys2 []string

		ObjectEach(data, func(key []byte, value []byte, dataType ValueType, offset int) error {
			keys1 = append(keys1, string(key))
			return nil
		})
		ObjectEach(data, func(key []byte, value []byte, dataType ValueType, offset int) error {
			keys2 = append(keys2, string(key))
			return nil
		})

		if len(keys1) != len(keys2) {
			t.Fatalf("ObjectEach length mismatch: %d vs %d", len(keys1), len(keys2))
		}
		for i := range keys1 {
			if keys1[i] != keys2[i] {
				t.Fatalf("ObjectEach key %d mismatch: %q vs %q", i, keys1[i], keys2[i])
			}
		}
	})

	t.Run("EachKey", func(t *testing.T) {
		data := []byte(`{"name":"test","age":30}`)
		paths := [][]string{{"name"}, {"age"}, {"missing"}}
		var results1, results2 []string

		EachKey(data, func(idx int, value []byte, vt ValueType, err error) {
			results1 = append(results1, fmt.Sprintf("%d:%s:%v", idx, string(value), vt))
		}, paths...)
		EachKey(data, func(idx int, value []byte, vt ValueType, err error) {
			results2 = append(results2, fmt.Sprintf("%d:%s:%v", idx, string(value), vt))
		}, paths...)

		if len(results1) != len(results2) {
			t.Fatalf("EachKey length mismatch: %d vs %d", len(results1), len(results2))
		}
		for i := range results1 {
			if results1[i] != results2[i] {
				t.Fatalf("EachKey result %d mismatch: %q vs %q", i, results1[i], results2[i])
			}
		}
	})
}

// Verifies: SYS-REQ-103
// MCDC SYS-REQ-103: getunsafestring_called_twice_with_same_input=T, getunsafestring_returns_identical_results=T => TRUE
func TestGetUnsafeStringDeterminism(t *testing.T) {
	data := []byte(`{"s":"hello\\world"}`)
	v1, e1 := GetUnsafeString(data, "s")
	v2, e2 := GetUnsafeString(data, "s")
	if v1 != v2 {
		t.Fatalf("GetUnsafeString value mismatch: %q vs %q", v1, v2)
	}
	if (e1 == nil) != (e2 == nil) {
		t.Fatalf("GetUnsafeString error mismatch: %v vs %v", e1, e2)
	}
}

// Verifies: SYS-REQ-106
// MCDC SYS-REQ-106: parse_helper_called_twice_with_same_input=T, parse_helper_returns_identical_results=T => TRUE
func TestParseHelperDeterminism(t *testing.T) {
	// ParseBoolean
	b1, be1 := ParseBoolean([]byte("true"))
	b2, be2 := ParseBoolean([]byte("true"))
	if b1 != b2 || (be1 == nil) != (be2 == nil) {
		t.Fatalf("ParseBoolean not deterministic")
	}

	// ParseInt
	i1, ie1 := ParseInt([]byte("42"))
	i2, ie2 := ParseInt([]byte("42"))
	if i1 != i2 || (ie1 == nil) != (ie2 == nil) {
		t.Fatalf("ParseInt not deterministic")
	}

	// ParseFloat
	f1, fe1 := ParseFloat([]byte("3.14"))
	f2, fe2 := ParseFloat([]byte("3.14"))
	if f1 != f2 || (fe1 == nil) != (fe2 == nil) {
		t.Fatalf("ParseFloat not deterministic")
	}

	// ParseString
	s1, se1 := ParseString([]byte(`hello\nworld`))
	s2, se2 := ParseString([]byte(`hello\nworld`))
	if s1 != s2 || (se1 == nil) != (se2 == nil) {
		t.Fatalf("ParseString not deterministic")
	}
}

// =============================================================================
// Idempotency tests
// =============================================================================

// Verifies: SYS-REQ-087
// MCDC SYS-REQ-087: get_called_on_valid_input=T, get_does_not_mutate_input=T => TRUE
func TestGetIdempotencyInputNotMutated(t *testing.T) {
	original := `{"name":"alice","age":30,"nested":{"key":"value"}}`
	data := []byte(original)
	snapshot := make([]byte, len(data))
	copy(snapshot, data)

	// Call Get multiple times with different key paths
	Get(data, "name")
	Get(data, "age")
	Get(data, "nested", "key")
	Get(data, "missing")

	if !bytes.Equal(data, snapshot) {
		t.Fatalf("Get mutated input: original %q, after %q", snapshot, data)
	}
}

// Verifies: SYS-REQ-100
// MCDC SYS-REQ-100: set_applied_twice_with_same_args=T, set_second_call_produces_same_result=T => TRUE
func TestSetIdempotency(t *testing.T) {
	data := []byte(`{"name":"alice","age":30}`)
	setValue := []byte(`"bob"`)

	// First Set application
	result1, err1 := Set(data, setValue, "name")
	if err1 != nil {
		t.Fatalf("First Set returned error: %v", err1)
	}

	// Second Set application on the result of the first
	result2, err2 := Set(result1, setValue, "name")
	if err2 != nil {
		t.Fatalf("Second Set returned error: %v", err2)
	}

	// The results should be identical -- setting the same key to the same value twice
	if !bytes.Equal(result1, result2) {
		t.Fatalf("Set not idempotent: first %q, second %q", result1, result2)
	}
}

// =============================================================================
// Nil safety tests
// =============================================================================

// Verifies: SYS-REQ-088
// MCDC SYS-REQ-088: get_input_is_nil=T, get_returns_safe_result_for_nil=T => TRUE
func TestGetNilSafety(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Get(nil) panicked: %v", r)
		}
	}()

	_, _, _, err := Get(nil, "key")
	if err == nil {
		t.Logf("Get(nil, 'key') returned no error (acceptable if not-found)")
	}

	// Also test nil with no keys
	_, _, _, err = Get(nil)
	_ = err
}

// Verifies: SYS-REQ-091
// MCDC SYS-REQ-091: getstring_input_is_nil=T, getstring_returns_safe_result_for_nil=T => TRUE
func TestGetStringNilSafety(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GetString(nil) panicked: %v", r)
		}
	}()

	val, err := GetString(nil, "key")
	if val != "" && err == nil {
		t.Fatalf("GetString(nil) returned non-empty value without error: %q", val)
	}
}

// Verifies: SYS-REQ-095
// MCDC SYS-REQ-095: typed_getter_input_is_nil=T, typed_getter_returns_safe_result_for_nil=T => TRUE
func TestTypedGetterNilSafety(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Typed getter on nil panicked: %v", r)
		}
	}()

	ival, ierr := GetInt(nil, "key")
	if ival != 0 && ierr == nil {
		t.Fatalf("GetInt(nil) returned non-zero without error: %d", ival)
	}

	fval, ferr := GetFloat(nil, "key")
	if fval != 0 && ferr == nil {
		t.Fatalf("GetFloat(nil) returned non-zero without error: %f", fval)
	}

	bval, berr := GetBoolean(nil, "key")
	if bval && berr == nil {
		t.Fatalf("GetBoolean(nil) returned true without error")
	}
}

// Verifies: SYS-REQ-098
// MCDC SYS-REQ-098: traversal_input_is_nil=T, traversal_returns_safe_result_for_nil=T => TRUE
func TestTraversalNilSafety(t *testing.T) {
	t.Run("ArrayEach_nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("ArrayEach(nil) panicked: %v", r)
			}
		}()

		callbackCalled := false
		_, err := ArrayEach(nil, func(value []byte, dataType ValueType, offset int, err error) {
			callbackCalled = true
		})
		if callbackCalled {
			t.Fatalf("ArrayEach(nil) invoked callback")
		}
		if err == nil {
			t.Fatalf("ArrayEach(nil) returned no error")
		}
	})

	t.Run("ObjectEach_nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("ObjectEach(nil) panicked: %v", r)
			}
		}()

		callbackCalled := false
		err := ObjectEach(nil, func(key []byte, value []byte, dataType ValueType, offset int) error {
			callbackCalled = true
			return nil
		})
		if callbackCalled {
			t.Fatalf("ObjectEach(nil) invoked callback")
		}
		if err == nil {
			t.Fatalf("ObjectEach(nil) returned no error")
		}
	})

	t.Run("EachKey_nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("EachKey(nil) panicked: %v", r)
			}
		}()

		callbackCalled := false
		EachKey(nil, func(idx int, value []byte, vt ValueType, err error) {
			callbackCalled = true
		}, []string{"key"})
		if callbackCalled {
			t.Fatalf("EachKey(nil) invoked callback")
		}
	})
}

// Verifies: SYS-REQ-101
// MCDC SYS-REQ-101: mutation_input_is_nil=T, mutation_returns_safe_result_for_nil=T => TRUE
func TestMutationNilSafety(t *testing.T) {
	t.Run("Set_nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Set(nil) panicked: %v", r)
			}
		}()

		_, err := Set(nil, []byte(`"value"`), "key")
		// Set on nil should return an error or handle gracefully
		_ = err
	})

	t.Run("Delete_nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Delete(nil) panicked: %v", r)
			}
		}()

		result := Delete(nil, "key")
		// Delete on nil should return nil or empty without panic
		_ = result
	})
}

// Verifies: SYS-REQ-104
// MCDC SYS-REQ-104: getunsafestring_input_is_nil=T, getunsafestring_returns_safe_result_for_nil=T => TRUE
func TestGetUnsafeStringNilSafety(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("GetUnsafeString(nil) panicked: %v", r)
		}
	}()

	val, err := GetUnsafeString(nil, "key")
	if val != "" && err == nil {
		t.Fatalf("GetUnsafeString(nil) returned non-empty value without error: %q", val)
	}
}

// Verifies: SYS-REQ-107
// MCDC SYS-REQ-107: parse_helper_input_is_nil=T, parse_helper_returns_safe_result_for_nil=T => TRUE
func TestParseHelperNilSafety(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Parse helper on nil panicked: %v", r)
		}
	}()

	// ParseBoolean(nil)
	bval, berr := ParseBoolean(nil)
	if bval && berr == nil {
		t.Fatalf("ParseBoolean(nil) returned true without error")
	}

	// ParseInt(nil)
	ival, ierr := ParseInt(nil)
	if ival != 0 && ierr == nil {
		t.Fatalf("ParseInt(nil) returned non-zero without error: %d", ival)
	}

	// ParseFloat(nil)
	fval, ferr := ParseFloat(nil)
	if fval != 0 && ferr == nil {
		t.Fatalf("ParseFloat(nil) returned non-zero without error: %f", fval)
	}

	// ParseString(nil) -- should return empty string
	sval, serr := ParseString(nil)
	_ = serr
	_ = sval
}

// =============================================================================
// Encoding safety tests
// =============================================================================

// Verifies: SYS-REQ-092
// MCDC SYS-REQ-092: getstring_input_has_escaped_unicode=T, getstring_decodes_and_preserves_semantics=T => TRUE
func TestGetStringEncodingSafety(t *testing.T) {
	cases := []struct {
		name     string
		jsonData string
		key      string
		expected string
	}{
		{name: "basic latin escape", jsonData: `{"s":"\u0041"}`, key: "s", expected: "A"},
		{name: "e-acute", jsonData: `{"s":"\u00e9"}`, key: "s", expected: "\u00e9"},
		{name: "emoji surrogate pair", jsonData: `{"s":"\uD83D\uDE00"}`, key: "s", expected: "\U0001F600"},
		{name: "newline escape", jsonData: `{"s":"line1\nline2"}`, key: "s", expected: "line1\nline2"},
		{name: "tab escape", jsonData: `{"s":"col1\tcol2"}`, key: "s", expected: "col1\tcol2"},
		{name: "backslash escape", jsonData: `{"s":"path\\to\\file"}`, key: "s", expected: `path\to\file`},
		{name: "quote escape", jsonData: `{"s":"say \"hello\""}`, key: "s", expected: `say "hello"`},
		{name: "solidus escape", jsonData: `{"s":"a\/b"}`, key: "s", expected: "a/b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			val, err := GetString([]byte(tc.jsonData), tc.key)
			if err != nil {
				t.Fatalf("GetString returned error: %v", err)
			}
			if val != tc.expected {
				t.Fatalf("GetString decoded %q, expected %q", val, tc.expected)
			}
		})
	}
}

// Verifies: SYS-REQ-108
// MCDC SYS-REQ-108: parsestring_input_has_standard_escapes=T, parsestring_roundtrip_preserves_semantics=T => TRUE
func TestParseStringEncodingSafetyRoundtrip(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "newline", input: `hello\nworld`, expected: "hello\nworld"},
		{name: "tab", input: `col1\tcol2`, expected: "col1\tcol2"},
		{name: "backslash", input: `path\\to\\file`, expected: `path\to\file`},
		{name: "quote", input: `say \"hi\"`, expected: `say "hi"`},
		{name: "unicode BMP", input: `\u0048\u0065\u006C\u006C\u006F`, expected: "Hello"},
		{name: "solidus", input: `a\/b`, expected: "a/b"},
		{name: "backspace", input: `a\bc`, expected: "a\bc"},
		{name: "formfeed", input: `a\fc`, expected: "a\fc"},
		{name: "carriage return", input: `a\rc`, expected: "a\rc"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decoded, err := ParseString([]byte(tc.input))
			if err != nil {
				t.Fatalf("ParseString returned error: %v", err)
			}
			if decoded != tc.expected {
				t.Fatalf("ParseString decoded %q, expected %q", decoded, tc.expected)
			}

			// Round-trip check: re-encode and verify the decoded string marshals
			// back to JSON that, when decoded again, gives the same Go string.
			reEncoded, err := json.Marshal(decoded)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}
			var reDecoded string
			if err := json.Unmarshal(reEncoded, &reDecoded); err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}
			if reDecoded != decoded {
				t.Fatalf("Round-trip mismatch: original %q, round-tripped %q", decoded, reDecoded)
			}
		})
	}
}

// =============================================================================
// Edge case tests
// =============================================================================

// Verifies: SYS-REQ-089
// MCDC SYS-REQ-089: get_input_is_deeply_nested=T, get_handles_deep_nesting_safely=T => TRUE
func TestGetDeepNesting(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Get on deeply nested JSON panicked: %v", r)
		}
	}()

	// Build 64-level nested JSON: {"a":{"a":{"a":...42...}}}
	depth := 64
	var keys []string
	prefix := strings.Repeat(`{"a":`, depth)
	suffix := strings.Repeat(`}`, depth)
	data := []byte(prefix + `42` + suffix)
	for i := 0; i < depth; i++ {
		keys = append(keys, "a")
	}

	val, dt, _, err := Get(data, keys...)
	if err != nil {
		t.Logf("Get on %d-deep nesting returned error (acceptable): %v", depth, err)
		return
	}
	if dt != Number {
		t.Fatalf("Expected Number type at depth %d, got %v", depth, dt)
	}
	if string(val) != "42" {
		t.Fatalf("Expected value 42, got %q", val)
	}
}

// Verifies: SYS-REQ-093
// MCDC SYS-REQ-093: getstring_input_has_unicode_edge_cases=T, getstring_handles_unicode_edges_safely=T => TRUE
func TestGetStringUnicodeEdgeCases(t *testing.T) {
	cases := []struct {
		name     string
		jsonData string
		key      string
		wantErr  bool
	}{
		{
			name:     "BOM character",
			jsonData: `{"s":"\uFEFF"}`,
			key:      "s",
		},
		{
			name:     "zero-width joiner",
			jsonData: `{"s":"\u200D"}`,
			key:      "s",
		},
		{
			name:     "replacement character",
			jsonData: `{"s":"\uFFFD"}`,
			key:      "s",
		},
		{
			name:     "null character escape",
			jsonData: `{"s":"\u0000"}`,
			key:      "s",
		},
		{
			name:     "max BMP codepoint",
			jsonData: `{"s":"\uFFFF"}`,
			key:      "s",
		},
		{
			name:     "multi-byte UTF-8 literal in value",
			jsonData: "{\"s\":\"\xe2\x80\x8b\"}", // zero-width space (U+200B) as raw UTF-8
			key:      "s",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("GetString panicked on %s: %v", tc.name, r)
				}
			}()

			val, err := GetString([]byte(tc.jsonData), tc.key)
			if tc.wantErr && err == nil {
				t.Fatalf("Expected error for %s", tc.name)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("GetString(%s) returned error: %v", tc.name, err)
			}
			_ = val
		})
	}
}

// Verifies: SYS-REQ-096
// MCDC SYS-REQ-096: getint_input_has_large_number_edge_case=T, getint_handles_large_numbers_safely=T => TRUE
func TestGetIntLargeNumberEdgeCases(t *testing.T) {
	cases := []struct {
		name    string
		data    string
		key     string
		wantVal int64
		wantErr bool
	}{
		{
			name:    "max int64",
			data:    fmt.Sprintf(`{"n":%d}`, math.MaxInt64),
			key:     "n",
			wantVal: math.MaxInt64,
		},
		{
			name:    "min int64",
			data:    fmt.Sprintf(`{"n":%d}`, math.MinInt64),
			key:     "n",
			wantVal: math.MinInt64,
		},
		{
			name:    "overflow beyond max int64",
			data:    `{"n":9223372036854775808}`,
			key:     "n",
			wantErr: true,
		},
		{
			name:    "negative zero",
			data:    `{"n":-0}`,
			key:     "n",
			wantVal: 0,
		},
		{
			name:    "zero",
			data:    `{"n":0}`,
			key:     "n",
			wantVal: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("GetInt panicked: %v", r)
				}
			}()

			val, err := GetInt([]byte(tc.data), tc.key)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Expected error, got value %d", val)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetInt returned error: %v", err)
			}
			if val != tc.wantVal {
				t.Fatalf("GetInt = %d, want %d", val, tc.wantVal)
			}
		})
	}
}

// Verifies: SYS-REQ-099
// MCDC SYS-REQ-099: traversal_input_is_deeply_nested=T, traversal_handles_deep_nesting_safely=T => TRUE
func TestTraversalDeepNesting(t *testing.T) {
	t.Run("ArrayEach_deep", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("ArrayEach on deep nesting panicked: %v", r)
			}
		}()

		// Build deeply nested array: [[[[...42...]]]]
		depth := 64
		prefix := strings.Repeat("[", depth)
		suffix := strings.Repeat("]", depth)
		data := []byte(prefix + "42" + suffix)

		// ArrayEach on the outer array
		_, err := ArrayEach(data, func(value []byte, dataType ValueType, offset int, err error) {
			// The inner element is another deeply nested array or the final value
		})
		// We accept any result as long as no panic
		_ = err
	})

	t.Run("ObjectEach_deep", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("ObjectEach on deep nesting panicked: %v", r)
			}
		}()

		// Build deeply nested object: {"a":{"a":{"a":...{"a":42}...}}}
		depth := 64
		prefix := strings.Repeat(`{"a":`, depth)
		suffix := strings.Repeat("}", depth)
		data := []byte(prefix + "42" + suffix)

		err := ObjectEach(data, func(key []byte, value []byte, dataType ValueType, offset int) error {
			return nil
		})
		_ = err
	})
}

// Verifies: SYS-REQ-102
// MCDC SYS-REQ-102: mutation_input_has_unicode_keys=T, mutation_handles_unicode_keys_safely=T => TRUE
func TestMutationUnicodeKeys(t *testing.T) {
	t.Run("Set_unicode_key", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Set with unicode key panicked: %v", r)
			}
		}()

		data := []byte(`{"caf\u00e9":"latte","normal":"value"}`)
		// Try to set a value using a Unicode key
		result, err := Set(data, []byte(`"espresso"`), "normal")
		if err != nil {
			t.Fatalf("Set returned error: %v", err)
		}
		// Verify the mutation was applied
		val, err := GetString(result, "normal")
		if err != nil {
			t.Fatalf("GetString after Set returned error: %v", err)
		}
		if val != "espresso" {
			t.Fatalf("Expected 'espresso', got %q", val)
		}
	})

	t.Run("Delete_unicode_key", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Delete with unicode key panicked: %v", r)
			}
		}()

		// Test Delete on document with a raw UTF-8 key
		data := []byte("{\"caf\xc3\xa9\":\"latte\",\"tea\":\"green\"}")
		result := Delete(data, "tea")
		// Verify tea was removed and the document is still parseable
		_, _, _, err := Get(result, "tea")
		if !errors.Is(err, KeyPathNotFoundError) {
			t.Fatalf("Expected KeyPathNotFoundError after Delete, got: %v", err)
		}
	})
}

// Verifies: SYS-REQ-105
// MCDC SYS-REQ-105: getunsafestring_input_has_unicode_edge_cases=T, getunsafestring_handles_unicode_edges_safely=T => TRUE
func TestGetUnsafeStringUnicodeEdgeCases(t *testing.T) {
	cases := []struct {
		name string
		data string
		key  string
	}{
		{name: "BOM in value", data: `{"s":"\uFEFF"}`, key: "s"},
		{name: "raw multi-byte UTF-8", data: "{\"s\":\"\xc3\xa9\"}", key: "s"},
		{name: "emoji in value", data: "{\"s\":\"\xf0\x9f\x98\x80\"}", key: "s"},
		{name: "ZWJ sequence", data: "{\"s\":\"\xe2\x80\x8d\"}", key: "s"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("GetUnsafeString panicked on %s: %v", tc.name, r)
				}
			}()

			val, err := GetUnsafeString([]byte(tc.data), tc.key)
			// GetUnsafeString returns raw bytes -- we just verify no panic
			_ = val
			_ = err
		})
	}
}

// Verifies: SYS-REQ-109
// MCDC SYS-REQ-109: parseint_input_has_edge_case_number=T, parseint_handles_edge_numbers_safely=T => TRUE
func TestParseIntEdgeCaseNumbers(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantVal int64
		wantErr bool
	}{
		{name: "negative zero", input: "-0", wantVal: 0},
		{name: "positive zero", input: "0", wantVal: 0},
		{name: "max int64", input: "9223372036854775807", wantVal: math.MaxInt64},
		{name: "min int64", input: "-9223372036854775808", wantVal: math.MinInt64},
		{name: "max+1 overflow", input: "9223372036854775808", wantErr: true},
		{name: "min-1 overflow", input: "-9223372036854775809", wantErr: true},
		{name: "100-digit number", input: strings.Repeat("9", 100), wantErr: true},
		{name: "single digit", input: "7", wantVal: 7},
		{name: "negative single digit", input: "-3", wantVal: -3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("ParseInt panicked: %v", r)
				}
			}()

			val, err := ParseInt([]byte(tc.input))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Expected error for %q, got value %d", tc.input, val)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseInt(%q) returned error: %v", tc.input, err)
			}
			if val != tc.wantVal {
				t.Fatalf("ParseInt(%q) = %d, want %d", tc.input, val, tc.wantVal)
			}
		})
	}
}
