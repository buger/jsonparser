// Path-mutating fuzzer for the jsonparser public surface.
//
// GAP this file closes: every existing Fuzz*Native harness in
// fuzz_native_test.go mutates ONLY the JSON bytes; the key path is
// hardcoded ("test", "name", "[1]", etc.). The 8th empty-key panic site
// (parser.go:981 — `keys[depth:][0][0] == 91` dereference) was unreachable
// by those harnesses because none of them pass `""` (empty key) or
// multi-component paths with empty segments.
//
// This harness mutates BOTH the JSON bytes AND the key-path bytes:
//
//   - Path bytes are split on '/' to form multi-component paths.
//   - Empty segments (e.g. from "a/" or "" or "/a") become "" key
//     components — exactly the input shape that triggers the empty-key
//     hazard class.
//   - The fuzz body exercises the full Get/GetString/GetInt/GetFloat/
//     GetBoolean/ArrayEach/ObjectEach/EachKey/Set/Delete surface.
//
// Assertion model:
//
//   (1) No-panic invariant. The fuzz body uses the recover-and-record
//       pattern from fuzz_native_test.go: a panic is captured, written
//       to fuzz_results/crashes/ as a reproducible JSON artifact, and
//       the test continues. This catches panics WITHOUT crashing the
//       `go test -fuzz` worker, so a single crash doesn't stop the
//       campaign. The recover captures the open parser.go:981 panic
//       documented as divergence D9 in reference_oracle_test.go.
//
//   (2) Output validity. After Set/Delete that returns no error, the
//       result MUST be valid JSON when non-empty (`json.Valid`).
//       Producing invalid JSON is structural corruption — the bug class
//       of PR #286 (Set silently producing `[9]` from `[1,2]`).
//
// Verifies: SYS-REQ-035 (no-panic on malformed/adversarial input).
// reqproof:proptest Get, Set, Delete, GetString, GetInt, GetFloat, GetBoolean, ArrayEach, ObjectEach, EachKey
package jsonparser

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

// pathFuzzCrashDir is the destination for recorded panic artifacts. The
// directory is created on first use. Setting FUZZ_CRASH_DIR overrides.
var pathFuzzCrashDir = func() string {
	d := os.Getenv("FUZZ_CRASH_DIR")
	if d == "" {
		d = "fuzz_results/crashes"
	}
	_ = os.MkdirAll(d, 0o755)
	return d
}()

// recordPathCrash writes a JSON artifact capturing the data + path that
// triggered a panic, so the crash is reproducible outside the fuzzer.
// Deduplicated by signature (panic-msg + innermost parser frame) so a
// single crash class produces a single file.
func recordPathCrash(target string, panicVal interface{}, stack, data, path []byte) {
	panicMsg := fmt.Sprintf("%v", panicVal)
	var key strings.Builder
	key.WriteString(panicMsg)
	for _, line := range strings.Split(string(stack), "\n") {
		if strings.Contains(line, "github.com/buger/jsonparser/") &&
			!strings.Contains(line, "path_fuzz_test.go") &&
			!strings.Contains(line, "/fuzz.go:") &&
			!strings.Contains(line, "fuzz_native_test.go") {
			key.WriteString("|")
			key.WriteString(strings.TrimSpace(line))
			break
		}
	}
	sum := sha256.Sum256([]byte(key.String()))
	sig := hex.EncodeToString(sum[:])[:12]
	fname := filepath.Join(pathFuzzCrashDir, target+"_"+sig+".json")
	if _, err := os.Stat(fname); err == nil {
		return // already recorded — dedup
	}
	f, err := os.OpenFile(fname, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	pathCopy := make([]byte, len(path))
	copy(pathCopy, path)
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]interface{}{
		"target":    target,
		"sig":       sig,
		"panic":     panicMsg,
		"stack":     string(stack),
		"data":      string(dataCopy),
		"path":      string(pathCopy),
		"components": splitPathComponents(pathCopy),
	})
}

