package jsonparser

import (
	"bytes"
	"encoding/json"
	"testing"
	"unicode/utf8"
)

// Verifies: SYS-REQ-014 [boundary]
// MCDC SYS-REQ-014: N/A
func TestH2I(t *testing.T) {
	hexChars := []byte{'0', '9', 'A', 'F', 'a', 'f', 'x', '\000'}
	hexValues := []int{0, 9, 10, 15, 10, 15, -1, -1}

	for i, c := range hexChars {
		if v := h2I(c); v != hexValues[i] {
			t.Errorf("h2I('%c') returned wrong value (obtained %d, expected %d)", c, v, hexValues[i])
		}
	}
}

type escapedUnicodeRuneTest struct {
	in    string
	isErr bool
	out   rune
	len   int
}

var commonUnicodeEscapeTests = []escapedUnicodeRuneTest{
	{in: `\u0041`, out: 'A', len: 6},
	{in: `\u0000`, out: 0, len: 6},
	{in: `\u00b0`, out: '°', len: 6},
	{in: `\u00B0`, out: '°', len: 6},

	{in: `\x1234`, out: 0x1234, len: 6}, // These functions do not check the \u prefix

	{in: ``, isErr: true},
	{in: `\`, isErr: true},
	{in: `\u`, isErr: true},
	{in: `\u1`, isErr: true},
	{in: `\u11`, isErr: true},
	{in: `\u111`, isErr: true},
	{in: `\u123X`, isErr: true},
}

var singleUnicodeEscapeTests = append([]escapedUnicodeRuneTest{
	{in: `\uD83D`, out: 0xD83D, len: 6},
	{in: `\uDE03`, out: 0xDE03, len: 6},
	{in: `\uFFFF`, out: 0xFFFF, len: 6},
	{in: `\uFF11`, out: '１', len: 6},
}, commonUnicodeEscapeTests...)

var multiUnicodeEscapeTests = append([]escapedUnicodeRuneTest{
	// Lone surrogates: malformed per RFC 8259/WHATWG. Match encoding/json by
	// substituting U+FFFD (utf8.RuneError) and consuming only the 6 bytes of
	// the lone-surrogate escape. See DEFECT-260727-SNGT.
	{in: `\uD83D`, out: utf8.RuneError, len: 6}, // lone high surrogate
	{in: `\uDE03`, out: utf8.RuneError, len: 6}, // lone low surrogate
	{in: `\uFFFF`, out: '\uFFFF', len: 6},
	{in: `\uFF11`, out: '１', len: 6},

	{in: `\uD83D\uDE03`, out: '\U0001F603', len: 12},
	{in: `\uD800\uDC00`, out: '\U00010000', len: 12},

	{in: `\uD800\`, out: utf8.RuneError, len: 6},      // high surrogate not followed by "\u" → lone high
	{in: `\uD800\u`, isErr: true},                     // second escape truncated → malformed
	{in: `\uD800\uD`, isErr: true},                    // second escape truncated → malformed
	{in: `\uD800\uDC`, isErr: true},                   // second escape truncated → malformed
	{in: `\uD800\uDC0`, isErr: true},                  // second escape truncated → malformed
	{in: `\uD800\uDBFF`, out: utf8.RuneError, len: 6}, // second escape is high surrogate, not low → lone high
}, commonUnicodeEscapeTests...)

// Verifies: SYS-REQ-014 [malformed]
// MCDC SYS-REQ-014: N/A
func TestDecodeSingleUnicodeEscape(t *testing.T) {
	for _, test := range singleUnicodeEscapeTests {
		r, ok := decodeSingleUnicodeEscape([]byte(test.in))
		isErr := !ok

		if isErr != test.isErr {
			t.Errorf("decodeSingleUnicodeEscape(%s) returned isErr mismatch: expected %t, obtained %t", test.in, test.isErr, isErr)
		} else if isErr {
			continue
		} else if r != test.out {
			t.Errorf("decodeSingleUnicodeEscape(%s) returned rune mismatch: expected %x (%c), obtained %x (%c)", test.in, test.out, test.out, r, r)
		}
	}
}

// Verifies: SYS-REQ-014 [malformed]
// MCDC SYS-REQ-014: N/A
func TestDecodeUnicodeEscape(t *testing.T) {
	for _, test := range multiUnicodeEscapeTests {
		r, len := decodeUnicodeEscape([]byte(test.in))
		isErr := (len == -1)

		if isErr != test.isErr {
			t.Errorf("decodeUnicodeEscape(%s) returned isErr mismatch: expected %t, obtained %t", test.in, test.isErr, isErr)
		} else if isErr {
			continue
		} else if len != test.len {
			t.Errorf("decodeUnicodeEscape(%s) returned length mismatch: expected %d, obtained %d", test.in, test.len, len)
		} else if r != test.out {
			t.Errorf("decodeUnicodeEscape(%s) returned rune mismatch: expected %x (%c), obtained %x (%c)", test.in, test.out, test.out, r, r)
		}
	}
}

type unescapeTest struct {
	in       string // escaped string
	out      string // expected unescaped string
	canAlloc bool   // can unescape cause an allocation (depending on buffer size)? true iff 'in' contains escape sequence(s)
	isErr    bool   // should this operation result in an error
}

var unescapeTests = []unescapeTest{
	{in: ``, out: ``, canAlloc: false},
	{in: `a`, out: `a`, canAlloc: false},
	{in: `abcde`, out: `abcde`, canAlloc: false},

	{in: `ab\\de`, out: `ab\de`, canAlloc: true},
	{in: `ab\"de`, out: `ab"de`, canAlloc: true},
	{in: `ab \u00B0 de`, out: `ab ° de`, canAlloc: true},
	{in: `ab \uFF11 de`, out: `ab １ de`, canAlloc: true},
	{in: `\uFFFF`, out: "\uFFFF", canAlloc: true},
	{in: `ab \uD83D\uDE03 de`, out: "ab \U0001F603 de", canAlloc: true},
	{in: `\u0000\u0000\u0000\u0000\u0000`, out: "\u0000\u0000\u0000\u0000\u0000", canAlloc: true},
	{in: `\u0000 \u0000 \u0000 \u0000 \u0000`, out: "\u0000 \u0000 \u0000 \u0000 \u0000", canAlloc: true},
	{in: ` \u0000 \u0000 \u0000 \u0000 \u0000 `, out: " \u0000 \u0000 \u0000 \u0000 \u0000 ", canAlloc: true},

	// Lone surrogates: malformed per RFC 8259/WHATWG. Match encoding/json by
	// substituting U+FFFD and preserving the following literal bytes. See
	// DEFECT-260727-SNGT.
	{in: `\uD800`, out: "\uFFFD", canAlloc: true},
	{in: `abcde\uD800`, out: "abcde\uFFFD", canAlloc: true},
	{in: `ab\uD800de`, out: "ab\uFFFDde", canAlloc: true},
	{in: `\uD800abcde`, out: "\uFFFDabcde", canAlloc: true},
	{in: `\uDB29A7FA`, out: "\uFFFDA7FA", canAlloc: true},   // high surrogate followed by non-escape bytes
	{in: `\uD800\u0041`, out: "\uFFFDA", canAlloc: true},    // high surrogate followed by BMP escape
	{in: `\uDC00`, out: "\uFFFD", canAlloc: true},           // lone low surrogate
	{in: `\uD800\uDC00`, out: "\U00010000", canAlloc: true}, // valid pair still works

	// Truncated / malformed escape sequences (non-surrogate) still error.
	{in: `abcde\`, isErr: true},
	{in: `abcde\x`, isErr: true},
	{in: `abcde\u`, isErr: true},
	{in: `abcde\u1`, isErr: true},
	{in: `abcde\u12`, isErr: true},
	{in: `abcde\u123`, isErr: true},
}

// isSameMemory checks if two slices contain the same memory pointer (meaning one is a
// subslice of the other, with possibly differing lengths/capacities).
// Test helper for SYS-REQ-014.
func isSameMemory(a, b []byte) bool {
	if cap(a) == 0 || cap(b) == 0 {
		return cap(a) == cap(b)
	} else if a, b = a[:1], b[:1]; a[0] != b[0] {
		return false
	} else {
		a[0]++
		same := (a[0] == b[0])
		a[0]--
		return same
	}

}

// Verifies: SYS-REQ-014 [malformed]
// MCDC SYS-REQ-014: N/A
func TestUnescape(t *testing.T) {
	for _, test := range unescapeTests {
		type bufferTestCase struct {
			buf        []byte
			isTooSmall bool
		}

		var bufs []bufferTestCase

		if len(test.in) == 0 {
			// If the input string is length 0, only a buffer of size 0 is a meaningful test
			bufs = []bufferTestCase{{nil, false}}
		} else {
			// For non-empty input strings, we can try several buffer sizes (0, len-1, len)
			bufs = []bufferTestCase{
				{nil, true},
				{make([]byte, 0, len(test.in)-1), true},
				{make([]byte, 0, len(test.in)), false},
			}
		}

		for _, buftest := range bufs {
			in := []byte(test.in)
			buf := buftest.buf

			out, err := Unescape(in, buf)
			isErr := (err != nil)
			isAlloc := !isSameMemory(out, in) && !isSameMemory(out, buf)

			if isErr != test.isErr {
				t.Errorf("Unescape(`%s`, bufsize=%d) returned isErr mismatch: expected %t, obtained %t", test.in, cap(buf), test.isErr, isErr)
				break
			} else if isErr {
				continue
			} else if !bytes.Equal(out, []byte(test.out)) {
				t.Errorf("Unescape(`%s`, bufsize=%d) returned unescaped mismatch: expected `%s` (%v, len %d), obtained `%s` (%v, len %d)", test.in, cap(buf), test.out, []byte(test.out), len(test.out), string(out), out, len(out))
				break
			} else if isAlloc != (test.canAlloc && buftest.isTooSmall) {
				t.Errorf("Unescape(`%s`, bufsize=%d) returned isAlloc mismatch: expected %t, obtained %t", test.in, cap(buf), buftest.isTooSmall, isAlloc)
				break
			}
		}
	}
}

// =============================================================================
// Lone-Unicode-surrogate regression tests (DEFECT-260727-SNGT).
//
// Root cause: decodeUnicodeEscape (escape.go:115) read a high surrogate and
// then called decodeSingleUnicodeEscape(in[6:]) to fetch the low surrogate
// WITHOUT verifying in[6:8] == "\u". decodeSingleUnicodeEscape assumes the
// "\u" prefix and reads hex at fixed offsets, so literal bytes following a
// lone high surrogate (e.g. the "A7FA" after "\uDB29") were misread as a low
// surrogate and a bogus non-BMP code point was synthesized. The fix matches
// encoding/json: a lone surrogate (high or low) is malformed → substitute
// U+FFFD and consume only the 6 bytes of the lone-surrogate escape.
// =============================================================================

const replacementChar = "\uFFFD"

// Verifies: SYS-REQ-014 [boundary] — lone high surrogate → U+FFFD, no
// following bytes consumed.
func TestUnescapeLoneHighSurrogate(t *testing.T) {
	out, err := Unescape([]byte(`\uDB29`), nil)
	if err != nil {
		t.Fatalf("Unescape(`\\uDB29`) returned error %v; expected U+FFFD substitution", err)
	}
	if got := string(out); got != replacementChar {
		t.Errorf("Unescape(`\\uDB29`) = %q; expected %q (U+FFFD)", got, replacementChar)
	}
}

// Verifies: SYS-REQ-014 [boundary] — the bug: high surrogate followed by
// non-escape bytes was synthesizing U+DC271; must be U+FFFD + literal "A7FA".
func TestUnescapeLoneHighSurrogateFollowedByNonEscape(t *testing.T) {
	out, err := Unescape([]byte(`\uDB29A7FA`), nil)
	if err != nil {
		t.Fatalf("Unescape(`\\uDB29A7FA`) returned error %v; expected U+FFFD + literal A7FA", err)
	}
	if want, got := replacementChar+"A7FA", string(out); got != want {
		t.Errorf("Unescape(`\\uDB29A7FA`) = %q; expected %q", got, want)
	}
}

// Verifies: SYS-REQ-014 [boundary] — lone low surrogate → U+FFFD.
func TestUnescapeLoneLowSurrogate(t *testing.T) {
	out, err := Unescape([]byte(`\uDC00`), nil)
	if err != nil {
		t.Fatalf("Unescape(`\\uDC00`) returned error %v; expected U+FFFD substitution", err)
	}
	if got := string(out); got != replacementChar {
		t.Errorf("Unescape(`\\uDC00`) = %q; expected %q (U+FFFD)", got, replacementChar)
	}
}

// Verifies: SYS-REQ-014 [boundary] — high surrogate followed by a "\u" escape
// that is NOT a low surrogate (here a BMP codepoint U+0041 'A') → U+FFFD + 'A'.
func TestUnescapeHighSurrogateThenNonSurrogate(t *testing.T) {
	out, err := Unescape([]byte(`\uD800\u0041`), nil)
	if err != nil {
		t.Fatalf("Unescape(`\\uD800\\u0041`) returned error %v; expected U+FFFD + A", err)
	}
	if want, got := replacementChar+"A", string(out); got != want {
		t.Errorf("Unescape(`\\uD800\\u0041`) = %q; expected %q", got, want)
	}
}

// Verifies: SYS-REQ-014 [happy-path] — a valid UTF-16 surrogate pair must
// still combine into the supplementary-plane code point (regression guard).
func TestUnescapeValidSurrogatePairStillWorks(t *testing.T) {
	out, err := Unescape([]byte(`\uD800\uDC00`), nil)
	if err != nil {
		t.Fatalf("Unescape(`\\uD800\\uDC00`) returned error %v; expected U+10000", err)
	}
	if want, got := "\U00010000", string(out); got != want {
		t.Errorf("Unescape(`\\uD800\\uDC00`) = %q; expected %q", got, want)
	}
}

// Verifies: SYS-REQ-014 [differential] — for a corpus of lone-surrogate
// inputs, ParseString must byte-match encoding/json.Unmarshal (the oracle
// exercised by the FuzzJSONStructureAware checkParseStringGate finder).
func TestParseStringLoneSurrogateMatchesEncodingJSON(t *testing.T) {
	corpus := []string{
		`\uDB29`,
		`\uDB29A7FA`,
		`\uDB29A7FA71 a`,
		`\uD800`,
		`\uD83D`,
		`\uDBFF`,
		`\uDC00`,
		`\uDFFF`,
		`\uDE03`,
		`\uD800\u0041`,
		`\uD800\uD800`,
		`\uD800\uDBFF`,
		`\uD800\uE000`,
		`\uD800\uFFFF`,
		`\uDB29\uDC00`,
		`\uD83D\uDE03 more`,
		`\uD800\uDC00`,
		`\u0000\uD800\u0000`,
	}
	for _, esc := range corpus {
		// Reference oracle: encoding/json unescapes the quoted string.
		var want string
		jsonInput := `"` + esc + `"`
		if err := json.Unmarshal([]byte(jsonInput), &want); err != nil {
			// encoding/json rejects malformed escapes (e.g. truncated "\u"). The
			// corpus above is chosen to all be accepted; if one is rejected we
			// want to know rather than silently skip.
			t.Errorf("encoding/json rejected corpus input %q: %v", jsonInput, err)
			continue
		}

		got, err := ParseString([]byte(esc))
		if err != nil {
			t.Errorf("ParseString(%q) returned error %v; encoding/json accepted it as %q", esc, err, want)
			continue
		}
		if got != want {
			t.Errorf("ParseString(%q) divergence:\n  got  = %q (% x)\n  want = %q (% x)",
				esc, got, []byte(got), want, []byte(want))
		}
	}
}

// Verifies: SYS-REQ-014 (escape handling)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestEscape(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "hello", want: `"hello"`},
		{name: "quotation mark", in: `"`, want: `"\""`},
		{name: "reverse solidus", in: `\`, want: `"\\"`},
		{name: "solidus", in: `/`, want: `"/"`},
		{name: "backspace", in: "\b", want: `"\b"`},
		{name: "form feed", in: "\f", want: `"\f"`},
		{name: "line feed", in: "\n", want: `"\n"`},
		{name: "carriage return", in: "\r", want: `"\r"`},
		{name: "tab", in: "\t", want: `"\t"`},
		{name: "unicode escape", in: "\x01", want: `"\u0001"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			escaped := Escape(test.in)
			if got := string(escaped); got != test.want {
				t.Fatalf("Escape(%q) = %q; want %q", test.in, got, test.want)
			}

			unescaped, err := Unescape(escaped[1:len(escaped)-1], nil)
			if err != nil {
				t.Fatalf("Unescape(Escape(%q)) returned error: %v", test.in, err)
			}
			if got := string(unescaped); got != test.in {
				t.Fatalf("Unescape(Escape(%q)) = %q; want %q", test.in, got, test.in)
			}
		})
	}
}

