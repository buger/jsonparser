package jsonparser

import (
	"errors"
	"testing"
)

// Regression coverage for the empty-string key path component hazard
// (hazard-sweep finding). A caller passing "" as a path component used to
// trigger `runtime error: index out of range [0] with length 0` at the
// unguarded `keys[i][0]` / `p[level][0]` dereference sites in searchKeys,
// EachKey, createInsertComponent, and calcAllocateSpace. The fix adds the
// same `len(...) > 0` guard that already existed in Delete (parser.go:835),
// routing an empty key component through the existing not-found / no-callback
// / unchanged-payload path instead of panicking.
//
// These cases assert panic-free degradation: each entry MUST surface a typed
// not-found outcome (or, for Set, produce a defined document) rather than
// crashing the goroutine.

// runNoPanic executes fn and fails the test if it panics, returning the
// recovered value so callers can also assert on the post-fix result.
func runNoPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked (empty-key regression): %v", name, r)
		}
	}()
	fn()
}

// =============================================================================
// Get family — empty key component resolves to KeyPathNotFoundError (SYS-REQ-016)
// =============================================================================

// Verifies: SYS-REQ-016 [boundary]
// An empty-string path component is not a resolvable object key or array index;
// Get must surface KeyPathNotFoundError rather than panicking.
func TestGetEmptyKeyPathComponent(t *testing.T) {
	cases := []struct {
		name string
		data string
		keys []string
	}{
		{name: "empty component on array root", data: `[1,2,3]`, keys: []string{""}},
		{name: "empty component on object root", data: `{"a":1}`, keys: []string{""}},
		{name: "empty component after valid key", data: `{"a":[1]}`, keys: []string{"a", ""}},
		{name: "empty component before valid key", data: `{"a":1}`, keys: []string{"", "a"}},
		{name: "two empty components", data: `{"a":1}`, keys: []string{"", ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				val []byte
				dt  ValueType
				off int
				err error
			)
			runNoPanic(t, tc.name, func() {
				val, dt, off, err = Get([]byte(tc.data), tc.keys...)
			})
			if !errors.Is(err, KeyPathNotFoundError) {
				t.Fatalf("Get(%q,%v) err = %v, want KeyPathNotFoundError", tc.data, tc.keys, err)
			}
			if dt != NotExist {
				t.Fatalf("Get(%q,%v) type = %v, want NotExist", tc.data, tc.keys, dt)
			}
			if off != -1 {
				t.Fatalf("Get(%q,%v) offset = %d, want -1", tc.data, tc.keys, off)
			}
			if val != nil {
				t.Fatalf("Get(%q,%v) value = %v, want nil", tc.data, tc.keys, val)
			}
		})
	}
}

// Verifies: SYS-REQ-016 [boundary]
// Typed Get accessors must propagate KeyPathNotFoundError for an empty key
// component instead of panicking on the underlying searchKeys dereference.
func TestTypedGetEmptyKeyPathComponent(t *testing.T) {
	t.Run("GetString", func(t *testing.T) {
		var err error
		runNoPanic(t, "GetString", func() {
			_, err = GetString([]byte(`{"a":"x"}`), "")
		})
		if !errors.Is(err, KeyPathNotFoundError) {
			t.Fatalf("GetString empty-key err = %v, want KeyPathNotFoundError", err)
		}
	})
	t.Run("GetInt", func(t *testing.T) {
		var err error
		runNoPanic(t, "GetInt", func() {
			_, err = GetInt([]byte(`{"a":1}`), "")
		})
		if !errors.Is(err, KeyPathNotFoundError) {
			t.Fatalf("GetInt empty-key err = %v, want KeyPathNotFoundError", err)
		}
	})
	t.Run("GetFloat", func(t *testing.T) {
		var err error
		runNoPanic(t, "GetFloat", func() {
			_, err = GetFloat([]byte(`{"a":1.5}`), "")
		})
		if !errors.Is(err, KeyPathNotFoundError) {
			t.Fatalf("GetFloat empty-key err = %v, want KeyPathNotFoundError", err)
		}
	})
	t.Run("GetBoolean", func(t *testing.T) {
		var err error
		runNoPanic(t, "GetBoolean", func() {
			_, err = GetBoolean([]byte(`{"a":true}`), "")
		})
		if !errors.Is(err, KeyPathNotFoundError) {
			t.Fatalf("GetBoolean empty-key err = %v, want KeyPathNotFoundError", err)
		}
	})
	t.Run("GetUnsafeString", func(t *testing.T) {
		var err error
		runNoPanic(t, "GetUnsafeString", func() {
			_, err = GetUnsafeString([]byte(`{"a":"x"}`), "")
		})
		if !errors.Is(err, KeyPathNotFoundError) {
			t.Fatalf("GetUnsafeString empty-key err = %v, want KeyPathNotFoundError", err)
		}
	})
}