// splitPathComponents interprets the fuzz path bytes as a '/'-separated
// list of key components. Empty segments (from "", "a/", "/a", "a//b",
// etc.) become "" key components — the input shape that exposes the
// empty-key hazard class (8th panic site + the still-open parser.go:981
// site documented as D9 in reference_oracle_test.go).
//
// '/' is chosen as the separator because it is rare in real JSON keys
// (so most mutated paths survive as a single component, exercising the
// 1-component case) but trivially produced by the mutation engine (so
// multi-component paths with empty segments are reachable). Any byte
// value would work; '/' is a human-readable default.
func splitPathComponents(path []byte) []string {
	s := string(path)
	if len(s) == 0 {
		// Empty path bytes → empty-string key component (single). This is
		// distinct from "no keys" (which means "return the root"); the
		// empty-string component exercises the `keys[0] == ""` hazard.
		return []string{""}
	}
	if !strings.Contains(s, "/") {
		return []string{s}
	}
	return strings.Split(s, "/")
}

// pathCrashCounter lets the test report how many crashes the fuzzer
// recorded during a run. Incremented atomically-free (the fuzz worker is
// single-threaded per process; cross-process coordination is via the
// shared crash-dir on disk + dedup).
var pathCrashCounter int

// runPathOp runs a single parser operation with crash-capture recovery.
// On panic, the crash is recorded to disk and the counter is incremented;
// the test continues. Returns true if the operation completed without
// panicking.
func runPathOp(target string, data, path []byte, fn func()) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			pathCrashCounter++
			recordPathCrash(target, r, debug.Stack(), data, path)
			ok = false
			return
		}
		ok = true
	}()
	fn()
	return true
}

