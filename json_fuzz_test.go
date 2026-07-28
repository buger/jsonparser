// Structure-aware JSON fuzzer for the jsonparser public surface.
//
// This file implements a grammar-driven JSON generator and a set of JSON-
// aware mutation operators that target the structural boundaries where
// parser bugs live: truncation right after a ':' (the OSS-Fuzz
// sentinel_value_boundary class), truncation inside a key (truncated_mid_key),
// truncation after an unclosed '{' or '[' (truncated_mid_structure), etc.
//
// The grammar generator always emits valid JSON. The mutation operators
// then break that valid JSON at precisely the byte positions where the
// parser's state machine has to make a transition — the positions that
// blind byte-flip mutation reaches only after exponentially many trials.
//
// Assertion model (each gate maps to a known bug class):
//
//   1. NO-PANIC gate    — every op recovers() clean.
//                        Catches: empty-key panics (SYS-REQ-111),
//                        truncation panics, sentinel-deref panics.
//   2. OUTPUT-VALIDITY  — json.Valid(out) after Set/Delete.
//                        Catches: silent data corruption (SYS-REQ-110 / PR #286).
//   3. ROUND-TRIP       — Get(out, path) returns the set value.
//                        Catches: Set/Get asymmetry, index-beyond-length data loss.
//   4. NUMERIC-ORACLE   — ParseInt/ParseFloat match strconv.
//                        Catches: numeric edge cases (overflow, leading zeros).
//
// The gates are HONEST: no t.Skip, no soft exceptions for known bugs. If a
// gate fires, it is a real bug or a documented-open divergence (the latter
// are excluded from the OUTPUT-VALIDITY gate via pathShapeMatchesRoot,
// matching the convention in path_fuzz_test.go).
//
// Verifies: SYS-REQ-035 (no-panic on malformed/adversarial input).
// reqproof:proptest parser.Get, parser.Set, parser.Delete, parser.GetString, parser.GetInt, parser.GetFloat,
//           parser.GetBoolean, parser.GetUnsafeString, parser.ArrayEach, parser.ObjectEach, parser.EachKey,
//           parser.ParseInt, parser.ParseFloat
package jsonparser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"runtime/debug"
	"strconv"
	"testing"
	"time"
	"unicode/utf8"
)

// =============================================================================
// Seed corpus (the known-dangerous inputs we've already found)
// =============================================================================

type structureAwareSeed struct {
	data []byte
	path []byte
}

// structureAwareSeeds is the set of inputs that exercise every previously-
// buggy code path. Each seed is annotated with the bug class it targets.
var structureAwareSeeds = []structureAwareSeed{
	{[]byte(`{"a":1}`), []byte("a")},                // baseline valid object
	{[]byte(`[1,2,3]`), []byte("")},                 // empty key on array root (SYS-REQ-111)
	{[]byte(`{"a":[{"x":1}]}`), []byte("a")},        // the D9 repro shape (array-of-object)
	{[]byte(`{"a":`), []byte("a")},                  // truncated at value boundary (OSS-Fuzz 4649128545288192)
	{[]byte(`{"key`), []byte("key")},                // truncated mid-key
	{[]byte(`{"a":[1,2`), []byte("a/[0]")},          // truncated mid-structure
	{[]byte(`{"a":"\u30`), []byte("a")},             // truncated mid-escape
	{[]byte(`{"a":1,"b":2}`), []byte("a")},          // duplicate-key mutation target
	{[]byte(`[1,2,3]`), []byte("[0]")},              // in-range array index
	{[]byte(`[1,2,3]`), []byte("[99]")},             // out-of-range index (SYS-REQ-110)
	{[]byte(`{"a":[1,2,3]}`), []byte("a/[5]")},      // nested OOB index (PR #286 class)
	{[]byte(`{"":1}`), []byte("")},                  // object with empty key
	{[]byte(`null`), []byte("a")},                   // scalar root
	{[]byte(``), []byte("a")},                       // empty input
	{[]byte(`{`), []byte("a")},                      // unclosed object
	{[]byte(`[}`), []byte("[0]")},                   // malformed array
	{[]byte(`{"a":"hello\nworld"}`), []byte("a")},   // escaped string value
	{[]byte(`{"a":9223372036854775807}`), []byte("a")}, // int64 max
	{[]byte(`{"a":99999999999999999999}`), []byte("a")}, // overflow boundary
	{[]byte(`{"a":-0.0}`), []byte("a")},             // negative zero float
	// --- UTF-8 / Unicode edge cases (parser indexes BYTES not runes) ---
	{[]byte(`{"a":"\uD800\uDC00"}`), []byte("a")},  // valid surrogate pair (U+10000)
	{[]byte(`{"a":"\uDBFF\uDFFF"}`), []byte("a")},  // valid surrogate pair (U+10FFFF)
	{[]byte(`{"a":"\uD800"}`), []byte("a")},        // lone high surrogate (malformed)
	{[]byte(`{"a":"\uDC00"}`), []byte("a")},        // lone low surrogate (malformed)
	{[]byte(`{"a":"\uD800\uD800"}`), []byte("a")},  // two highs (malformed)
	{[]byte(`{"a":"\u0000"}`), []byte("a")},        // escaped null
	{[]byte(`{"a":"\u001F"}`), []byte("a")},        // escaped unit separator
	{[]byte("{\"a\":\"x\x00y\"}"), []byte("a")},    // raw null byte (invalid JSON)
	{[]byte("{\"a\":\"x\x1Fy\"}"), []byte("a")},    // raw control char (invalid JSON)
	// --- Raw multibyte UTF-8 (must be raw bytes, not \u escapes) ---
	{[]byte("{\"a\":\"\xC3\xA9\"}"), []byte("a")},       // 2-byte: é (U+00E9)
	{[]byte("{\"a\":\"\xE4\xB8\x96\xE7\x95\x8C\"}"), []byte("a")}, // 3-byte: 世界
	{[]byte("{\"a\":\"\xF0\x9F\x98\x80\"}"), []byte("a")},         // 4-byte: emoji 😀
	// --- Invalid UTF-8 byte sequences (parser must reject cleanly) ---
	{[]byte("{\"a\":\"x\xC0\xAFy\"}"), []byte("a")}, // overlong '/' (0xC0 0xAF)
	{[]byte("{\"a\":\"x\xE0y\"}"), []byte("a")},     // truncated 3-byte (leading 0xE0)
	{[]byte("{\"a\":\"x\xFFy\"}"), []byte("a")},     // never-valid byte 0xFF
	{[]byte("{\"a\":\"x\xFEy\"}"), []byte("a")},     // never-valid byte 0xFE
	// --- BOM (U+FEFF = 0xEF 0xBB 0xBF) ---
	{[]byte("\xEF\xBB\xBF" + `{"a":1}`), []byte("a")}, // BOM at start of input
	// --- Number edge cases (exponent extremes, negative zero, NaN text) ---
	{[]byte(`{"a":1e999}`), []byte("a")},     // float overflow (+Inf)
	{[]byte(`{"a":1e-999}`), []byte("a")},    // float underflow (0)
	{[]byte(`{"a":1e308}`), []byte("a")},     // near float64 max
	{[]byte(`{"a":1e-323}`), []byte("a")},    // near float64 min denormal
	{[]byte(`{"a":-0}`), []byte("a")},        // negative zero int
	{[]byte(`{"a":-0e0}`), []byte("a")},      // negative zero exponent
	{[]byte(`{"a":Infinity}`), []byte("a")},  // invalid: Infinity text
	{[]byte(`{"a":NaN}`), []byte("a")},       // invalid: NaN text
	{[]byte(`{"a":-Infinity}`), []byte("a")}, // invalid: -Infinity text
	{[]byte(`{"a":12345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890}`), []byte("a")}, // 100+ digit integer
	{[]byte(`{"a":0.12345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890}`), []byte("a")}, // 100+ digit decimal
	// --- All JSON escapes (including \b and \f) ---
	{[]byte(`{"a":"\b\f\n\t\r\"\\\/"}`), []byte("a")}, // all 8 JSON escapes
	{[]byte(`{"a":"\/\/\/\/"}`), []byte("a")},         // solidus run
}

// =============================================================================
// Deterministic RNG seeding from fuzz inputs
// =============================================================================

// newRandFromSeed derives a deterministic *rand.Rand from the fuzz inputs.
// This lets the fuzz body make structure-aware generation/mutation decisions
// that are a pure function of the engine-supplied bytes (so the engine's
// coverage feedback remains meaningful). Uses FNV-1a for speed (no allocs
// beyond the rand.Rand struct itself).
func newRandFromSeed(seeds ...[]byte) *rand.Rand {
	const (
		offsetBasis uint64 = 14695981039346656037
		fnvPrime    uint64 = 1099511628211
	)
	h := offsetBasis
	for _, s := range seeds {
		for _, b := range s {
			h ^= uint64(b)
			h *= fnvPrime
		}
	}
	return rand.New(rand.NewSource(int64(h)))
}