// =============================================================================
// EachKey — empty key component emits no callback (SYS-REQ-008)
// =============================================================================

// Verifies: SYS-REQ-008 [boundary]
// An empty-string path component cannot address an array index, so EachKey
// must skip the path (missing-request => no callback) and must not panic on
// the `p[level][0]` dereference.
func TestEachKeyEmptyKeyPathComponent(t *testing.T) {
	cases := []struct {
		name  string
		data  string
		paths [][]string
	}{
		{name: "single empty path on array root", data: `[1,2,3]`, paths: [][]string{{""}}},
		{name: "empty path mixed with valid path", data: `{"a":1,"b":2}`, paths: [][]string{{""}, {"a"}}},
		{name: "empty leading component", data: `{"a":1}`, paths: [][]string{{"", "a"}}},
		{name: "empty trailing component", data: `{"a":[1]}`, paths: [][]string{{"a", ""}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			emptyCalled := false
			runNoPanic(t, tc.name, func() {
				EachKey([]byte(tc.data), func(idx int, val []byte, dt ValueType, err error) {
					// Callbacks may fire for the valid path, but the empty path
					// must never resolve to a real callback slot.
					if len(tc.paths[idx]) == 0 || tc.paths[idx][0] == "" {
						emptyCalled = true
					}
				}, tc.paths...)
			})
			if emptyCalled {
				t.Fatalf("EachKey(%q,%v) emitted a callback for an empty path component", tc.data, tc.paths)
			}
			// No assertion on the valid-path callback: it may or may not fire
			// depending on the case; the regression gate is "no panic, no
			// callback for the empty component".
		})
	}
}

// =============================================================================
// Set — empty key component produces a defined document, no panic (SYS-REQ-009)
// =============================================================================

// Verifies: SYS-REQ-009 [boundary]
// Set with an empty-string key component must not panic in
// createInsertComponent / calcAllocateSpace. The empty key is treated as an
// object property name (not an array index) and produces a defined document.
func TestSetEmptyKeyPathComponent(t *testing.T) {
	cases := []struct {
		name    string
		data    string
		setData string
		keys    []string
	}{
		{name: "single empty key on empty object", data: `{}`, setData: `"v"`, keys: []string{""}},
		{name: "single empty key on populated object", data: `{"a":1}`, setData: `"v"`, keys: []string{""}},
		{name: "empty key leading", data: `{}`, setData: `"v"`, keys: []string{"", "a"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				val []byte
				err error
			)
			runNoPanic(t, tc.name, func() {
				val, err = Set([]byte(tc.data), []byte(tc.setData), tc.keys...)
			})
			if err != nil {
				t.Fatalf("Set(%q,%q,%v) returned unexpected error: %v", tc.data, tc.setData, tc.keys, err)
			}
			if val == nil {
				t.Fatalf("Set(%q,%q,%v) returned nil document", tc.data, tc.setData, tc.keys)
			}
		})
	}
}

// =============================================================================
// Delete — empty key component leaves the payload unchanged, no panic
// (SYS-REQ-034 / SYS-REQ-035)
// =============================================================================

// Verifies: SYS-REQ-034 [boundary]
// Verifies: SYS-REQ-035 [boundary]
// Delete with an empty-string key component cannot resolve a target; the
// parser must return the original byte payload unchanged and must not panic
// on the `keys[lk-1][0]` dereference.
func TestDeleteEmptyKeyPathComponent(t *testing.T) {
	cases := []struct {
		name string
		data string
		keys []string
	}{
		{name: "single empty key", data: `{"a":1,"b":2}`, keys: []string{""}},
		{name: "empty key trailing", data: `{"a":1}`, keys: []string{"a", ""}},
		{name: "empty key leading", data: `{"a":1}`, keys: []string{"", "a"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []byte
			runNoPanic(t, tc.name, func() {
				got = Delete([]byte(tc.data), tc.keys...)
			})
			want := []byte(tc.data)
			if string(got) != string(want) {
				t.Fatalf("Delete(%q,%v) = %q, want original payload %q (unchanged)",
					tc.data, tc.keys, string(got), tc.data)
			}
		})
	}
}