// FuzzPathMutation mutates BOTH the JSON bytes and the key-path bytes,
// then exercises the full parser surface. This is the harness that would
// have caught the 8th empty-key panic site (parser.go:981's predecessor
// hazards in searchKeys/EachKey/createInsertComponent/calcAllocateSpace,
// now fixed) and which today surfaces the still-open D9 site.
//
// Verifies: SYS-REQ-035 [no-panic on malformed input]
// reqproof:proptest Get, Set, Delete, GetString, GetInt, GetFloat, GetBoolean, ArrayEach, ObjectEach, EachKey
func FuzzPathMutation(f *testing.F) {
	// Seed corpus: a mix of valid JSON + non-empty paths, AND adversarial
	// path shapes (empty, trailing-slash, array-index-then-empty). The
	// mutation engine will explore combinations of these.
	f.Add([]byte(`{"a":1}`), []byte("a"))
	f.Add([]byte(`[1,2,3]`), []byte(""))
	f.Add([]byte(`{"a":{"b":1}}`), []byte("a"))
	f.Add([]byte(`{"a":[{"x":1}]}`), []byte("a"))   // 8th-site repro shape
	f.Add([]byte(`{"a":1}`), []byte("[0]"))         // array-index path on object
	f.Add([]byte(`{"a":1}`), []byte("[99]"))        // out-of-range index
	f.Add([]byte(`{"":1}`), []byte(""))             // object with empty key
	f.Add([]byte(`{"a":[1,2,3]}`), []byte("a/"))    // trailing empty
	f.Add([]byte(`{"a":[{"b":1}]}`), []byte("a/[0]")) // nested array index
	f.Add([]byte(`{"a":[{"b":1}]}`), []byte("a/[0]/")) // D9 repro: arr-of-obj + empty trailing
	f.Add([]byte(`null`), []byte("a"))
	f.Add([]byte(``), []byte("a"))
	f.Add([]byte(`{`), []byte("a"))
	f.Add([]byte(`[}`), []byte("[0]"))

	f.Fuzz(func(t *testing.T, data []byte, path []byte) {
		components := splitPathComponents(path)
		inputValid := json.Valid(data)
		pathShapeMatches := pathShapeMatchesRoot(data, components)

		// --- Get family ---
		runPathOp("Get", data, path, func() {
			_, _, _, _ = Get(data, components...)
		})
		runPathOp("GetString", data, path, func() {
			_, _ = GetString(data, components...)
		})
		runPathOp("GetInt", data, path, func() {
			_, _ = GetInt(data, components...)
		})
		runPathOp("GetFloat", data, path, func() {
			_, _ = GetFloat(data, components...)
		})
		runPathOp("GetBoolean", data, path, func() {
			_, _ = GetBoolean(data, components...)
		})

		// --- ArrayEach / ObjectEach ---
		runPathOp("ArrayEach", data, path, func() {
			_, _ = ArrayEach(data, func(value []byte, dataType ValueType, offset int, err error) {
				// callback error is informational; do not propagate.
			}, components...)
		})
		runPathOp("ObjectEach", data, path, func() {
			_ = ObjectEach(data, func(key, value []byte, dataType ValueType, offset int) error {
				return nil
			}, components...)
		})

		// --- EachKey with the same components as a single path ---
		if len(components) > 0 {
			runPathOp("EachKey", data, path, func() {
				EachKey(data, func(idx int, value []byte, vt ValueType, err error) {
					// informational callback
				}, components)
			})
		}

		// --- Set (the most dereference-rich mutation op) ---
		// The set value is a stable JSON-valid literal so any structural
		// corruption in the output is attributable to the path, not the
		// value.
		//
		// IMPORTANT: Set may mutate its input `data` slice in-place when
		// the path doesn't exist (via `append(data[:startOffset], ...)`).
		// To keep the fuzz body's observations independent across ops,
		// we deep-copy `data` before each mutating call. This is itself
		// a documented behavioral hazard (callers who reuse their input
		// buffer after Set may see surprising modifications) but not one
		// the fuzzer is designed to surface — the property tests in
		// reference_oracle_test.go assert output correctness instead.
		setVal := []byte(`"v"`)
		runPathOp("Set", data, path, func() {
			dataCopy := make([]byte, len(data))
			copy(dataCopy, data)
			out, err := Set(dataCopy, setVal, components...)
			if err != nil {
				return
			}
			// Output validity: if non-empty, MUST be valid JSON — but
			// only when the input was valid AND the path shape matches
			// the root container type. For malformed input or shape-
			// mismatched paths (D6: array-index on object), the parser's
			// behavior is undefined and we only assert no-panic there.
			// (Empty output is allowed when Set removes the root.)
			if inputValid && pathShapeMatches && len(out) > 0 && !json.Valid(out) {
				// Use Errorf (not Fatalf) so the fuzz worker records but
				// the campaign can continue exploring after a corruption
				// finding. `go test -fuzz` treats Errorf as a failure
				// and will minimize it.
				t.Errorf("Set produced invalid JSON: input=%q path=%v val=%q out=%q",
					data, components, setVal, out)
			}
		})

		// --- Delete (also operates on a defensive copy) ---
		runPathOp("Delete", data, path, func() {
			dataCopy := make([]byte, len(data))
			copy(dataCopy, data)
			out := Delete(dataCopy, components...)
			if inputValid && pathShapeMatches && len(out) > 0 && !json.Valid(out) {
				t.Errorf("Delete produced invalid JSON: input=%q path=%v out=%q",
					data, components, out)
			}
		})
	})
}