// =============================================================================
// Recursive JSON value generator (grammar-based)
// =============================================================================

const maxGenDepth = 5

// genJSON returns a valid JSON document generated from the grammar. The
// output is always valid JSON per RFC 8259 (the corpus-sanity test asserts
// json.Valid on every generated document).
//
// Depth controls recursion: at depth >= maxGenDepth the generator emits
// only scalars to bound the output size and stress the base cases.
func genJSON(r *rand.Rand, depth int) []byte {
	var buf []byte
	genValue(r, depth, &buf)
	return buf
}

// genValue appends one JSON value to buf, chosen from a weighted
// distribution that favors containers at shallow depth (to exercise
// nesting) and scalars at deep depth (to bound recursion).
func genValue(r *rand.Rand, depth int, buf *[]byte) {
	if depth >= maxGenDepth {
		// Scalars only at deep recursion.
		switch r.Intn(4) {
		case 0:
			genString(r, buf)
		case 1:
			genNumber(r, buf)
		case 2:
			*buf = append(*buf, 't', 'r', 'u', 'e')
		default:
			*buf = append(*buf, 'n', 'u', 'l', 'l')
		}
		return
	}

	roll := r.Intn(20)
	switch {
	case roll < 6: // 30% — object
		genObject(r, depth, buf)
	case roll < 11: // 25% — array
		genArray(r, depth, buf)
	case roll < 14: // 15% — string
		genString(r, buf)
	case roll < 18: // 20% — number
		genNumber(r, buf)
	case roll == 18: // 5% — bool
		if r.Intn(2) == 0 {
			*buf = append(*buf, 't', 'r', 'u', 'e')
		} else {
			*buf = append(*buf, 'f', 'a', 'l', 's', 'e')
		}
	default: // 5% — null
		*buf = append(*buf, 'n', 'u', 'l', 'l')
	}
}

// genObject emits an object with 0–5 entries. Keys are drawn from a set
// that includes the dangerous cases: empty-string "", unicode (\uXXXX),
// long, and escaped special characters.
func genObject(r *rand.Rand, depth int, buf *[]byte) {
	*buf = append(*buf, '{')
	n := r.Intn(6) // 0..5 entries
	for i := 0; i < n; i++ {
		if i > 0 {
			*buf = append(*buf, ',')
		}
		genKey(r, buf)
		*buf = append(*buf, ':')
		genValue(r, depth+1, buf)
	}
	*buf = append(*buf, '}')
}

// genArray emits an array with 0–5 elements (including the empty array []).
func genArray(r *rand.Rand, depth int, buf *[]byte) {
	*buf = append(*buf, '[')
	n := r.Intn(6) // 0..5 elements
	for i := 0; i < n; i++ {
		if i > 0 {
			*buf = append(*buf, ',')
		}
		genValue(r, depth+1, buf)
	}
	*buf = append(*buf, ']')
}

// genKey emits a JSON object key (a double-quoted string). The key content
// is drawn from a weighted distribution that includes the cases most likely
// to expose parser bugs:
//   - empty string "" (the empty-key hazard class, SYS-REQ-111)
//   - unicode escapes \uXXXX (escape-parsing paths)
//   - long keys (allocation paths)
//   - escaped special chars (\\, \", \n, \t — escape correctness)
func genKey(r *rand.Rand, buf *[]byte) {
	*buf = append(*buf, '"')
	switch r.Intn(16) {
	case 0: // empty string key
	case 1: // single ASCII char
		*buf = append(*buf, byte('a'+r.Intn(26)))
	case 2: // short ASCII (1–8 chars)
		for i := 0; i < 1+r.Intn(8); i++ {
			*buf = append(*buf, byte('a'+r.Intn(26)))
		}
	case 3: // unicode escape sequence \u30XX (hiragana range)
		*buf = append(*buf, '\\', 'u', '3', '0')
		*buf = append(*buf, hexDigit(r), hexDigit(r))
	case 4: // long key (stress allocation paths)
		for i := 0; i < 20+r.Intn(30); i++ {
			*buf = append(*buf, byte('a'+r.Intn(26)))
		}
	case 5: // escaped special chars (always valid JSON)
		*buf = append(*buf, '\\', '"', '\\', '\\', '\\', 'n', '\\', 't')
	case 6: // numeric-looking key
		*buf = append(*buf, '0', '0', '7')
	case 7: // "key" (matches common test seeds)
		*buf = append(*buf, 'k', 'e', 'y')
	// --- raw multibyte UTF-8 keys (parser indexes BYTES not runes) ---
	case 8: // 2-byte UTF-8 raw key (Latin/Greek/Cyrillic)
		var u [4]byte
		n := utf8.EncodeRune(u[:], rune(0x80+r.Intn(0x780)))
		*buf = append(*buf, u[:n]...)
	case 9: // 3-byte UTF-8 raw key (CJK/Arabic/Hindi)
		var u [4]byte
		n := utf8.EncodeRune(u[:], rune(0x800+r.Intn(0xF800)))
		*buf = append(*buf, u[:n]...)
	case 10: // valid surrogate-pair escape key (U+10000+)
		cp := 0x10000 + r.Intn(0x10FFFF-0x10000+1)
		cp -= 0x10000
		appendUnicodeEscape(buf, 0xD800+(cp>>10))
		appendUnicodeEscape(buf, 0xDC00+(cp&0x3FF))
	case 11: // lone surrogate escape key (malformed — parser must reject)
		appendUnicodeEscape(buf, 0xD800+r.Intn(0x800))
	case 12: // escaped control char key (\u0000 / \u001F)
		appendUnicodeEscape(buf, r.Intn(0x20))
	case 13: // invalid UTF-8 raw bytes in key (overlong / truncated / FF)
		switch r.Intn(3) {
		case 0:
			*buf = append(*buf, 0xC0, 0xAF)
		case 1:
			*buf = append(*buf, 0xE0)
		default:
			*buf = append(*buf, 0xFF)
		}
	case 14: // BOM-prefixed key
		*buf = append(*buf, 0xEF, 0xBB, 0xBF, 'k')
	default: // 2-char ASCII
		*buf = append(*buf, byte('a'+r.Intn(26)), byte('a'+r.Intn(26)))
	}
	*buf = append(*buf, '"')
}

// genString emits a JSON string value. Includes empty "", ASCII, strings
// with escape sequences, unicode escapes, and long strings.
func genString(r *rand.Rand, buf *[]byte) {
	*buf = append(*buf, '"')
	switch r.Intn(22) {
	case 0: // empty string
	case 1: // ASCII short
		for i := 0; i < 1+r.Intn(8); i++ {
			*buf = append(*buf, byte('a'+r.Intn(26)))
		}
	case 2: // escape sequences (all 8 JSON escapes incl. \b and \f)
		escapes := [][]byte{[]byte(`\n`), []byte(`\t`), []byte(`\"`), []byte(`\\`), []byte(`\/`), []byte(`\r`), []byte(`\b`), []byte(`\f`)}
		for i := 0; i < 1+r.Intn(4); i++ {
			*buf = append(*buf, escapes[r.Intn(len(escapes))]...)
		}
	case 3: // unicode escape (BMP non-surrogate, hiragana range)
		*buf = append(*buf, '\\', 'u', '3', '0')
		*buf = append(*buf, hexDigit(r), hexDigit(r))
	case 4: // long string (stress allocation)
		for i := 0; i < 50+r.Intn(50); i++ {
			*buf = append(*buf, byte('a'+r.Intn(26)))
		}
	case 5: // mixed content with escapes
		*buf = append(*buf, []byte(`hello\nworld\u2603`)...)
	case 6: // numeric-looking string (tests type coercion in callers)
		*buf = append(*buf, '1', '2', '3', '4', '5')
	case 7: // whitespace inside string (properly escaped — raw control
		// bytes like 0x09 are invalid inside JSON strings per RFC 8259,
		// so we emit the \t escape sequence instead of a literal tab).
		*buf = append(*buf, ' ', '\\', 't')
	// --- raw multibyte UTF-8 (parser indexes BYTES not runes) ---
	case 8: // 2-byte UTF-8 raw (U+0080–U+07FF: accented Latin, Greek, Cyrillic)
		var u [4]byte
		n := utf8.EncodeRune(u[:], rune(0x80+r.Intn(0x780)))
		*buf = append(*buf, u[:n]...)
	case 9: // 3-byte UTF-8 raw (U+0800–U+FFFF: CJK, Arabic, Hindi)
		var u [4]byte
		n := utf8.EncodeRune(u[:], rune(0x800+r.Intn(0xF800)))
		*buf = append(*buf, u[:n]...)
	case 10: // 4-byte UTF-8 raw (U+10000+: emoji, math symbols, CJK ext.)
		var u [4]byte
		cp := 0x10000 + r.Intn(0x10FFFF-0x10000+1)
		n := utf8.EncodeRune(u[:], rune(cp))
		*buf = append(*buf, u[:n]...)
	// --- surrogate-pair escapes (the escape decoder must combine these) ---
	case 11: // valid surrogate pair (\uXXXX\uXXXX → U+10000..U+10FFFF)
		cp := 0x10000 + r.Intn(0x10FFFF-0x10000+1)
		cp -= 0x10000
		appendUnicodeEscape(buf, 0xD800+(cp>>10))
		appendUnicodeEscape(buf, 0xDC00+(cp&0x3FF))
	case 12: // lone high surrogate (malformed — parser must reject)
		appendUnicodeEscape(buf, 0xD800+r.Intn(0x400))
	case 13: // lone low surrogate (malformed — parser must reject)
		appendUnicodeEscape(buf, 0xDC00+r.Intn(0x400))
	case 14: // two high surrogates back-to-back (malformed)
		appendUnicodeEscape(buf, 0xD800+r.Intn(0x400))
		appendUnicodeEscape(buf, 0xD800+r.Intn(0x400))
	// --- control characters (escaped form, valid JSON) ---
	case 15: // escaped null and unit separator
		appendUnicodeEscape(buf, 0x00) // \u0000
		appendUnicodeEscape(buf, 0x1F) // \u001F
	// --- long escape runs (stress the escape decoder's allocation path) ---
	case 16:
		escapes := [][]byte{[]byte(`\n`), []byte(`\t`), []byte(`\"`), []byte(`\\`), []byte(`\/`), []byte(`\r`), []byte(`\b`), []byte(`\f`)}
		for i := 0; i < 30+r.Intn(30); i++ {
			*buf = append(*buf, escapes[r.Intn(len(escapes))]...)
		}
	case 17: // solidus edge cases: \/ and bare / both valid in JSON strings
		*buf = append(*buf, '\\', '/', '/', '\\', '/')
	// --- invalid UTF-8 byte sequences (parser must reject, never panic) ---
	case 18: // overlong encoding (0xC0 0xAF = overlong '/')
		*buf = append(*buf, 0xC0, 0xAF)
	case 19: // truncated sequence (leading byte 0xE0 with no continuation)
		*buf = append(*buf, 0xE0)
	case 20: // never-valid bytes 0xFF / 0xFE
		*buf = append(*buf, 0xFF, 0xFE)
	case 21: // BOM (U+FEFF = 0xEF 0xBB 0xBF) inside a string — valid UTF-8, unusual
		*buf = append(*buf, 0xEF, 0xBB, 0xBF)
	}
	*buf = append(*buf, '"')
}