// Verifies: SYS-REQ-014 (escape handling)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestEscapeControlChars(t *testing.T) {
	for c := byte(0); c < 0x20; c++ {
		escaped := Escape(string([]byte{c}))

		var want string
		switch c {
		case '\b':
			want = `"\b"`
		case '\f':
			want = `"\f"`
		case '\n':
			want = `"\n"`
		case '\r':
			want = `"\r"`
		case '\t':
			want = `"\t"`
		default:
			want = string([]byte{'"', '\\', 'u', '0', '0', lowerHex[c>>4], lowerHex[c&0x0f], '"'})
		}

		if got := string(escaped); got != want {
			t.Errorf("Escape control character 0x%02x = %q; want %q", c, got, want)
		}

		unescaped, err := Unescape(escaped[1:len(escaped)-1], nil)
		if err != nil {
			t.Errorf("Unescape(Escape(0x%02x)) returned error: %v", c, err)
			continue
		}
		if len(unescaped) != 1 || unescaped[0] != c {
			t.Errorf("Unescape(Escape(0x%02x)) = % x; want %02x", c, unescaped, c)
		}
	}
}

// Verifies: SYS-REQ-014 (escape handling)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestEscapeUnicode(t *testing.T) {
	in := "你好，世界 🌍"
	escaped := Escape(in)
	if want := `"` + in + `"`; string(escaped) != want {
		t.Fatalf("Escape(%q) = %q; want %q", in, escaped, want)
	}

	unescaped, err := Unescape(escaped[1:len(escaped)-1], nil)
	if err != nil {
		t.Fatalf("Unescape(Escape(%q)) returned error: %v", in, err)
	}
	if got := string(unescaped); got != in {
		t.Fatalf("Unescape(Escape(%q)) = %q; want %q", in, got, in)
	}
}