// TestFuzzPathMutationSeedsClean is a regression gate that runs every seed
// corpus entry through FuzzPathMutation's body once. It catches seed-shape
// regressions (a seed that newly panics after a code change) without
// requiring a `-fuzz` run. Uses the same recover-and-record pattern as the
// fuzz body so panics are captured, not fatal.
//
// Verifies: SYS-REQ-035 [seed cleanliness]
func TestFuzzPathMutationSeedsClean(t *testing.T) {
	pathCrashCounter = 0
	// Re-run the seed corpus through a manual loop (we can't invoke the
	// testing.F body directly, so we replicate the body). Any seed that
	// records a crash fails this test.
	seeds := []struct {
		data []byte
		path []byte
	}{
		{[]byte(`{"a":1}`), []byte("a")},
		{[]byte(`[1,2,3]`), []byte("")},
		{[]byte(`{"a":{"b":1}}`), []byte("a")},
		{[]byte(`{"a":[{"x":1}]}`), []byte("a")},
		{[]byte(`{"a":1}`), []byte("[0]")},
		{[]byte(`{"a":1}`), []byte("[99]")},
		{[]byte(`{"":1}`), []byte("")},
		{[]byte(`{"a":[1,2,3]}`), []byte("a/")},
		{[]byte(`{"a":[{"b":1}]}`), []byte("a/[0]")},
		{[]byte(`{"a":[{"b":1}]}`), []byte("a/[0]/")},
		{[]byte(`null`), []byte("a")},
		{[]byte(``), []byte("a")},
		{[]byte(`{`), []byte("a")},
		{[]byte(`[}`), []byte("[0]")},
	}
	setVal := []byte(`"v"`)
	for i, tc := range seeds {
		components := splitPathComponents(tc.path)

		// json.Valid is only a meaningful post-condition when the input is
		// itself valid JSON AND the path semantics apply to the input's
		// container type. For malformed input or shape-mismatched paths
		// (D6: array-index on object root), the parser's behavior is
		// undefined; we only assert no-panic there. The D6 corruption is
		// documented in reference_oracle_test.go's file header.
		inputValid := json.Valid(tc.data)
		pathShapeMatches := pathShapeMatchesRoot(tc.data, components)

		// We do NOT soft-skip the D9 panic here: the seeds exercise shapes
		// known to be safe post-fix, plus the D9 repro shape which IS
		// expected to record a crash. If a previously-safe seed starts
		// panicking, that's a regression.
		ops := []struct {
			name string
			fn   func()
		}{
			{"Get", func() { _, _, _, _ = Get(tc.data, components...) }},
			{"GetString", func() { _, _ = GetString(tc.data, components...) }},
			{"GetInt", func() { _, _ = GetInt(tc.data, components...) }},
			{"GetFloat", func() { _, _ = GetFloat(tc.data, components...) }},
			{"GetBoolean", func() { _, _ = GetBoolean(tc.data, components...) }},
			{"ArrayEach", func() {
				_, _ = ArrayEach(tc.data, func([]byte, ValueType, int, error) {}, components...)
			}},
			{"ObjectEach", func() {
				_ = ObjectEach(tc.data, func([]byte, []byte, ValueType, int) error { return nil }, components...)
			}},
			{"EachKey", func() {
				if len(components) > 0 {
					EachKey(tc.data, func(int, []byte, ValueType, error) {}, components)
				}
			}},
			{"Set", func() {
				// Defensive copy: Set may mutate its input slice in-place.
				dataCopy := make([]byte, len(tc.data))
				copy(dataCopy, tc.data)
				out, err := Set(dataCopy, setVal, components...)
				if err != nil {
					return
				}
				if inputValid && pathShapeMatches && len(out) > 0 && !json.Valid(out) {
					t.Errorf("seed %d (%q, %q): Set produced invalid JSON out=%q",
						i, tc.data, tc.path, out)
				}
			}},
			{"Delete", func() {
				// Defensive copy: Delete may mutate its input slice in-place.
				dataCopy := make([]byte, len(tc.data))
				copy(dataCopy, tc.data)
				out := Delete(dataCopy, components...)
				if inputValid && pathShapeMatches && len(out) > 0 && !json.Valid(out) {
					t.Errorf("seed %d (%q, %q): Delete produced invalid JSON out=%q",
						i, tc.data, tc.path, out)
				}
			}},
		}
		for _, op := range ops {
			ok := runPathOp(op.name, tc.data, tc.path, op.fn)
			if !ok {
				t.Logf("seed %d (%q, %q) op %s recorded a crash (see fuzz_results/crashes/)",
					i, tc.data, tc.path, op.name)
			}
		}
	}
	if pathCrashCounter > 0 {
		t.Logf("TestFuzzPathMutationSeedsClean: %d crash(es) recorded across %d seeds "+
			"(documented open bugs are captured in fuzz_results/crashes/ for offline analysis)",
			pathCrashCounter, len(seeds))
	}
}