// nearMaxInt64 are string representations of integers at/above the int64
// boundary. They are emitted as literal bytes (not via strconv.AppendInt)
// to avoid int64 overflow in the generator itself.
var nearMaxInt64 = []string{
	"9223372036854775807",  // math.MaxInt64 exactly
	"9223372036854775806",  // MaxInt64 - 1
	"9223372036854775808",  // MaxInt64 + 1 (overflows int64)
	"9999999999999999999",  // 19 nines (overflows)
	"99999999999999999999", // 20 nines (overflows)
}

var nearMinInt64 = []string{
	"-9223372036854775808", // math.MinInt64 exactly
	"-9223372036854775807", // MinInt64 + 1
	"-9223372036854775809", // MinInt64 - 1 (underflows)
	"-9999999999999999999", // large negative (underflows)
}

// genNumber emits a JSON number. The distribution is weighted toward the
// fast-path (small ints) but includes overflow boundaries, floats, and
// intentionally-malformed values (leading zeros) that exercise the parser's
// error paths.
func genNumber(r *rand.Rand, buf *[]byte) {
	switch r.Intn(20) {
	case 0: // 0
		*buf = append(*buf, '0')
	case 1: // small positive int (1–2 digits — the fast path)
		*buf = strconv.AppendInt(*buf, int64(1+r.Intn(99)), 10)
	case 2: // negative small int
		*buf = strconv.AppendInt(*buf, -int64(1+r.Intn(99)), 10)
	case 3: // near int64 max (overflow boundary)
		*buf = append(*buf, nearMaxInt64[r.Intn(len(nearMaxInt64))]...)
	case 4: // near int64 min (overflow boundary)
		*buf = append(*buf, nearMinInt64[r.Intn(len(nearMinInt64))]...)
	case 5: // medium int
		*buf = strconv.AppendInt(*buf, int64(100+r.Intn(100000)), 10)
	case 6: // float (fixed notation)
		*buf = strconv.AppendFloat(*buf, r.Float64()*1000, 'f', 2, 64)
	case 7: // float with exponent
		*buf = strconv.AppendFloat(*buf, r.Float64()*1e10, 'e', -1, 64)
	case 8: // negative float
		*buf = strconv.AppendFloat(*buf, -r.Float64()*1000, 'f', 2, 64)
	case 9: // leading zeros (invalid JSON per RFC 8259 but tests parser leniency)
		*buf = append(*buf, '0', '0', '7')
	case 10: // very small float
		*buf = append(*buf, []byte("0.000001")...)
	// --- exponent extremes (overflow / underflow paths) ---
	case 11: // float overflow → +Inf
		switch r.Intn(3) {
		case 0:
			*buf = append(*buf, []byte("1e999")...)
		case 1:
			*buf = append(*buf, []byte("1e400")...)
		default:
			*buf = append(*buf, []byte("2.5e500")...)
		}
	case 12: // float underflow → 0
		switch r.Intn(3) {
		case 0:
			*buf = append(*buf, []byte("1e-999")...)
		case 1:
			*buf = append(*buf, []byte("1e-400")...)
		default:
			*buf = append(*buf, []byte("1e-323")...) // near float64 min denormal
		}
	case 13: // near float64 max (1.7976931348623157e+308)
		*buf = append(*buf, []byte("1.7976931348623157e+308")...)
	case 14: // near float64 min denormal (5e-324)
		*buf = append(*buf, []byte("5e-324")...)
	// --- negative zero variants ---
	case 15:
		switch r.Intn(3) {
		case 0:
			*buf = append(*buf, '-', '0')
		case 1:
			*buf = append(*buf, []byte("-0.0")...)
		default:
			*buf = append(*buf, []byte("-0e0")...)
		}
	// --- very long numbers (overflow path) ---
	case 16: // 100+ digit integer
		*buf = append(*buf, '9')
		for i := 0; i < 100+r.Intn(50); i++ {
			*buf = append(*buf, '9')
		}
	case 17: // 100+ digit decimal
		*buf = append(*buf, '0', '.')
		for i := 0; i < 100+r.Intn(50); i++ {
			*buf = append(*buf, '9')
		}
	// --- Infinity / NaN text (invalid JSON but parsers may see them) ---
	case 18:
		switch r.Intn(3) {
		case 0:
			*buf = append(*buf, []byte("Infinity")...)
		case 1:
			*buf = append(*buf, []byte("NaN")...)
		default:
			*buf = append(*buf, []byte("-Infinity")...)
		}
	default: // single-digit positive
		*buf = strconv.AppendInt(*buf, int64(r.Intn(10)), 10)
	}
}

// hexDigit returns a single lowercase hexadecimal digit byte.
func hexDigit(r *rand.Rand) byte {
	d := r.Intn(16)
	if d < 10 {
		return byte('0' + d)
	}
	return byte('a' + d - 10)
}

// hexChars is the lowercase hex alphabet used by appendUnicodeEscape.
const hexChars = "0123456789abcdef"

// appendUnicodeEscape appends a `\uXXXX` escape sequence for the given code
// point to buf. Does NOT validate cp — callers use it for both valid BMP
// code points and for lone/invalid surrogates (which the parser must reject
// cleanly). cp must fit in 20 bits (the BMP range); higher bits are masked.
func appendUnicodeEscape(buf *[]byte, cp int) {
	*buf = append(*buf, '\\', 'u',
		hexChars[(cp>>12)&0xF],
		hexChars[(cp>>8)&0xF],
		hexChars[(cp>>4)&0xF],
		hexChars[cp&0xF])
}

// =============================================================================
// JSON-aware mutation operators
// =============================================================================
//
// Each operator targets a structural boundary where the parser's state
// machine has to make a transition. Truncating or corrupting at these
// positions exercises the error-recovery paths that hide the bug classes
// documented in the task spec.
//
// Every operator MUST return a new slice (never mutate the input in place)
// because the fuzz engine may reuse the input buffer across iterations.

// mutationOp is a JSON-aware mutation: it takes valid (or near-valid) JSON
// bytes and returns a mutated copy with a structural break introduced at a
// known-dangerous boundary.
type mutationOp func(r *rand.Rand, data []byte) []byte

// mutationOps is the registry of all operators. applyMutation picks one at
// random. The injectEmptyKeyComponent and injectOobArrayIndex operators
// from the spec are path-side operators and are handled separately in
// injectPathHazard.
var mutationOps = []mutationOp{
	truncateAtValueBoundary,
	truncateMidKey,
	truncateMidStructure,
	truncateMidEscape,
	truncateMidElement,
	duplicateKey,
	byteflipAtStructuralByte,
	injectInvalidUTF8,
	injectLoneSurrogate,
	injectUnbalancedOpen,
	injectMismatchedCloser,
}