// Verifies: SYS-REQ-014 (escape handling)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestEscapeEmptyString(t *testing.T) {
	if got, want := string(Escape("")), `""`; got != want {
		t.Fatalf("Escape(\"\") = %q; want %q", got, want)
	}
}

// Verifies: SYS-REQ-009 (Set)
// Verifies: SYS-REQ-009 (Set)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestSetString(t *testing.T) {
	got, err := SetString([]byte(`{"key":"old"}`), "hello", "key")
	if err != nil {
		t.Fatalf("SetString returned error: %v", err)
	}
	if !json.Valid(got) {
		t.Fatalf("SetString returned invalid JSON: %s", got)
	}
	if want := `{"key":"hello"}`; string(got) != want {
		t.Fatalf("SetString result = %s; want %s", got, want)
	}
}

// Verifies: SYS-REQ-002 (GetString)
// Verifies: SYS-REQ-009 (Set)
// Verifies: SYS-REQ-009 (Set)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestSetStringRoundTrip(t *testing.T) {
	const want = "你好 🌍"
	data, err := SetString([]byte(`{"key":null}`), want, "key")
	if err != nil {
		t.Fatalf("SetString returned error: %v", err)
	}

	got, err := GetString(data, "key")
	if err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
	if got != want {
		t.Fatalf("GetString after SetString = %q; want %q", got, want)
	}
}

// Verifies: SYS-REQ-002 (GetString)
// Verifies: SYS-REQ-009 (Set)
// Verifies: SYS-REQ-009 (Set)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestSetStringSpecialChars(t *testing.T) {
	tests := []string{
		`a\b`,
		`a"b`,
		"a\nb",
		"a\tb",
	}

	for _, want := range tests {
		t.Run(want, func(t *testing.T) {
			data, err := SetString([]byte(`{"key":""}`), want, "key")
			if err != nil {
				t.Fatalf("SetString(%q) returned error: %v", want, err)
			}
			if !json.Valid(data) {
				t.Fatalf("SetString(%q) returned invalid JSON: %s", want, data)
			}

			got, err := GetString(data, "key")
			if err != nil {
				t.Fatalf("GetString after SetString(%q) returned error: %v", want, err)
			}
			if got != want {
				t.Fatalf("GetString after SetString(%q) = %q; want %q", want, got, want)
			}
		})
	}
}
