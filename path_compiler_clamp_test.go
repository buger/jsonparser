package jsonparser

import (
	"strings"
	"testing"
)

// TestParsePathHintClampPreservesResults checks that clamping the
// pre-allocation hint does not change which paths are accepted or what they
// parse to, including paths with far more components than the clamp.
// Literals are used deliberately so this test compiles and passes both with
// and without the clamp.
func TestParsePathHintClampPreservesResults(t *testing.T) {
	for _, n := range []int{1, 2, 511, 512, 513, 2600} {
		keys := make([]string, n)
		for i := range keys {
			keys[i] = "k"
		}
		path := strings.Join(keys, ".")

		got, err := ParsePath(path)
		if err != nil {
			t.Fatalf("n=%d: unexpected error: %v", n, err)
		}
		if len(got) != n {
			t.Fatalf("n=%d: got %d components, want %d", n, len(got), n)
		}
		for i, c := range got {
			if c != "k" {
				t.Fatalf("n=%d idx=%d: got %q, want \"k\"", n, i, c)
			}
		}
	}

	// Bracket notation past the clamp.
	var sb strings.Builder
	sb.WriteString("a")
	for i := 0; i < 1500; i++ {
		sb.WriteString("[0]")
	}
	if got, err := ParsePath(sb.String()); err != nil {
		t.Fatalf("bracket path: %v", err)
	} else if len(got) != 1501 {
		t.Fatalf("bracket path: got %d components, want 1501", len(got))
	}

	// Malformed paths must still be rejected.
	for _, bad := range []string{"", ".", "..", ".a", "a..b", "a[", "a]"} {
		if _, err := ParsePath(bad); err == nil {
			t.Errorf("ParsePath(%q): expected an error", bad)
		}
	}
}

// TestParsePathHintDoesNotRegressDeepPaths pins the reason the bound is
// proportional to the path length instead of a constant.
//
// A constant ceiling silently penalises valid deep paths: with a 512 ceiling a
// well-formed 2000-component path allocated 113177 B across 5 allocations,
// against 32781 B in 1 allocation unclamped, because append has to regrow. The
// correctness test above cannot see that -- it passed with the constant too --
// so the property needs an allocation assertion of its own.
//
// One allocation is the whole point: the hint has to be large enough that
// append never regrows for a path that really does have this many components.
func TestParsePathHintDoesNotRegressDeepPaths(t *testing.T) {
	const n = 2000
	keys := make([]string, n)
	for i := range keys {
		keys[i] = "k"
	}
	path := strings.Join(keys, ".")

	var got []string
	allocs := testing.AllocsPerRun(50, func() {
		got, _ = ParsePath(path)
	})
	if len(got) != n {
		t.Fatalf("got %d components, want %d", len(got), n)
	}
	if allocs > 1 {
		t.Errorf("ParsePath on a valid %d-component path used %.0f allocations, want 1; "+
			"the pre-allocation hint is under-reserving and append is regrowing", n, allocs)
	}
}