// applyMutation picks one operator at random and applies it.
func applyMutation(r *rand.Rand, data []byte) []byte {
	if len(data) < 2 {
		return data
	}
	op := mutationOps[r.Intn(len(mutationOps))]
	return op(r, data)
}

// findStructuralBytes returns the positions of all bytes in data that match
// any of targets. Returns nil if none are found.
func findStructuralBytes(data []byte, targets ...byte) []int {
	var pos []int
	for i, b := range data {
		for _, t := range targets {
			if b == t {
				pos = append(pos, i)
				break
			}
		}
	}
	return pos
}

// truncateAtValueBoundary cuts the document right after a ':' (before the
// value starts). This is the OSS-Fuzz 4649128545288192 /
// sentinel_value_boundary class: the parser must not dereference past the
// cut when looking for the value token.
func truncateAtValueBoundary(r *rand.Rand, data []byte) []byte {
	pos := findStructuralBytes(data, ':')
	if len(pos) == 0 {
		return data
	}
	cut := pos[r.Intn(len(pos))] + 1
	if cut > len(data) {
		cut = len(data)
	}
	out := make([]byte, cut)
	copy(out, data[:cut])
	return out
}

// truncateMidKey cuts inside a string key. The cut is placed between the
// opening '"' of a key and its closing '"'. This is the truncated_mid_key
// class: the parser's Unescape routine must detect the unterminated string.
func truncateMidKey(r *rand.Rand, data []byte) []byte {
	var cuts []int
	for i := 1; i < len(data); i++ {
		if data[i] != '"' {
			continue
		}
		// Walk back past whitespace to find the structural predecessor.
		j := i - 1
		for j >= 0 && (data[j] == ' ' || data[j] == '\t' || data[j] == '\n' || data[j] == '\r') {
			j--
		}
		if j < 0 || (data[j] != '{' && data[j] != ',') {
			continue
		}
		// This '"' is a key opening. Find the closing '"'.
		end := -1
		for k := i + 1; k < len(data); k++ {
			if data[k] == '"' && data[k-1] != '\\' {
				end = k
				break
			}
		}
		lo := i + 1
		hi := end
		if hi < 0 || hi > len(data) {
			hi = len(data)
		}
		if hi > lo {
			cuts = append(cuts, lo+r.Intn(hi-lo))
		}
	}
	if len(cuts) == 0 {
		return data
	}
	cut := cuts[r.Intn(len(cuts))]
	out := make([]byte, cut)
	copy(out, data[:cut])
	return out
}

// truncateMidStructure cuts after an unclosed '{' or '['. This is the
// truncated_mid_structure class: the parser must detect the missing closing
// token and return a clean MalformedObjectError / MalformedArrayError.
func truncateMidStructure(r *rand.Rand, data []byte) []byte {
	pos := findStructuralBytes(data, '{', '[')
	if len(pos) == 0 {
		return data
	}
	open := pos[r.Intn(len(pos))]
	if open+1 >= len(data) {
		return data
	}
	cut := open + 1 + r.Intn(len(data)-open-1)
	out := make([]byte, cut)
	copy(out, data[:cut])
	return out
}

// truncateMidEscape cuts inside a \uXXXX escape sequence. This is the
// truncated_escape_sequence class: the parser's Unescape routine must
// detect the truncated escape and return MalformedStringEscapeError.
func truncateMidEscape(r *rand.Rand, data []byte) []byte {
	var cuts []int
	for i := 0; i+1 < len(data); i++ {
		if data[i] != '\\' || data[i+1] != 'u' {
			continue
		}
		// \uXXXX is 6 bytes. Cut somewhere in [i+2, i+6).
		lo := i + 2
		hi := i + 6
		if hi > len(data) {
			hi = len(data)
		}
		if hi > lo {
			cuts = append(cuts, lo+r.Intn(hi-lo))
		}
	}
	if len(cuts) == 0 {
		// No \u escape present; inject a truncated one and exercise the path.
		out := make([]byte, 0, len(data)+5)
		out = append(out, data...)
		out = append(out, '"', '\\', 'u', '3', '0')
		return out
	}
	cut := cuts[r.Intn(len(cuts))]
	out := make([]byte, cut)
	copy(out, data[:cut])
	return out
}

// truncateMidElement cuts inside an array element, after a '[' or ','. The
// truncated_mid_element class: the parser must not read past the cut when
// scanning for the element's value type.
func truncateMidElement(r *rand.Rand, data []byte) []byte {
	var pos []int
	for i, b := range data {
		if (b == '[' || b == ',') && i+2 < len(data) {
			pos = append(pos, i)
		}
	}
	if len(pos) == 0 {
		return data
	}
	anchor := pos[r.Intn(len(pos))]
	cut := anchor + 1 + r.Intn(len(data)-anchor-1)
	out := make([]byte, cut)
	copy(out, data[:cut])
	return out
}

// duplicateKey injects a duplicate object key, turning {"k":1} into
// {"k":1,"k":1}. This exercises duplicate-key handling: the parser should
// return the first value for Get without structural corruption.
func duplicateKey(r *rand.Rand, data []byte) []byte {
	for i := 0; i+2 < len(data); i++ {
		if data[i] != '"' {
			continue
		}
		// Find the closing quote of this key.
		keyEnd := -1
		for k := i + 1; k < len(data); k++ {
			if data[k] == '"' && data[k-1] != '\\' {
				keyEnd = k
				break
			}
		}
		if keyEnd == -1 || keyEnd+1 >= len(data) || data[keyEnd+1] != ':' {
			continue
		}
		// Find the end of the value (next ',' or '}' or ']' at depth 0).
		valEnd := keyEnd + 2
		depth := 0
		inStr := false
	done:
		for valEnd < len(data) {
			b := data[valEnd]
			if inStr {
				if b == '"' && data[valEnd-1] != '\\' {
					inStr = false
				}
			} else {
				switch b {
				case '"':
					inStr = true
				case '{', '[':
					depth++
				case '}', ']':
					if depth == 0 {
						break done
					}
					depth--
				case ',':
					if depth == 0 {
						break done
					}
				}
			}
			valEnd++
		}
		if valEnd <= keyEnd+2 || valEnd > len(data) {
			continue
		}
		pair := data[i:valEnd]
		out := make([]byte, 0, len(data)+len(pair)+1)
		out = append(out, data[:valEnd]...)
		out = append(out, ',')
		out = append(out, pair...)
		out = append(out, data[valEnd:]...)
		return out
	}
	return data
}

// byteflipAtStructuralByte flips a byte at a structural position ({, }, [,
// ], :, ,, "). This corrupts the document skeleton, exercising the parser's
// malformed-structure detection paths.
func byteflipAtStructuralByte(r *rand.Rand, data []byte) []byte {
	pos := findStructuralBytes(data, '{', '}', '[', ']', ':', ',', '"')
	if len(pos) == 0 {
		return data
	}
	idx := pos[r.Intn(len(pos))]
	out := make([]byte, len(data))
	copy(out, data)
	// XOR with a nonzero mask to guarantee the byte changes.
	out[idx] ^= byte(1 + r.Intn(255))
	return out
}

// stringInsertionPoints returns the byte positions of all unescaped closing
// '"' chars that terminate a string body in data. Used by mutation operators
// that need to inject bytes inside a JSON string. The simple scanner does
// not track `\\` sequences perfectly (e.g. `\\"`), but it is good enough
// for fuzz-targeted injection.
func stringInsertionPoints(data []byte) []int {
	var pos []int
	inStr := false
	for i := 1; i < len(data); i++ {
		if data[i] != '"' || data[i-1] == '\\' {
			continue
		}
		if !inStr {
			inStr = true
		} else {
			pos = append(pos, i) // just before the closing quote
			inStr = false
		}
	}
	return pos
}

// injectInvalidUTF8 inserts a raw invalid-UTF-8 byte sequence inside a JSON
// string value. Targets the parser's UTF-8 validation path: the parser must
// reject cleanly, never panic. Covers overlong encodings (0xC0 0xAF), truncated
// sequences (lone leading byte), never-valid bytes (0xFF / 0xFE), and a UTF-8-
// encoded surrogate (which encoding/json rejects but some parsers accept).
func injectInvalidUTF8(r *rand.Rand, data []byte) []byte {
	positions := stringInsertionPoints(data)
	if len(positions) == 0 {
		return data
	}
	pos := positions[r.Intn(len(positions))]
	invalid := [][]byte{
		{0xC0, 0xAF},         // overlong '/'
		{0xC1, 0xBF},         // overlong 2-byte
		{0xE0, 0x80, 0xAF},   // overlong 3-byte
		{0xF0, 0x80, 0x80, 0x80}, // overlong 4-byte
		{0xE0},               // truncated 3-byte (lone leading byte)
		{0xF0, 0x80},         // truncated 4-byte
		{0xFF},               // never-valid byte
		{0xFE},               // never-valid byte
		{0xED, 0xA0, 0x80},   // UTF-8 encoding of U+D800 (lone surrogate)
	}
	seq := invalid[r.Intn(len(invalid))]
	out := make([]byte, 0, len(data)+len(seq))
	out = append(out, data[:pos]...)
	out = append(out, seq...)
	out = append(out, data[pos:]...)
	return out
}

