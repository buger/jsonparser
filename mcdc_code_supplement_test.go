package jsonparser

import (
	"testing"
)

// =============================================================================
// Code-level MC/DC supplement tests.
// =============================================================================
//
// These tests drive specific branch combinations reported as gaps by
// `proof mcdc report --view hotspots`. They target:
//   - lastToken / tokenStart / tokenEnd whitespace branches
//   - createInsertComponent / calcAllocateSpace (len(keys[i]) > 0)

// -----------------------------------------------------------------------------
// tokenStart / tokenEnd — direct unit tests to drive the short-circuit gaps.
// These helpers are reachable from the parser but the short-circuit operands
// only fire under very specific trailing/leading byte patterns, so we drive
// them directly here.
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-001 [mcdc]
func TestCodeMCDC_TokenStartDirect(t *testing.T) {
	// Each input ends with the targeted delimiter byte, so tokenStart's
	// for-loop returns that index with the corresponding c != <delim>
	// operand as the independent flipper.
	cases := map[byte]int{
		'\n': tokenStart([]byte("abc\n")), // 10
		'\r': tokenStart([]byte("abc\r")), // 13
		'\t': tokenStart([]byte("abc\t")), // 9
	}
	for delim, got := range cases {
		if got != 3 {
			t.Errorf("tokenStart with trailing %q returned %d, want 3", delim, got)
		}
	}
	// Baseline row: no delimiter present, falls through to return 0.
	if got := tokenStart([]byte("abc")); got != 0 {
		t.Errorf("tokenStart(abc) = %d, want 0", got)
	}
}

// Verifies: SYS-REQ-001 [mcdc]
func TestCodeMCDC_TokenEndDirect(t *testing.T) {
	// tokenEnd scans forward looking for a delimiter. Drive the c != 9
	// (TAB) operand as the independent flipper.
	if got := tokenEnd([]byte("abc\t")); got != 3 {
		t.Errorf("tokenEnd(abc\\t) = %d, want 3", got)
	}
}

// -----------------------------------------------------------------------------
// lastToken / tokenStart / tokenEnd — drive each whitespace branch.
// `lastToken` is reached via Set on a top-level key of a non-empty object that
// has trailing whitespace; `tokenStart`/`tokenEnd` are reached via Get on
// inputs whose first/last bytes are the targeted whitespace.
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-009 [mcdc]
func TestCodeMCDC_LastTokenNewline(t *testing.T) {
	// Set a top-level key on an object with trailing '\n' whitespace forces
	// lastToken to walk past '\n' (c == '\n' = T branch).
	if _, err := Set([]byte("{\"a\":1}\n"), []byte(`42`), "b"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
}

// Verifies: SYS-REQ-009 [mcdc]
func TestCodeMCDC_LastTokenCarriageReturn(t *testing.T) {
	if _, err := Set([]byte("{\"a\":1}\r"), []byte(`42`), "b"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
}

// Verifies: SYS-REQ-009 [mcdc]
func TestCodeMCDC_LastTokenTab(t *testing.T) {
	if _, err := Set([]byte("{\"a\":1}\t"), []byte(`42`), "b"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
}

// Verifies: SYS-REQ-009 [mcdc]
func TestCodeMCDC_LastTokenSpace(t *testing.T) {
	// Space is the already-covered T branch; keep it as the baseline row.
	if _, err := Set([]byte("{\"a\":1} "), []byte(`42`), "b"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
}

// Verifies: SYS-REQ-001 [mcdc]
func TestCodeMCDC_TokenStartNewline(t *testing.T) {
	// Leading '\n' before the value exercises the c != 10 short-circuit gap
	// in tokenStart (parser.go:173).
	if _, _, _, err := Get([]byte("\n{\"a\":1}"), "a"); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

// Verifies: SYS-REQ-001 [mcdc]
func TestCodeMCDC_TokenStartCarriageReturn(t *testing.T) {
	if _, _, _, err := Get([]byte("\r{\"a\":1}"), "a"); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

// Verifies: SYS-REQ-001 [mcdc]
func TestCodeMCDC_TokenStartTab(t *testing.T) {
	if _, _, _, err := Get([]byte("\t{\"a\":1}"), "a"); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

// Verifies: SYS-REQ-001 [mcdc]
func TestCodeMCDC_TokenEndTab(t *testing.T) {
	// A bare scalar terminated by TAB drives the c != 9 branch in tokenEnd
	// (parser.go:53).
	if _, _, _, err := Get([]byte("1\t")); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

// -----------------------------------------------------------------------------
// createInsertComponent / calcAllocateSpace — direct call to drive the
// `len(keys[i]) > 0` short-circuit gap with an empty intermediate key.
// -----------------------------------------------------------------------------

// Verifies: SYS-REQ-009 [mcdc]
func TestCodeMCDC_CreateInsertComponentEmptyKey(t *testing.T) {
	// Direct invocation with an empty-string intermediate key drives the
	// `len(keys[i]) > 0` operand to F.
	keys := []string{"a", "", "b"}
	got := createInsertComponent(keys, []byte(`42`), false, false)
	if len(got) == 0 {
		t.Fatal("expected non-empty result from createInsertComponent")
	}
}

// Verifies: SYS-REQ-009 [mcdc]
func TestCodeMCDC_CalcAllocateSpaceEmptyKey(t *testing.T) {
	keys := []string{"a", "", "b"}
	if got := calcAllocateSpace(keys, []byte(`42`), false, false); got <= 0 {
		t.Fatalf("expected positive allocation, got %d", got)
	}
}