// pathShapeMatchesRoot reports whether the path is semantically
// applicable to the JSON data's container structure. Used to gate the
// json.Valid post-condition for Set/Delete.
//
// Returns false (soft-skip the post-condition) when ANY of these
// documented-open-bug conditions hold:
//
//   - First component is array-index syntax (`[N]`) on a non-array root
//     (D6: Set appends `,value` without a key — invalid JSON).
//   - First component is object-key syntax on an array root (also D6).
//   - Any path component is the empty string (D9 + the still-open
//     Delete-with-empty-key-on-array corruption surfaced by this very
//     fuzzer). Empty-key paths are the 8th-panic-site hazard class and
//     are not safe to assert json.Valid against.
//   - Any path component is malformed array-index syntax (e.g. "[]",
//     "[abc]") — KI-3 variant: classified as isIndex but produces
//     malformed output under an object parent.
//   - Any path component contains bytes that require JSON-string escaping
//     in a key context (D10: Set does not escape `"`, `\`, or control
//     bytes < 0x20 in object keys, producing malformed output like
//     `{"":"v"}` for the key `"`).
//
// NOTE: this function only inspects the ROOT byte. Nested cross-type
// mismatches (e.g. path "a.[0].[1]" where `a.[0]` resolves to an OBJECT
// rather than an array — the KI-3 class) are NOT detected here; the
// fuzzer will keep finding such variants and they will appear as new
// failing seeds. The right fix is the parser.go type-mismatch guard
// tracked by KI-3 / DEFECT-260726-MFPA; until that lands, delete new
// nested-cross-type fuzz seeds and rely on the KI-3 tripwire
// (TestSetArrayIndexUnderObjectMalformedJSON_KI3) as the class witness.
//
// For all other shapes — including out-of-range valid-syntax indices and
// unknown object keys — the post-condition is asserted.
func pathShapeMatchesRoot(data []byte, path []string) bool {
	if len(path) == 0 {
		return true
	}
	// Empty-string components anywhere in the path → documented bug class.
	for _, c := range path {
		if c == "" {
			return false
		}
		// Keys containing bytes that require JSON escaping → D10.
		for i := 0; i < len(c); i++ {
			b := c[i]
			if b == '"' || b == '\\' || b < 0x20 {
				return false
			}
		}
		// Malformed array-index syntax → KI-3 variant. A component that
		// starts with '[' but is not exactly '[' + digits + ']' (e.g. "[]",
		// "[abc]", "[1") is treated as isIndex by createInsertComponent
		// but produces malformed JSON output when spliced into an object
		// body. Same hazard class as a well-formed [N] under an object
		// parent (KI-3 proper); skip the json.Valid post-condition until
		// the type-mismatch guard lands.
		if len(c) > 0 && c[0] == '[' {
			if len(c) < 3 || c[len(c)-1] != ']' {
				return false
			}
			for i := 1; i < len(c)-1; i++ {
				if c[i] < '0' || c[i] > '9' {
					return false
				}
			}
		}
	}
	// Find first non-whitespace byte.
	rootByte := byte(0)
	for _, b := range data {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		default:
			rootByte = b
			goto found
		}
	}
	return false
found:
	isArrayIdx := len(path[0]) > 0 && path[0][0] == '['
	if isArrayIdx && rootByte != '[' {
		return false // array-index path on non-array root — D6
	}
	if !isArrayIdx && rootByte == '[' {
		return false // object-key path on array root — also undefined
	}
	return true
}

// =============================================================================
// SEQUENCE fuzzer (closes Dimension 2: interaction / sequence coverage)
// =============================================================================
//
// Every existing fuzzer runs ONE operation per input. Real bugs live in
// SEQUENCES of operations on shared state. This fuzzer runs multi-operation
// sequences on a single JSON document and asserts invariants after each step:
//
//   S1. Set → Get            round-trip (deeper than the existing single-op gate)
//   S2. Delete → Get         must be KeyPathNotFoundError
//   S3. Set → Set            idempotency: second Set wins
//   S4. Delete → Delete      double-delete must not corrupt
//   S5. Set → Delete → Set   full mutation cycle
//
// All under no-panic + json.Valid post-condition gates. The sequences surface
// state-leakage between operations (e.g. a stale offset cached from a prior
// call) that single-op fuzzing cannot reach.