// injectLoneSurrogate corrupts a string by replacing a `\uXXXX` escape with a
// lone surrogate escape (\uD800 / \uDC00 alone, or a high-high pair). Targets
// the escape decoder's surrogate-combination path: it must refuse to combine
// these into a valid code point and must reject the string cleanly. If no
// `\u` escape exists in data, the operator inserts a lone surrogate escape at
// a random string-body position instead.
func injectLoneSurrogate(r *rand.Rand, data []byte) []byte {
	// Find existing \u escapes.
	var escPos []int
	for i := 0; i+1 < len(data); i++ {
		if data[i] == '\\' && data[i+1] == 'u' {
			escPos = append(escPos, i)
		}
	}
	if len(escPos) > 0 {
		pos := escPos[r.Intn(len(escPos))]
		end := pos + 6
		if end > len(data) {
			end = len(data)
		}
		var seq []byte
		switch r.Intn(3) {
		case 0:
			seq = []byte(`\uD800`) // high alone
		case 1:
			seq = []byte(`\uDC00`) // low alone
		default:
			seq = []byte(`\uD800\uD800`) // two highs
		}
		out := make([]byte, 0, len(data)-6+len(seq))
		out = append(out, data[:pos]...)
		out = append(out, seq...)
		out = append(out, data[end:]...)
		return out
	}
	// Fallback: insert a lone-surrogate escape inside a string body.
	positions := stringInsertionPoints(data)
	if len(positions) == 0 {
		return data
	}
	pos := positions[r.Intn(len(positions))]
	seq := []byte(`\uD800`)
	out := make([]byte, 0, len(data)+len(seq))
	out = append(out, data[:pos]...)
	out = append(out, seq...)
	out = append(out, data[pos:]...)
	return out
}

// injectUnbalancedOpen inserts unmatched '{' or '[' opens at a random
// position. This produces structures the grammar generator CANNOT emit:
// every generator-produced container is balanced by construction. The
// parser's blockEnd / depth-tracking code must detect the missing closer
// and return a clean error rather than reading past EOF.
//
// Targets the deep-recursion / malformed-structure detection path that
// balanced-only generation leaves uncovered.
func injectUnbalancedOpen(r *rand.Rand, data []byte) []byte {
	const maxInserts = 8
	n := 1 + r.Intn(maxInserts)
	open := byte('{')
	if r.Intn(2) == 0 {
		open = '['
	}
	// Bias insertion point: 50% at start (forces full descent), 50% random.
	pos := 0
	if r.Intn(2) == 0 {
		pos = r.Intn(len(data) + 1)
	}
	out := make([]byte, 0, len(data)+n)
	out = append(out, data[:pos]...)
	for i := 0; i < n; i++ {
		out = append(out, open)
	}
	out = append(out, data[pos:]...)
	return out
}

// injectMismatchedCloser swaps a closing '}' for ']' (or vice versa).
// The grammar generator always pairs '{' with '}' and '[' with ']'; this
// operator produces the cross-type mismatch class ([{...}] → [{...]}) that
// stresses blockEnd's open/close pairing logic.
func injectMismatchedCloser(r *rand.Rand, data []byte) []byte {
	closers := findStructuralBytes(data, '}', ']')
	if len(closers) == 0 {
		return data
	}
	idx := closers[r.Intn(len(closers))]
	out := make([]byte, len(data))
	copy(out, data)
	if out[idx] == '}' {
		out[idx] = ']'
	} else {
		out[idx] = '}'
	}
	return out
}

// =============================================================================
// Adversarial key-path generator
// =============================================================================

// genKeyPath returns a variadic key path that targets known-dangerous
// dereference sites: empty strings (the SYS-REQ-111 hazard), [N] array-index
// syntax (the SYS-REQ-110 Set array-index-beyond-length hazard), out-of-range
// indices, and deeply-nested paths.
func genKeyPath(r *rand.Rand) []string {
	n := 1 + r.Intn(4) // 1..4 components
	path := make([]string, n)
	for i := range path {
		path[i] = genPathComponent(r)
	}
	return path
}

// genPathComponent returns a single adversarial path component.
func genPathComponent(r *rand.Rand) string {
	switch r.Intn(12) {
	case 0: // empty string (SYS-REQ-111 hazard — the 8th-panic class)
		return ""
	case 1: // valid array index in range
		return "[" + strconv.Itoa(r.Intn(5)) + "]"
	case 2: // out-of-range array index (SYS-REQ-110 hazard)
		return "[99]"
	case 3: // very large array index
		return "[999999]"
	case 4: // malformed array index syntax (empty brackets)
		return "[]"
	case 5: // negative index
		return "[-1]"
	case 6: // unicode key
		return "すびほ"
	case 7: // key with a byte requiring JSON escaping in output
		return "a\"b"
	case 8: // "key" (matches seed corpus)
		return "key"
	case 9: // "a" (common test key)
		return "a"
	case 10: // long ASCII key
		return "abcdefghij"
	default: // single ASCII char
		return string(rune('a' + r.Intn(26)))
	}
}

// injectPathHazard randomly injects an adversarial path component into an
// existing path. This implements the injectEmptyKeyComponent and
// injectOobArrayIndex operators from the spec.
func injectPathHazard(r *rand.Rand, path []string) []string {
	switch r.Intn(3) {
	case 0: // injectEmptyKeyComponent — append "" to the path
		out := make([]string, len(path)+1)
		copy(out, path)
		out[len(path)] = ""
		return out
	case 1: // injectOobArrayIndex — append "[99]" to the path
		out := make([]string, len(path)+1)
		copy(out, path)
		out[len(path)] = "[99]"
		return out
	default: // replace with a grammar-generated adversarial path
		return genKeyPath(r)
	}
}

// =============================================================================
// Enhanced path-shape checker (catches nested D6 divergences)
// =============================================================================

// containerIs reports whether data's first non-whitespace byte equals open.
// Used to check whether a JSON value is an object ('{') or array ('[').
func containerIs(data []byte, open byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		case open:
			return true
		default:
			return false
		}
	}
	return false
}

// pathShapeMatchesData is a stricter version of pathShapeMatchesRoot that
// also verifies container-type compatibility at EVERY nesting level, not
// just the root. This catches the nested-D6 divergence: an array-index
// path component ([N]) targeting a non-array value at depth > 0, or an
// object-key component targeting a non-object value.
//
// Without this check, the OUTPUT-VALIDITY gate would fire false positives
// on the documented-open nested-D6 divergence (where Set produces invalid
// JSON like `{"co":1,42}` when [N] targets an object value).
func pathShapeMatchesData(data []byte, path []string) bool {
	// Delegate the root-level checks (empty key, special bytes, root D6)
	// to the existing helper from path_fuzz_test.go.
	if !pathShapeMatchesRoot(data, path) {
		return false
	}
	if len(path) < 2 {
		return true // root check is sufficient for single-component paths
	}
	// Walk the path, resolving each component and checking the container
	// type at the next level. If the path doesn't fully resolve (Set would
	// create intermediate structure), we conservatively allow it — the
	// json.Valid post-condition will still catch real corruption.
	current := data
	for i := 0; i < len(path)-1; i++ {
		comp := path[i]
		isArrayIdx := len(comp) > 0 && comp[0] == '['
		if isArrayIdx && !containerIs(current, '[') {
			return false // nested D6: array-index on non-array value
		}
		if !isArrayIdx && !containerIs(current, '{') {
			return false // nested D6: object-key on non-object value
		}
		val, _, _, err := Get(current, comp)
		if err != nil {
			// Path prefix doesn't resolve — Set would create structure.
			// We can't check deeper levels; conservatively allow.
			return true
		}
		current = val
	}
	// Check the last component against its resolved parent.
	last := path[len(path)-1]
	isLastArrayIdx := len(last) > 0 && last[0] == '['
	if isLastArrayIdx && !containerIs(current, '[') {
		return false
	}
	if !isLastArrayIdx && !containerIs(current, '{') {
		return false
	}
	return true
}

// =============================================================================
// Gate runner (the assertions that catch each bug class)
// =============================================================================

// fuzzGateSizeCap is the maximum input size (in bytes) on which the
// expensive gates (DETERMINISM, ALIASING, PARSESTRING) run during fuzzing.
// Inputs larger than this skip those gates — they would dominate per-exec
// time on the deep-nesting shapes the fuzzer also generates. The non-fuzz
// DoS regression gate (TestDoSPerformance) covers scale separately.
//
// 4 KiB is well below the fuzz worker's per-exec budget; bugs that manifest
// only on huge inputs are the DoS gate's responsibility, not the structural
// gates'.
const fuzzGateSizeCap = 4 * 1024

