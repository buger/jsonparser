package jsonparser

import (
	"bytes"
	"math/rand"
	"testing"
)

// =============================================================================
// Focused equivalence tests for the stringEnd fast path (PR #285-style
// "fast path for the provably-safe common case" pattern).
//
// These tests pin the contract that stringEnd's new SIMD fast path returns
// EXACTLY the same (endIndex, escaped) tuple the original per-byte loop
// would return across the full input domain:
//
//   1. no '"' and no '\'           -> (-1, false)
//   2. no '"' with '\' somewhere   -> (-1, true)
//   3. '"' before any '\'          -> (quoteIdx+1, false)
//   4. '\' before any '"'          -> slow path (validate against reference)
//   5. random fuzz against a reference implementation
// =============================================================================

// referenceStringEnd is the original byte-at-a-time implementation, kept here
// as an oracle. Do NOT modify — it is the semantic reference.
func referenceStringEnd(data []byte) (int, bool) {
	escaped := false
	for i, c := range data {
		if c == '"' {
			if !escaped {
				return i + 1, false
			} else {
				j := i - 1
				for {
					if j < 0 || data[j] != '\\' {
						return i + 1, true
					}
					j--
					if j < 0 || data[j] != '\\' {
						break
					}
					j--
				}
			}
		} else if c == '\\' {
			escaped = true
		}
	}
	return -1, escaped
}

// Verifies: SYS-REQ-045
func TestStringEnd_FastPathEquivalence_Cases(t *testing.T) {
	cases := [][]byte{
		{},                        // empty
		[]byte(`hello`),           // no '"' no '\'  -> (-1, false)
		[]byte(`he\llo`),          // '\' no '"'     -> (-1, true)
		[]byte(`hello"`),          // '"' no '\'     -> (6, false)
		[]byte(`a\"b"`),           // '\' then '"'   -> slow path, escaped
		[]byte(`a\\"b"`),          // '\\' then '"'  -> slow path, unescaped
		[]byte(`a\\\"b"`),         // '\\\' then '"' -> slow path, escaped
		[]byte(`"`),                // lone quote
		[]byte(`\`),                // lone backslash
		[]byte(`\"`),               // '\' then '"' (escaped quote, no terminator)
		[]byte(`""`),               // two unescaped quotes — should return at first
		[]byte(`a"b"c"`),          // first quote wins
		[]byte(`a\""b"`),          // escaped then unescaped
		[]byte(`a\\\"\\\\"`),      // complex escape mix ending unescaped
		[]byte(`\u00e9"`),         // unicode escape, common
	}
	for i, in := range cases {
		gotEnd, gotEsc := stringEnd(in)
		wantEnd, wantEsc := referenceStringEnd(in)
		if gotEnd != wantEnd || gotEsc != wantEsc {
			t.Errorf("case %d (%q): stringEnd=(%d,%v) want=(%d,%v)",
				i, in, gotEnd, gotEsc, wantEnd, wantEsc)
		}
	}
}

// Verifies: SYS-REQ-045
func TestStringEnd_FastPathEquivalence_Random(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	alphabets := [][]byte{
		[]byte(`abcXYZ 123`),
		[]byte(`abc\"\\`),
		[]byte(`abc"`),
	}
	const iterations = 20000
	maxLen := 32

	for i := 0; i < iterations; i++ {
		n := 1 + r.Intn(maxLen)
		buf := make([]byte, n)
		for j := range buf {
			alphabet := alphabets[r.Intn(len(alphabets))]
			buf[j] = alphabet[r.Intn(len(alphabet))]
		}
		gotEnd, gotEsc := stringEnd(buf)
		wantEnd, wantEsc := referenceStringEnd(buf)
		if gotEnd != wantEnd || gotEsc != wantEsc {
			t.Fatalf("iter %d input=%q: stringEnd=(%d,%v) want=(%d,%v)",
				i, buf, gotEnd, gotEsc, wantEnd, wantEsc)
		}
	}
}

// Directly exercise the fast-path branches with long inputs that would be
// expensive under per-byte scanning.
// Verifies: SYS-REQ-045
func TestStringEnd_FastPath_LongInputs(t *testing.T) {
	// 1 KiB of 'a' followed by closing quote — exercises IndexByte SIMD scan.
	long := bytes.Repeat([]byte{'a'}, 1024)
	longWithQuote := append(append([]byte{}, long...), '"')
	gotEnd, gotEsc := stringEnd(longWithQuote)
	if gotEnd != 1025 || gotEsc {
		t.Fatalf("long no-escape: stringEnd=(%d,%v) want=(1025,false)", gotEnd, gotEsc)
	}

	// Long with backslash far after the quote — still fast path.
	longLateEscape := append(append([]byte{}, long...), '"', '\\', 'n')
	gotEnd, gotEsc = stringEnd(longLateEscape)
	if gotEnd != 1025 || gotEsc {
		t.Fatalf("long late-escape: stringEnd=(%d,%v) want=(1025,false)", gotEnd, gotEsc)
	}

	// Long with backslash BEFORE the quote — slow path; result must still match.
	longWithEscape := append(append([]byte{}, long...), '\\', '"')
	gotEnd, gotEsc = stringEnd(longWithEscape)
	wantEnd, wantEsc := referenceStringEnd(longWithEscape)
	if gotEnd != wantEnd || gotEsc != wantEsc {
		t.Errorf("long early-escape: stringEnd=(%d,%v) want=(%d,%v)", gotEnd, gotEsc, wantEnd, wantEsc)
	}
}