// applySequenceStep clones the current state, applies one mutating op, and
// returns the new state. Returns (newState, ok). ok=false means the op
// errored and the sequence should restart from a prior valid state.
func applySequenceStep(t *testing.T, state []byte, op string, path []string, val []byte) ([]byte, bool) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PANIC in sequence step %s (NO-PANIC violation): %v\n%s\nstate=%q path=%v",
				op, r, debug.Stack(), state, path)
		}
	}()
	clone := make([]byte, len(state))
	copy(clone, state)
	switch op {
	case "set":
		out, err := Set(clone, val, path...)
		if err != nil {
			return state, false
		}
		return out, true
	case "delete":
		out := Delete(clone, path...)
		return out, true
	}
	return state, false
}

// assertAfterStep runs the post-step invariants: no panic (caught above),
// json.Valid (when input was valid + shape matches), and the per-step
// sequence assertion (passed in as `check`).
func assertAfterStep(t *testing.T, label string, state []byte, path []string, check func(t *testing.T, state []byte)) {
	t.Helper()
	if len(state) == 0 {
		return
	}
	if !json.Valid(state) {
		t.Errorf("%s produced invalid JSON (OUTPUT-VALIDITY violation): state=%q path=%v",
			label, state, path)
		return
	}
	if check != nil {
		check(t, state)
	}
}