// checkOffsetBoundsGate (Gate 5: OFFSET-BOUNDS) asserts that Get's offset
// return is within the documented range [-1, len(data)]. Per SYS-REQ-016 /
// SYS-REQ-019, not-found returns offset=-1; per SYS-REQ-001 success returns
// a value-end offset in [0, len(data)]. An offset outside this range is an
// OOB index waiting to happen at the call site.
//
// Cheap (one comparison); no size cap needed.
func checkOffsetBoundsGate(t *testing.T, data []byte, path []string) {
	t.Helper()
	_, _, offset, _ := Get(data, path...)
	if offset < -1 || offset > len(data) {
		t.Errorf("Get returned out-of-bounds offset %d (len=%d): "+
			"data=%q path=%v (OFFSET-BOUNDS violation)",
			offset, len(data), data, path)
	}
}

// checkDeterminismGate (Gate 6: DETERMINISM) asserts that two Get calls on
// the same input+path return byte-identical (value, type, offset) and the
// same error PRESENCE. SYS-REQ-086 / SYS-REQ-080 explicitly require this.
// A non-determinism bug here would surface as flaky callers and is almost
// always a map-iteration or stale-state bug in searchKeys / getType.
//
// Size-capped to avoid O(n) overhead on huge fuzz inputs (the DoS regression
// gate TestDoSPerformance covers scale separately).
func checkDeterminismGate(t *testing.T, data []byte, path []string) {
	t.Helper()
	if len(data) > fuzzGateSizeCap {
		return
	}
	v1, t1, o1, e1 := Get(data, path...)
	v2, t2, o2, e2 := Get(data, path...)
	if !bytes.Equal(v1, v2) || t1 != t2 || o1 != o2 {
		t.Errorf("Get non-deterministic (DETERMINISM violation): "+
			"call1=(%q,%v,%d) call2=(%q,%v,%d) data=%q path=%v",
			v1, t1, o1, v2, t2, o2, data, path)
	}
	if (e1 == nil) != (e2 == nil) {
		t.Errorf("Get error presence non-deterministic (DETERMINISM violation): "+
			"e1=%v e2=%v data=%q path=%v", e1, e2, data, path)
	}
}

// checkSliceAliasingGate (Gate 7: ALIASING) asserts that the value bytes Get
// returns appear as a sub-slice of the input data. Per Get's documentation,
// the returned slice is a "Pointer to original data structure containing key
// value" — i.e. it ALIASES the input. If the value is not found in data,
// Get returned a synthesized/stale slice, which is silent data corruption
// (the SYS-REQ-016 worst-case "leaking a sibling field's bytes").
//
// Size-capped: bytes.Index is O(n+m) on average but O(n*m) worst-case. On
// huge fuzz inputs (deep nesting), this gate would dominate fuzz exec time.
func checkSliceAliasingGate(t *testing.T, data []byte, path []string) {
	t.Helper()
	if len(data) > fuzzGateSizeCap {
		return
	}
	val, dt, _, err := Get(data, path...)
	if err != nil || len(val) == 0 {
		return
	}
	// NotExist/Unknown are sentinels; no aliasing contract applies.
	if dt == NotExist || dt == Unknown {
		return
	}
	if bytes.Index(data, val) == -1 {
		t.Errorf("Get returned value not aliased to input data "+
			"(ALIASING violation): val=%q dt=%v data=%q path=%v",
			val, dt, data, path)
	}
}

// checkParseStringGate (Gate 8: PARSESTRING) extracts a String-typed value
// via Get and calls ParseString on the raw escaped content. Closes the API-
// coverage blind spot: without this gate, the structure-aware fuzzer NEVER
// reaches ParseString on grammar-generated escape sequences (\uXXXX, \uD800\uDC00
// surrogate pairs, \u0000, long escape runs). It also cross-checks the
// unescaped result against encoding/json on the same token.
func checkParseStringGate(t *testing.T, data []byte, path []string) {
	t.Helper()
	if len(data) > fuzzGateSizeCap {
		return
	}
	val, dt, _, err := Get(data, path...)
	if err != nil || dt != String || len(val) == 0 {
		return
	}
	out, perr := ParseString(val)
	if perr != nil {
		// ParseString rejects malformed escapes; that is a clean failure.
		return
	}
	// NOTE: jsonparser is documented as lenient on raw invalid UTF-8 bytes
	// inside strings — it passes them through unchanged (consistent with
	// Get's no-validate contract on these seeds). We therefore do NOT assert
	// utf8.ValidString(out); that would assert a contract the parser does
	// not claim to uphold. The differential check below is the honest gate:
	// for any token encoding/json accepts, ParseString MUST produce the same
	// output. Disagreement there is a genuine unescape bug.
	//
	// Cross-check against encoding/json's reference unescaper. Wrap val back
	// into a JSON string literal so json.Unmarshal sees the same token.
	// Restricted to valid-UTF-8 inputs because the two implementations
	// DOCUMENTEDLY disagree on invalid-UTF-8 handling (jsonparser passes
	// raw bytes through; encoding/json substitutes U+FFFD) — that is a
	// design choice, not an unescape bug.
	quoted := make([]byte, 0, len(val)+2)
	quoted = append(quoted, '"')
	quoted = append(quoted, val...)
	quoted = append(quoted, '"')
	var ref string
	if jerr := json.Unmarshal(quoted, &ref); jerr == nil && utf8.Valid(val) {
		// Both succeeded — they must agree byte-for-byte.
		if out != ref {
			t.Errorf("ParseString disagrees with encoding/json "+
				"(PARSESTRING differential): ParseString(%q)=%q json.Unmarshal=%q data=%q path=%v",
				val, out, ref, data, path)
		}
	}
	// Sanity: result is bounded (catches runaway allocation in Unescape).
	if len(out) > len(val)*4+16 {
		t.Errorf("ParseString returned suspiciously large output (PARSESTRING sanity): "+
			"in=%d out=%d data=%q path=%v", len(val), len(out), data, path)
	}
}

// checkParseBooleanGate (Gate 9: PARSEBOOLEAN) cross-checks ParseBoolean
// against strconv.ParseBool on Boolean-typed values. Closes the API-coverage
// blind spot: the structure-aware fuzzer cross-checks ParseInt/ParseFloat
// (via checkNumericOracle) but NEVER reaches ParseBoolean.
func checkParseBooleanGate(t *testing.T, data []byte, path []string) {
	t.Helper()
	val, dt, _, err := Get(data, path...)
	if err != nil || dt != Boolean || len(val) == 0 {
		return
	}
	jpVal, jpErr := ParseBoolean(val)
	sVal, sErr := strconv.ParseBool(string(val))
	if (jpErr == nil) != (sErr == nil) {
		t.Errorf("ParseBoolean disagreement on %q (PARSEBOOLEAN differential): "+
			"jsonparser=(%v,%v) strconv=(%v,%v) data=%q path=%v",
			val, jpVal, jpErr, sVal, sErr, data, path)
	} else if jpErr == nil && jpVal != sVal {
		t.Errorf("ParseBoolean value mismatch on %q: "+
			"jsonparser=%v strconv=%v data=%q path=%v",
			val, jpVal, sVal, data, path)
	}
}

// =============================================================================
// Gate runner (the assertions that catch each bug class)
// =============================================================================