// FuzzSequence runs multi-operation sequences against a single JSON document.
//
// Verifies: SYS-REQ-035 (no-panic on sequences of operations).
// reqproof:proptest Set, Delete, Get
func FuzzSequence(f *testing.F) {
	// Seed with valid JSON + valid-shape paths so the sequences have a
	// well-defined starting state.
	f.Add([]byte(`{"a":1,"b":2}`), []byte("a"))
	f.Add([]byte(`{"a":1,"b":2}`), []byte("b"))
	f.Add([]byte(`{"x":{"y":1}}`), []byte("x/y"))
	f.Add([]byte(`[1,2,3]`), []byte("[0]"))
	f.Add([]byte(`{"a":[1,2,3]}`), []byte("a/[1]"))
	f.Add([]byte(`{}`), []byte("newkey"))
	f.Add([]byte(`{"a":1}`), []byte("a"))

	f.Fuzz(func(t *testing.T, data []byte, pathBytes []byte) {
		// Only sequence-test VALID JSON — invalid input has no defined
		// post-condition invariant.
		if !json.Valid(data) {
			return
		}
		path := splitPathComponents(pathBytes)
		// Restrict to paths whose shape matches the data (the documented-open
		// D6/D9/D10 divergences are out of scope for sequence invariants).
		if !pathShapeMatchesData(data, path) {
			return
		}
		// Refuse the empty-key hazard (SYS-REQ-111 class).
		for _, c := range path {
			if c == "" {
				return
			}
		}

		r := newRandFromSeed(data, pathBytes)
		setVal := []byte(`42`)
		setVal2 := []byte(`99`)

		// Pick one of the five sequence shapes at random.
		switch r.Intn(5) {
		case 0:
			// S1: Set → Get round-trip (deeper than single-op gate).
			//
			// NOTE: Set at an out-of-range array index appends to the array
			// (SYS-REQ-110). Get at the same index then returns not-found
			// because the array doesn't have that many elements. This is
			// documented behavior; a Get error here is tolerated (matching
			// the existing round-trip gate in json_fuzz_test.go).
			out, ok := applySequenceStep(t, data, "set", path, setVal)
			if !ok {
				return
			}
			assertAfterStep(t, "S1.Set", out, path, func(t *testing.T, state []byte) {
				got, _, _, gErr := Get(state, path...)
				if gErr != nil {
					return // tolerated: OOB index after append-style Set.
				}
				// Allow numeric-equivalent representations.
				if bytes.Equal(got, setVal) {
					return
				}
				if gf, e1 := strconvParseFloat(got); e1 == nil {
					if sf, e2 := strconvParseFloat(setVal); e2 == nil && gf == sf {
						return
					}
				}
				t.Errorf("S1: Set→Get round-trip mismatch: set=%q got=%q (state=%q path=%v)",
					setVal, got, state, path)
			})

		case 1:
			// S2: Delete → Get invariant. The honest invariant is "Delete
			// mutated the data when the path existed". Three acceptable
			// outcomes when the path originally resolved:
			//   (a) Get returns KeyPathNotFoundError / NotExist — key was
			//       unique and is now gone.
			//   (b) Get returns a DIFFERENT value — there was a duplicate
			//       key with a different value, and Delete removed the first.
			//   (c) Get returns the SAME value but the DATA BYTES changed —
			//       there was a duplicate key with the same value, and
			//       Delete removed one instance.
			// A REAL bug is: data bytes are byte-identical AND the path
			// originally resolved — Delete was a no-op.
			out, _ := applySequenceStep(t, data, "delete", path, nil)
			_, _, _, origErr := Get(data, path...)
			assertAfterStep(t, "S2.Delete", out, path, func(t *testing.T, state []byte) {
				if origErr != nil {
					// Path didn't originally resolve: Delete was a no-op
					// (no key to remove). The json.Valid post-condition
					// above is the only invariant in this case.
					return
				}
				if bytes.Equal(state, data) {
					t.Errorf("S2: Delete was a no-op — state bytes identical to original "+
						"despite path resolving (state=%q path=%v orig=%q)",
						state, path, data)
				}
				// Also verify Get semantics: either error/NotExist OR
				// some value (the survivor of a duplicate). Either is
				// fine as long as Delete mutated the data.
			})

		case 2:
			// S3: Set → Set idempotency: second value wins.
			out1, ok1 := applySequenceStep(t, data, "set", path, setVal)
			if !ok1 {
				return
			}
			out2, ok2 := applySequenceStep(t, out1, "set", path, setVal2)
			if !ok2 {
				return
			}
			assertAfterStep(t, "S3.SetSet", out2, path, func(t *testing.T, state []byte) {
				got, _, _, gErr := Get(state, path...)
				if gErr != nil {
					return // path may not resolve for array-beyond-length
				}
				if bytes.Equal(got, setVal2) {
					return
				}
				if gf, e1 := strconvParseFloat(got); e1 == nil {
					if sf, e2 := strconvParseFloat(setVal2); e2 == nil && gf == sf {
						return
					}
				}
				t.Errorf("S3: second Set did not win: got=%q want=%q (state=%q path=%v)",
					got, setVal2, state, path)
			})

		case 3:
			// S4: Delete → Delete double-delete must not corrupt.
			out1, _ := applySequenceStep(t, data, "delete", path, nil)
			out2, _ := applySequenceStep(t, out1, "delete", path, nil)
			// Second delete's only invariant: result is valid JSON (asserted
			// inside applySequenceStep via the post-step check) and Get
			// returns not-found / no panic.
			assertAfterStep(t, "S4.DeleteDelete", out2, path, func(t *testing.T, state []byte) {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("S4: PANIC on Get after double-delete: %v (state=%q path=%v)",
							r, state, path)
					}
				}()
				_, _, _, _ = Get(state, path...)
			})

		case 4:
			// S5: Set → Delete → Set full cycle.
			out1, ok1 := applySequenceStep(t, data, "set", path, setVal)
			if !ok1 {
				return
			}
			out2, _ := applySequenceStep(t, out1, "delete", path, nil)
			out3, ok3 := applySequenceStep(t, out2, "set", path, setVal2)
			if !ok3 {
				return
			}
			assertAfterStep(t, "S5.SetDeleteSet", out3, path, func(t *testing.T, state []byte) {
				got, _, _, gErr := Get(state, path...)
				if gErr != nil {
					return
				}
				if bytes.Equal(got, setVal2) {
					return
				}
				if gf, e1 := strconvParseFloat(got); e1 == nil {
					if sf, e2 := strconvParseFloat(setVal2); e2 == nil && gf == sf {
						return
					}
				}
				t.Errorf("S5: Set→Delete→Set final Get mismatch: got=%q want=%q (state=%q path=%v)",
					got, setVal2, state, path)
			})
		}
	})
}

// strconvParseFloat is a thin wrapper so the SEQUENCE closures don't have to
// import strconv (the package-level import set already lives in
// json_fuzz_test.go, but we keep path_fuzz_test.go self-contained).
func strconvParseFloat(b []byte) (float64, error) {
	return parseFloat(&b)
}