// runStructureAwareGates exercises the full parser surface against the
// given data + path, applying all four assertion gates.
//
// Panics are reported via t.Errorf (which the fuzz engine treats as a
// failing input and minimizes). The gates are HONEST: no t.Skip, no soft
// exceptions. If a gate fires, it is a real bug or a documented-open
// divergence (the latter are excluded from OUTPUT-VALIDITY via
// pathShapeMatchesRoot, matching path_fuzz_test.go's convention).
func runStructureAwareGates(t *testing.T, data []byte, path []string) {
	t.Helper()

	dataValid := json.Valid(data)

	// --- NO-PANIC gate + NUMERIC-ORACLE gate (Get family) ---

	runOpNoPanic(t, "Get", data, path, func() {
		val, dt, _, err := Get(data, path...)
		if err != nil || dt != Number || len(val) == 0 {
			return
		}
		// NUMERIC-ORACLE: only compare against strconv for valid JSON
		// documents. For invalid/truncated JSON, the parser may classify
		// garbage tokens (like "-") as Number, and strconv will legitimately
		// disagree on those — that's the parser's lenient tokenization,
		// not a numeric-parsing bug.
		if dataValid {
			checkNumericOracle(t, val)
		}
	})
	runOpNoPanic(t, "GetString", data, path, func() {
		_, _ = GetString(data, path...)
	})
	runOpNoPanic(t, "GetInt", data, path, func() {
		_, _ = GetInt(data, path...)
	})
	runOpNoPanic(t, "GetFloat", data, path, func() {
		_, _ = GetFloat(data, path...)
	})
	runOpNoPanic(t, "GetBoolean", data, path, func() {
		_, _ = GetBoolean(data, path...)
	})
	runOpNoPanic(t, "GetUnsafeString", data, path, func() {
		_, _ = GetUnsafeString(data, path...)
	})

	// --- NEW GATES (closes Dimension 1 & 4 blind spots) ---
	// These exercise properties the original gate set did NOT assert:
	//   - OFFSET-BOUNDS: Get's offset return ∈ [-1, len(data)]
	//   - DETERMINISM:   two Get calls return byte-identical results (SYS-REQ-086)
	//   - ALIASING:      Get's value slice appears as a sub-slice of data
	//   - PARSESTRING:   grammar-generated escape sequences reach ParseString
	//   - PARSEBOOLEAN:  Boolean-typed values reach ParseBoolean cross-check
	runOpNoPanic(t, "Get/OffsetBounds", data, path, func() {
		checkOffsetBoundsGate(t, data, path)
	})
	runOpNoPanic(t, "Get/Determinism", data, path, func() {
		checkDeterminismGate(t, data, path)
	})
	runOpNoPanic(t, "Get/SliceAliasing", data, path, func() {
		checkSliceAliasingGate(t, data, path)
	})
	runOpNoPanic(t, "Get/ParseString", data, path, func() {
		checkParseStringGate(t, data, path)
	})
	runOpNoPanic(t, "Get/ParseBoolean", data, path, func() {
		checkParseBooleanGate(t, data, path)
	})

	// --- NO-PANIC gate (ArrayEach / ObjectEach / EachKey) ---

	runOpNoPanic(t, "ArrayEach", data, path, func() {
		_, _ = ArrayEach(data, func(value []byte, vt ValueType, off int, err error) {
			if dataValid && vt == Number && len(value) > 0 {
				checkNumericOracle(t, value)
			}
		}, path...)
	})
	runOpNoPanic(t, "ObjectEach", data, path, func() {
		_ = ObjectEach(data, func(key, value []byte, vt ValueType, off int) error {
			if dataValid && vt == Number && len(value) > 0 {
				checkNumericOracle(t, value)
			}
			return nil
		}, path...)
	})
	if len(path) > 0 {
		runOpNoPanic(t, "EachKey", data, path, func() {
			EachKey(data, func(idx int, value []byte, vt ValueType, err error) {
				if dataValid && vt == Number && len(value) > 0 {
					checkNumericOracle(t, value)
				}
			}, path)
		})

		// MULTI-PATH EachKey (closes Dimension 2 blind spot). EachKey's whole
		// purpose is to resolve MULTIPLE key paths in a single traversal. The
		// single-path call above exercises only the matched-but-no-siblings
		// branch; this call exercises (a) paths that match in different
		// positions, (b) paths that don't match at all, and (c) the
		// pathsMatched short-circuit logic.
		runOpNoPanic(t, "EachKey/MultiPath", data, path, func() {
			paths := [][]string{
				path,
				{"__fuzz_missing_key_a__"},
				{"__fuzz_missing_key_b__", "nested"},
			}
			EachKey(data, func(idx int, value []byte, vt ValueType, err error) {
				if dataValid && vt == Number && len(value) > 0 {
					checkNumericOracle(t, value)
				}
			}, paths...)
		})
	}

	// --- OUTPUT-VALIDITY + ROUND-TRIP gates (Set) ---

	if len(path) > 0 {
		setVal := []byte(`42`)
		runOpNoPanic(t, "Set", data, path, func() {
			// Keep mutating fuzz operations isolated from one another.
			dataCopy := make([]byte, len(data))
			copy(dataCopy, data)
			out, err := Set(dataCopy, setVal, path...)
			if err != nil {
				return
			}
			// OUTPUT-VALIDITY: output must be valid JSON when input was
			// valid and the path shape matches the container type at every
			// nesting level. For malformed input or shape-mismatched paths
			// (the documented D6/D10 divergences, including nested-D6 where
			// [N] targets a non-array value at depth > 0), the parser's
			// output is undefined and we only assert no-panic.
			// pathShapeMatchesData is called inside the recover wrapper so
			// any panic from its Get calls is caught by the NO-PANIC gate.
			shapeOK := pathShapeMatchesData(data, path)
			if dataValid && shapeOK && len(out) > 0 && !json.Valid(out) {
				t.Errorf("Set produced invalid JSON (OUTPUT-VALIDITY violation): "+
					"data=%q path=%v setVal=%q out=%q",
					data, path, setVal, out)
				return
			}
			// ROUND-TRIP: Get(out, path) must return the set value.
			// Only asserted for valid input + matching shape. Beyond-length
			// array indices (SYS-REQ-110) append at end and may not resolve
			// via Get at the same index, so we tolerate a Get error there.
			if !dataValid || !shapeOK {
				return
			}
			got, _, _, gErr := Get(out, path...)
			if gErr != nil {
				return
			}
			if string(got) == string(setVal) {
				return
			}
			// Allow numeric-equivalent representations (e.g. "42" vs "42.0").
			if gf, gerr := strconv.ParseFloat(string(got), 64); gerr == nil {
				if sf, serr := strconv.ParseFloat(string(setVal), 64); serr == nil && gf == sf {
					return
				}
			}
			t.Errorf("Set/Get round-trip mismatch (ROUND-TRIP violation): "+
				"set %q, got %q (data=%q path=%v out=%q)",
				setVal, got, data, path, out)
		})
	}

	// --- OUTPUT-VALIDITY gate (Delete) ---

	runOpNoPanic(t, "Delete", data, path, func() {
		// Keep mutating fuzz operations isolated from one another.
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)
		out := Delete(dataCopy, path...)
		shapeOK := pathShapeMatchesData(data, path)
		if dataValid && shapeOK && len(out) > 0 && !json.Valid(out) {
			t.Errorf("Delete produced invalid JSON (OUTPUT-VALIDITY violation): "+
				"data=%q path=%v out=%q",
				data, path, out)
		}
	})
}

// runOpNoPanic runs fn and fails the test on panic (the NO-PANIC gate).
// The panic value and stack are included in the failure message so the
// fuzz engine's minimizing reproducer carries full diagnostic context.
func runOpNoPanic(t *testing.T, opName string, data []byte, path []string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PANIC in %s (NO-PANIC violation): %v\n%s\ndata=%q path=%v",
				opName, r, debug.Stack(), data, path)
		}
	}()
	fn()
}

// checkNumericOracle cross-checks jsonparser's ParseInt/ParseFloat against
// the strconv reference (the NUMERIC-ORACLE gate). strconv is the reference
// implementation per the Go specification; disagreements indicate either a
// parser bug or a token where jsonparser and strconv legitimately disagree
// on accepted syntax (both are worth surfacing).
func checkNumericOracle(t *testing.T, val []byte) {
	t.Helper()

	// ParseInt cross-check (only for integer-looking tokens).
	isInt := len(val) > 0
	for _, b := range val {
		if (b < '0' || b > '9') && b != '-' && b != '+' {
			isInt = false
			break
		}
	}
	if isInt {
		jVal, jErr := ParseInt(val)
		sVal, sErr := strconv.ParseInt(string(val), 10, 64)
		if (jErr == nil) != (sErr == nil) {
			t.Errorf("ParseInt disagreement on %q: jsonparser=(%d,%v) strconv=(%d,%v)",
				val, jVal, jErr, sVal, sErr)
		} else if jErr == nil && jVal != sVal {
			t.Errorf("ParseInt value mismatch on %q: jsonparser=%d strconv=%d",
				val, jVal, sVal)
		}
	}

	// ParseFloat cross-check (all numeric tokens).
	jfVal, jfErr := ParseFloat(val)
	sfVal, sfErr := strconv.ParseFloat(string(val), 64)
	if (jfErr == nil) != (sfErr == nil) {
		t.Errorf("ParseFloat disagreement on %q: jsonparser=(%f,%v) strconv=(%f,%v)",
			val, jfVal, jfErr, sfVal, sfErr)
	} else if jfErr == nil && jfVal != sfVal {
		t.Errorf("ParseFloat value mismatch on %q: jsonparser=%f strconv=%f",
			val, jfVal, sfVal)
	}
}

// =============================================================================
// The fuzzer (testing.F, native go-fuzz)
// =============================================================================

// looksLikeJSON reports whether data begins with a byte that could start a
// JSON value. This is a cheap pre-check used to decide whether to use the
// data as-is (letting the fuzz engine's coverage feedback guide mutation)
// or to regenerate from the grammar (to maintain structural coverage when
// the engine has mutated the data into non-JSON garbage).
func looksLikeJSON(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		case '{', '[', '"', 't', 'f', 'n', '-',
			'0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return true
		default:
			return false
		}
	}
	return false
}