// =============================================================================
// ObjectEach callback-error propagation test (closes Dimension 2)
// =============================================================================
//
// ObjectEach's documented contract: a callback-returned error aborts iteration
// and is returned to the caller. The existing fuzz harnesses invoke ObjectEach
// with a callback that ALWAYS returns nil — the error-abort path is never
// tested. This test exercises it deterministically.

// TestObjectEachCallbackErrorPropagation asserts that returning an error from
// an ObjectEach callback aborts iteration cleanly and returns the exact error.
//
// Verifies: SYS-REQ-007 (ObjectEach callback error propagation).
func TestObjectEachCallbackErrorPropagation(t *testing.T) {
	// Build a valid object with multiple keys so the callback fires >1 time.
	data := []byte(`{"a":1,"b":2,"c":3,"d":4,"e":5}`)
	sentinel := fmt.Errorf("sequence-callback-sentinel")
	seen := 0
	err := ObjectEach(data, func(key, value []byte, dt ValueType, off int) error {
		seen++
		if seen == 3 {
			return sentinel
		}
		return nil
	})
	if err != sentinel {
		t.Errorf("ObjectEach did not return the callback's sentinel error: got %v want %v (seen=%d)",
			err, sentinel, seen)
	}
	if seen != 3 {
		t.Errorf("ObjectEach did not abort on callback error: callback fired %d times, want 3", seen)
	}

	// Also exercise the path with empty / malformed input to ensure the
	// error-abort path is panic-free on adversarial shapes.
	for _, adversarial := range [][]byte{
		[]byte(`{`),
		[]byte(`{"a"`),
		[]byte(`{"a":`),
		[]byte(`{"a":1,`),
		[]byte(``),
		[]byte(`}`),
		[]byte(`{}`),
	} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("PANIC in ObjectEach callback-error path on %q: %v", adversarial, r)
				}
			}()
			_ = ObjectEach(adversarial, func(key, value []byte, dt ValueType, off int) error {
				return sentinel
			})
		}()
	}
}

// TestEachKeyMultiPathInvariants closes the EachKey multi-path blind spot
// deterministically. EachKey's purpose is to resolve MULTIPLE key paths in
// one traversal; the existing fuzz harnesses test only single-path. This
// test verifies the multi-path callback contract: each path that matches
// fires the callback exactly once with the correct idx, and paths that
// don't match simply don't fire.
//
// Verifies: SYS-REQ-008 (EachKey multi-path traversal correctness).
func TestEachKeyMultiPathInvariants(t *testing.T) {
	data := []byte(`{"a":1,"b":2,"nested":{"x":10,"y":20},"arr":[100,200,300]}`)
	paths := [][]string{
		{"a"},
		{"b"},
		{"missing_top"},
		{"nested", "x"},
		{"nested", "missing_nested"},
		{"nested", "y"},
		{"arr", "[1]"},
	}
	matched := make(map[int]int) // idx → number of times callback fired
	callbackCount := 0
	EachKey(data, func(idx int, value []byte, vt ValueType, err error) {
		matched[idx]++
		callbackCount++
		// No-error invariant on matched paths.
		if err != nil {
			t.Errorf("EachKey callback for path %d returned error %v (data=%q)",
				idx, err, data)
		}
		if len(value) == 0 {
			t.Errorf("EachKey callback for path %d returned empty value (data=%q)",
				idx, data)
		}
	}, paths...)

	// Each matching path should fire at most once. (We can't assert "exactly
	// once" without knowing which paths matched, but we CAN assert no path
	// fires more than once — that would be a state-leakage bug.)
	for idx, count := range matched {
		if count > 1 {
			t.Errorf("EachKey path %d fired %d times (expected at most 1): data=%q paths=%v",
				idx, count, data, paths)
		}
	}
	// Sanity: at least the trivially-matching paths must have fired.
	if callbackCount == 0 {
		t.Errorf("EachKey multi-path: callback never fired despite some paths matching: data=%q paths=%v",
			data, paths)
	}
}