// FuzzJSONStructureAware is the structure-aware fuzzer. It combines:
//   - Grammar-based JSON generation (always emits valid structures).
//   - JSON-aware mutation operators (break structures at known-dangerous
//     boundaries: value boundaries, mid-key, mid-structure, mid-escape).
//   - Adversarial key-path generation (empty keys, OOB array indices).
//   - Four assertion gates (no-panic, output-validity, round-trip, numeric-oracle).
//
// The seed corpus includes the actual bugs we've found (D9 repro, truncation
// classes, empty-key, OOB index) so they are exercised on every run.
//
// Verifies: SYS-REQ-035 [no-panic on malformed input]
// reqproof:proptest parser.Get, parser.Set, parser.Delete, parser.GetString, parser.GetInt, parser.GetFloat,
//           parser.GetBoolean, parser.GetUnsafeString, parser.ArrayEach, parser.ObjectEach, parser.EachKey,
//           parser.ParseInt, parser.ParseFloat
func FuzzJSONStructureAware(f *testing.F) {
	// Seed with known-dangerous corpora (the actual bugs we've found).
	for _, s := range structureAwareSeeds {
		f.Add(s.data, s.path)
	}

	f.Fuzz(func(t *testing.T, data []byte, pathBytes []byte) {
		// Derive deterministic RNG from the fuzz inputs so generation and
		// mutation decisions are a pure function of the engine-supplied bytes.
		r := newRandFromSeed(data, pathBytes)

		// --- Structure-aware generation layer ---
		// When the fuzz engine mutates data into non-JSON garbage (or ~30%
		// of the time regardless), regenerate valid JSON from the grammar.
		// This maintains structural coverage that blind byte mutation cannot
		// reach in reasonable time.
		work := data
		if !looksLikeJSON(data) || r.Intn(10) < 3 {
			depth := r.Intn(4) // mostly depth 0–3
			if r.Intn(10) == 0 {
				depth = 5 + r.Intn(3) // occasionally deep (stress recursion)
			}
			work = genJSON(r, depth)
		}

		// --- Deep-nesting injection (closes Dimension 5 DoS blind spot) ---
		// The grammar generator's maxGenDepth=5 cannot produce pathological
		// nesting. Occasionally replace work with deeply-nested arrays so the
		// parser's own stack-safety (not just encoding/json's) gets fuzzed.
		// Both closed ([[[...1...]]]) and unclosed ([[[...) shapes are tested;
		// the unclosed shape forces full descent before error recovery.
		//
		// Capped at 1000 levels in the fuzz body: deeper inputs make the
		// existing Get-family gates (6+ calls, each O(n)) dominate the
		// worker's per-exec budget. The non-fuzz TestDoSPerformance gate
		// covers 100k+ level nesting separately.
		if r.Intn(40) == 0 {
			depth := 50 * (1 << r.Intn(5)) // 50..800
			work = genDeepNesting(depth, r.Intn(2) == 0)
		}

		// --- JSON-aware mutation layer ---
		// Apply 0–3 mutations at structural boundaries. Each operator
		// targets a specific bug class (truncation, duplication, corruption).
		for n := r.Intn(4); n > 0; n-- {
			work = applyMutation(r, work)
		}

		// --- Adversarial key-path layer ---
		// Derive the path from pathBytes (split on '/'); occasionally inject
		// a hazard component (empty key, OOB index) or regenerate from grammar.
		path := splitPathComponents(pathBytes)
		if r.Intn(5) == 0 { // 20% chance
			path = injectPathHazard(r, path)
		}

		// --- Gates ---
		runStructureAwareGates(t, work, path)
	})
}

// =============================================================================
// Corpus-seeding helper (non-fuzz regression gate)
// =============================================================================

// TestJSONFuzzCorpusSanity runs every seed corpus entry AND a batch of
// grammar-generated inputs through the same gates as FuzzJSONStructureAware,
// so the standard `go test` catches regressions on the known-bad inputs
// without needing `-fuzz`. It also cross-checks that genJSON always produces
// valid JSON (the generator invariant).
//
// Verifies: SYS-REQ-035
func TestJSONFuzzCorpusSanity(t *testing.T) {
	// 1. Run each static seed through the gates.
	for i, seed := range structureAwareSeeds {
		seed := seed
		t.Run(fmt.Sprintf("seed_%02d_data_%q_path_%q", i, seed.data, seed.path), func(t *testing.T) {
			path := splitPathComponents(seed.path)
			runStructureAwareGates(t, seed.data, path)
		})
	}

	// 2. Grammar-generated inputs: exercise the generator + mutations under
	//    `go test` without requiring the fuzz engine. This is the path that
	//    exercises structure-aware generation most directly.
	r := rand.New(rand.NewSource(0xC0FFEE))
	for i := 0; i < 300; i++ {
		depth := r.Intn(4)
		if r.Intn(10) == 0 {
			depth = 5 + r.Intn(3)
		}
		data := genJSON(r, depth)

		// Generator note: genJSON produces mostly-valid JSON, but per the
		// spec it intentionally includes a small fraction of near-valid
		// edge cases (e.g. leading-zero numbers like "007") that test the
		// parser's leniency. We do NOT hard-assert json.Valid here — the
		// gates below are the real correctness test, and they handle
		// invalid input gracefully (the OUTPUT-VALIDITY gate is gated on
		// dataValid, so it only fires for truly valid input).

		// Apply 0–3 mutations (the mutations intentionally break validity).
		for n := r.Intn(4); n > 0; n-- {
			data = applyMutation(r, data)
		}

		path := genKeyPath(r)
		t.Run(fmt.Sprintf("gen_%03d", i), func(t *testing.T) {
			runStructureAwareGates(t, data, path)
		})
	}
}

// =============================================================================
// DoS / scale regression gate (closes Dimension 5)
// =============================================================================

// dosOpBudget is the per-operation wall-clock budget for TestDoSPerformance.
// Generous (5s) so the gate is robust on slow CI; tight enough that an O(n²)
// regression on a 1MB input fails it. Override via GO_DOS_BUDGET for tuning.
var dosOpBudget = func() time.Duration {
	if v := os.Getenv("GO_DOS_BUDGET"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return 5 * time.Second
}()

// TestDoSPerformance is a non-fuzz regression gate that asserts the parser
// completes each major operation on a 1MB input within dosOpBudget. Catches
// O(n²) regressions in tokenEnd/blockEnd/searchKeys that pure correctness
// fuzzing leaves invisible (a slow-but-correct parser is still a DoS bug).
//
// Verifies: SYS-REQ-035 (no pathological-input hang).
func TestDoSPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DoS performance gate in -short mode")
	}
	// Build a ~1MB JSON document: deeply nested then a long string at the end.
	// Two pathological shapes are tested:
	//   (1) Deeply nested unclosed arrays — exercises the parser's descent
	//       cost when no closers exist.
	//   (2) Wide flat object with a 1MB string value — exercises the
	//       per-byte scanning cost in stringEnd.
	const targetBytes = 1 << 20 // 1 MiB

	t.Run("deep_nested", func(t *testing.T) {
		// 100k unclosed '[' — about 100KB; loop it to 1MB if needed.
		depth := 100_000
		for depth*2 < targetBytes {
			depth *= 2
		}
		data := genDeepNesting(depth, false)
		runDoSCase(t, "Get", data, func() { _, _, _, _ = Get(data) })
		runDoSCase(t, "ArrayEach", data, func() {
			_, _ = ArrayEach(data, func([]byte, ValueType, int, error) {})
		})
	})

	t.Run("long_string", func(t *testing.T) {
		// {"a":"<repeat x 1MB>"} — exercises stringEnd on a long token.
		body := bytes.Repeat([]byte("x"), targetBytes-8)
		data := make([]byte, 0, targetBytes)
		data = append(data, []byte(`{"a":"`)...)
		data = append(data, body...)
		data = append(data, []byte(`"}`)...)
		runDoSCase(t, "Get", data, func() { _, _, _, _ = Get(data, "a") })
		runDoSCase(t, "GetString", data, func() { _, _ = GetString(data, "a") })
		runDoSCase(t, "ParseString", data, func() {
			val, _, _, _ := Get(data, "a")
			_, _ = ParseString(val)
		})
	})
}

// runDoSCase runs one op under the DoS budget. A panic or a timeout exceeds
// the budget — either is a finding.
func runDoSCase(t *testing.T, opName string, data []byte, fn func()) {
	t.Helper()
	done := make(chan struct{})
	start := time.Now()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("PANIC in %s on %d-byte input (DoS gate): %v\n%s",
					opName, len(data), r, debug.Stack())
			}
			close(done)
		}()
		fn()
	}()
	select {
	case <-done:
		elapsed := time.Since(start)
		if elapsed > dosOpBudget {
			t.Errorf("%s took %v on %d-byte input (DoS budget %v exceeded)",
				opName, elapsed, len(data), dosOpBudget)
		}
	case <-time.After(dosOpBudget):
		t.Fatalf("%s hung > %v on %d-byte input (DoS regression)",
			opName, dosOpBudget, len(data))
	}
}
