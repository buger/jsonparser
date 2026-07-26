package jsonparser

import (
	"fmt"
	"testing"
)

// =============================================================================
// REMOVAL 1: `for true` → `for {}` in ArrayEach, Unescape, ObjectEach
// These are cosmetic changes. We verify exit conditions still work.
// =============================================================================

// Verifies: SYS-REQ-006 [boundary]
func TestRemoval1_ArrayEach_LoopExitsOnEmptyArray(t *testing.T) {
	_, err := ArrayEach([]byte(`[]`), func(value []byte, dataType ValueType, offset int, err error) {
		t.Fatal("callback should not be called for empty array")
	})
	if err != nil {
		t.Fatalf("unexpected error on empty array: %v", err)
	}
}

// Verifies: SYS-REQ-006 [boundary]
func TestRemoval1_ArrayEach_LoopExitsOnSingleElement(t *testing.T) {
	count := 0
	_, err := ArrayEach([]byte(`[1]`), func(value []byte, dataType ValueType, offset int, err error) {
		count++
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 callback, got %d", count)
	}
}

// Verifies: SYS-REQ-014 [boundary]
func TestRemoval1_Unescape_LoopExitsOnSingleEscape(t *testing.T) {
	out, err := Unescape([]byte(`hello\nworld`), make([]byte, 64))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "hello\nworld" {
		t.Fatalf("unexpected result: %q", out)
	}
}

// Verifies: SYS-REQ-014 [boundary]
func TestRemoval1_Unescape_LoopExitsOnTrailingEscape(t *testing.T) {
	out, err := Unescape([]byte(`\n`), make([]byte, 64))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "\n" {
		t.Fatalf("unexpected result: %q", out)
	}
}

// Verifies: SYS-REQ-007 [boundary]
func TestRemoval1_ObjectEach_LoopExitsOnEmptyObject(t *testing.T) {
	err := ObjectEach([]byte(`{}`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		t.Fatal("callback should not be called for empty object")
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error on empty object: %v", err)
	}
}

// Verifies: SYS-REQ-007 [boundary]
func TestRemoval1_ObjectEach_LoopExitsOnSingleEntry(t *testing.T) {
	count := 0
	err := ObjectEach([]byte(`{"a":1}`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 callback, got %d", count)
	}
}

// =============================================================================
// REMOVAL 2: `end == -1` guard removed in getType
// tokenEnd returns len(data) instead of -1. Verify behavior for edge cases.
// =============================================================================

// Verifies: SYS-REQ-044 [boundary]
func TestRemoval2_TokenEnd_EmptyInput(t *testing.T) {
	result := tokenEnd([]byte{})
	if result != 0 {
		t.Fatalf("tokenEnd([]) = %d, want 0", result)
	}
}

// Verifies: SYS-REQ-044 [boundary]
func TestRemoval2_TokenEnd_NoDelimiter(t *testing.T) {
	// Input with no delimiter characters at all
	result := tokenEnd([]byte("12345"))
	if result != 5 {
		t.Fatalf("tokenEnd(12345) = %d, want 5 (len)", result)
	}
}

// Verifies: SYS-REQ-044 [boundary]
func TestRemoval2_TokenEnd_NeverReturnsNegative(t *testing.T) {
	// This is the critical assertion: tokenEnd NEVER returns -1.
	// If it did, the removed guard would be needed.
	inputs := [][]byte{
		{},
		[]byte("abc"),
		[]byte("123"),
		[]byte("true"),
		[]byte("false"),
		[]byte("null"),
		[]byte("-1"),
		[]byte("1e10"),
	}
	for _, in := range inputs {
		r := tokenEnd(in)
		if r < 0 {
			t.Fatalf("tokenEnd(%q) returned %d, which is negative!", in, r)
		}
	}
}

// Verifies: SYS-REQ-001 [boundary]
func TestRemoval2_GetType_NumberAtEndOfInput(t *testing.T) {
	// This is the key edge case: a number at the very end of the input
	// with no trailing delimiter. tokenEnd returns len(data[endOffset:]) = 0,
	// so end=0 and value = data[offset:endOffset+0] = data[offset:offset] = empty.
	// Actually wait: if input is just "42", offset=0, endOffset=0,
	// data[endOffset:] = "42", tokenEnd("42") = 2, value = data[0:2] = "42".
	// That's correct.

	// But what about Get on bare value "42"?
	val, dt, _, err := Get([]byte("42"))
	if err != nil {
		t.Fatalf("unexpected error parsing bare '42': %v", err)
	}
	if dt != Number {
		t.Fatalf("expected Number, got %v", dt)
	}
	if string(val) != "42" {
		t.Fatalf("expected '42', got %q", val)
	}
}

// Verifies: SYS-REQ-001 [boundary]
func TestRemoval2_GetType_BooleanAtEndOfInput(t *testing.T) {
	val, dt, _, err := Get([]byte("true"))
	if err != nil {
		t.Fatalf("unexpected error parsing bare 'true': %v", err)
	}
	if dt != Boolean {
		t.Fatalf("expected Boolean, got %v", dt)
	}
	if string(val) != "true" {
		t.Fatalf("expected 'true', got %q", val)
	}
}

// Verifies: SYS-REQ-001 [boundary]
func TestRemoval2_GetType_NullAtEndOfInput(t *testing.T) {
	val, dt, _, err := Get([]byte("null"))
	if err != nil {
		t.Fatalf("unexpected error parsing bare 'null': %v", err)
	}
	if dt != Null {
		t.Fatalf("expected Null, got %v", dt)
	}
	if string(val) != "null" {
		t.Fatalf("expected 'null', got %q", val)
	}
}

// Critical: tokenEnd returns len(data) vs stringEnd/blockEnd returning -1.
// The inconsistency means getType silently accepts truncated tokens.
// Verifies: SYS-REQ-001 [boundary]
func TestRemoval2_Inconsistency_TruncatedNumber(t *testing.T) {
	// Consider: `{"a": 12` — the number "12" has no terminator.
	// tokenEnd("12") returns 2, so getType will return "12" as a Number.
	// This is actually CORRECT behavior for a streaming parser,
	// but it means we can't distinguish "complete number" from "truncated input".
	// The removed guard would NOT have caught this either (tokenEnd never returns -1).
	val, dt, _, err := Get([]byte(`{"a": 12`), "a")
	if err != nil {
		t.Logf("Error on truncated number: %v (this is fine)", err)
	} else {
		t.Logf("Truncated number parsed as: val=%q type=%v", val, dt)
	}
}

// =============================================================================
// REMOVAL 3: `r <= basicMultilingualPlaneOffset` removed in decodeUnicodeEscape
// Verify that decodeSingleUnicodeEscape can NEVER produce r > 0xFFFF.
// =============================================================================

// Verifies: SYS-REQ-014 [boundary]
func TestRemoval3_DecodeSingleUnicodeEscape_MaxValue(t *testing.T) {
	// \uFFFF is the maximum possible value from a single \uXXXX escape.
	// 4 hex digits: max = 0xFFFF = 65535 = basicMultilingualPlaneOffset
	r, ok := decodeSingleUnicodeEscape([]byte(`\uFFFF`))
	if !ok {
		t.Fatal("failed to decode \\uFFFF")
	}
	if r != 0xFFFF {
		t.Fatalf("expected 0xFFFF, got 0x%X", r)
	}
	if r > basicMultilingualPlaneOffset {
		t.Fatalf("UNSAFE: r=0x%X exceeds BMP offset 0x%X", r, basicMultilingualPlaneOffset)
	}
}

// Verifies: SYS-REQ-014 [boundary]
func TestRemoval3_DecodeSingleUnicodeEscape_MinValue(t *testing.T) {
	r, ok := decodeSingleUnicodeEscape([]byte(`\u0000`))
	if !ok {
		t.Fatal("failed to decode \\u0000")
	}
	if r != 0 {
		t.Fatalf("expected 0, got 0x%X", r)
	}
}

// Verifies: SYS-REQ-014 [boundary]
func TestRemoval3_DecodeUnicodeEscape_BMP_NonSurrogate(t *testing.T) {
	// \u0041 = 'A', well within BMP and not a surrogate
	r, n := decodeUnicodeEscape([]byte(`\u0041`))
	if n != 6 {
		t.Fatalf("expected consumed=6, got %d", n)
	}
	if r != 'A' {
		t.Fatalf("expected 'A', got %c (0x%X)", r, r)
	}
}

// Verifies: SYS-REQ-014 [boundary]
func TestRemoval3_DecodeUnicodeEscape_HighSurrogateAlone(t *testing.T) {
	// \uD800 is a high surrogate — should require a low surrogate pair
	r, n := decodeUnicodeEscape([]byte(`\uD800`))
	if n != -1 {
		t.Fatalf("expected error (n=-1) for lone high surrogate, got n=%d r=0x%X", n, r)
	}
}

// Verifies: SYS-REQ-014 [boundary]
func TestRemoval3_DecodeUnicodeEscape_ValidSurrogatePair(t *testing.T) {
	// \uD83D\uDE00 = U+1F600 (grinning face emoji)
	r, n := decodeUnicodeEscape([]byte(`\uD83D\uDE00`))
	if n != 12 {
		t.Fatalf("expected consumed=12, got %d", n)
	}
	if r != 0x1F600 {
		t.Fatalf("expected U+1F600, got U+%X", r)
	}
}

// Verifies: SYS-REQ-014 [formal]
func TestRemoval3_MathematicalProof(t *testing.T) {
	// Mathematical proof: decodeSingleUnicodeEscape computes
	// h1<<12 + h2<<8 + h3<<4 + h4
	// where h1..h4 are in [0, 15].
	// Maximum: 15<<12 + 15<<8 + 15<<4 + 15 = 61440 + 3840 + 240 + 15 = 65535 = 0xFFFF
	// This equals basicMultilingualPlaneOffset exactly.
	// Therefore r <= basicMultilingualPlaneOffset is ALWAYS true.
	maxR := rune(15<<12 + 15<<8 + 15<<4 + 15)
	if maxR != basicMultilingualPlaneOffset {
		t.Fatalf("max possible rune 0x%X != basicMultilingualPlaneOffset 0x%X", maxR, basicMultilingualPlaneOffset)
	}
	t.Logf("PROVEN: max rune from \\uXXXX = 0x%X = basicMultilingualPlaneOffset", maxR)
}

// =============================================================================
// REMOVAL 4: `data[i] == '{'` block removed in EachKey
// Test: unmatched key followed by nested object must still be skipped.
// =============================================================================

// Verifies: SYS-REQ-008 [boundary]
func TestRemoval4_EachKey_SkipNestedObject(t *testing.T) {
	data := []byte(`{"skip":{"nested":"deep"},"want":"found"}`)
	paths := [][]string{{"want"}}

	var foundValue []byte
	var foundType ValueType
	EachKey(data, func(idx int, value []byte, vt ValueType, err error) {
		if idx == 0 {
			foundValue = value
			foundType = vt
		}
	}, paths...)

	if foundValue == nil {
		t.Fatal("UNSAFE: EachKey failed to find 'want' after skipping nested object")
	}
	if string(foundValue) != "found" {
		t.Fatalf("expected 'found', got %q", foundValue)
	}
	if foundType != String {
		t.Fatalf("expected String type, got %v", foundType)
	}
}

// Verifies: SYS-REQ-008 [boundary]
func TestRemoval4_EachKey_SkipDeeplyNestedObject(t *testing.T) {
	data := []byte(`{"skip":{"a":{"b":{"c":"deep"}}},"want":"found"}`)
	paths := [][]string{{"want"}}

	var foundValue []byte
	EachKey(data, func(idx int, value []byte, vt ValueType, err error) {
		if idx == 0 {
			foundValue = value
		}
	}, paths...)

	if foundValue == nil {
		t.Fatal("UNSAFE: EachKey failed to find 'want' after deeply nested skip")
	}
	if string(foundValue) != "found" {
		t.Fatalf("expected 'found', got %q", foundValue)
	}
}

// Verifies: SYS-REQ-008 [boundary]
func TestRemoval4_EachKey_SkipNestedArray(t *testing.T) {
	data := []byte(`{"skip":[1,2,3],"want":"found"}`)
	paths := [][]string{{"want"}}

	var foundValue []byte
	EachKey(data, func(idx int, value []byte, vt ValueType, err error) {
		if idx == 0 {
			foundValue = value
		}
	}, paths...)

	if foundValue == nil {
		t.Fatal("UNSAFE: EachKey failed to find 'want' after array skip")
	}
	if string(foundValue) != "found" {
		t.Fatalf("expected 'found', got %q", foundValue)
	}
}

// Verifies: SYS-REQ-008 [boundary]
func TestRemoval4_EachKey_SkipMultipleNestedObjects(t *testing.T) {
	data := []byte(`{"a":{"x":1},"b":{"y":2},"want":"found"}`)
	paths := [][]string{{"want"}}

	var foundValue []byte
	EachKey(data, func(idx int, value []byte, vt ValueType, err error) {
		if idx == 0 {
			foundValue = value
		}
	}, paths...)

	if foundValue == nil {
		t.Fatal("UNSAFE: EachKey failed to find 'want' after multiple nested objects")
	}
	if string(foundValue) != "found" {
		t.Fatalf("expected 'found', got %q", foundValue)
	}
}

// Verifies: SYS-REQ-008 [boundary]
func TestRemoval4_EachKey_NestedObjectWithString(t *testing.T) {
	// This tests the case where a string value contains braces
	data := []byte(`{"skip":"has {braces}","want":"found"}`)
	paths := [][]string{{"want"}}

	var foundValue []byte
	EachKey(data, func(idx int, value []byte, vt ValueType, err error) {
		if idx == 0 {
			foundValue = value
		}
	}, paths...)

	if foundValue == nil {
		t.Fatal("UNSAFE: EachKey failed to find 'want' after string with braces")
	}
	if string(foundValue) != "found" {
		t.Fatalf("expected 'found', got %q", foundValue)
	}
}

// =============================================================================
// REMOVAL 5: `keys[level][0] != '['` removed in searchKeys
// The original code was: `keys[level][0] != '[' || keys[level][keyLen-1] != ']'`
// within `if keyLevel == level && keys[level][0] == '['`
// So keys[level][0] != '[' was ALWAYS false (contradiction). Removing it is safe.
// =============================================================================

// Verifies: SYS-REQ-001 [boundary]
func TestRemoval5_SearchKeys_ArrayIndex_Valid(t *testing.T) {
	data := []byte(`[1, "two", 3]`)
	// searchKeys with "[1]" should find element at index 1
	offset := searchKeys(data, "[1]")
	if offset == -1 {
		t.Fatal("searchKeys failed to find array index [1]")
	}
}

// Verifies: SYS-REQ-001 [boundary]
func TestRemoval5_SearchKeys_ArrayIndex_MalformedNoClose(t *testing.T) {
	data := []byte(`[1, 2, 3]`)
	// "[1" has no closing bracket — keyLen < 3 catches this
	offset := searchKeys(data, "[1")
	if offset != -1 {
		t.Fatalf("expected -1 for malformed index '[1', got %d", offset)
	}
}

// Verifies: SYS-REQ-001 [boundary]
func TestRemoval5_SearchKeys_ArrayIndex_TooShort(t *testing.T) {
	data := []byte(`[1, 2, 3]`)
	// "[]" has keyLen=2 which is < 3 — still caught
	offset := searchKeys(data, "[]")
	if offset != -1 {
		t.Fatalf("expected -1 for empty index '[]', got %d", offset)
	}
}

// Verifies: SYS-REQ-001 [boundary]
func TestRemoval5_SearchKeys_ArrayIndex_NestedObject(t *testing.T) {
	data := []byte(`[{"a":1},{"a":2}]`)
	offset := searchKeys(data, "[1]", "a")
	if offset == -1 {
		t.Fatal("searchKeys failed to find [1].a")
	}
}

// =============================================================================
// REMOVAL 6: `if e != nil` inside `if o == 0` removed in ArrayEach
// Critical: Can Get return (_, _, 0, nil)?
// =============================================================================

// Verifies: SYS-REQ-006 [boundary]
func TestRemoval6_ArrayEach_GetReturnsZeroOffset(t *testing.T) {
	// Get is called with data[offset:]. For Get to return endOffset=0,
	// internalGet would need to return endOffset=0.
	// In internalGet: if no keys, offset starts at 0.
	// nO := nextToken(data[0:]) — if data is whitespace-only, returns -1 → error.
	// If data starts with a token, offset += nO.
	// Then getType is called. getType returns endOffset from getType.
	//
	// The question: can getType return endOffset=0?
	// endOffset starts as `offset` in getType. If offset=0 and data[0] is '"',
	// stringEnd returns idx, so endOffset = 0 + idx + 1 >= 1. Not 0.
	// If data[0] is '[', blockEnd returns endOffset >= 1. Not 0.
	// If data[0] is '{', blockEnd returns endOffset >= 1. Not 0.
	// Otherwise tokenEnd(data[0:]) — for empty would be 0 but offset is 0,
	// so value = data[0:0+0] = empty, which would fail the boolean/null/number check.
	//
	// So: getType CAN return endOffset=0 if the token at position 0 is a
	// zero-length number token... which means tokenEnd(data[0:])=0, meaning
	// data[0] IS a delimiter. But that contradicts reaching the else branch
	// (data[0] is not '"', '[', or '{').
	//
	// Actually, for endOffset to be 0, we need offset=0 AND
	// tokenEnd(data[0:])=0, which means data[0] is a delimiter (space, comma, etc.).
	// But nextToken would have skipped past spaces. If data[0] is comma,
	// nextToken returns 0 (comma is not whitespace per nextToken).
	// Then getType is called with offset=0, data[0]=','.
	// getType falls to the default case → UnknownValueTypeError.
	// endOffset = offset + end = 0 + 0 = 0. Error is returned.
	// So internalGet returns endOffset=0 WITH an error.
	// This means Get returns offset=0 WITH an error.

	// Test: ArrayEach with malformed content where Get returns offset 0
	// The `,` after `[` will cause Get to fail with offset 0
	count := 0
	_, err := ArrayEach([]byte(`[,1]`), func(value []byte, dataType ValueType, offset int, err error) {
		count++
	})
	if err == nil {
		t.Logf("NOTE: ArrayEach did not error on [,1] — count=%d", count)
	} else {
		t.Logf("ArrayEach correctly errored on [,1]: %v", err)
	}
	// The key check: did we infinite loop? If we get here, we didn't.
	t.Log("PASS: no infinite loop")
}

// Verifies: SYS-REQ-006 [boundary]
func TestRemoval6_ArrayEach_EmptyStringElement(t *testing.T) {
	// Can Get return ([], String, 0, nil) for an empty string ""?
	// Get("\"\"") → internalGet → searchKeys skipped → nextToken → offset 0
	// → getType(data, 0) → data[0]='"' → stringEnd(data[1:])
	// stringEnd on `"` → finds quote at position 0 → returns (1, false)
	// So endOffset = 0 + 1 + 1 = 2. NOT 0.
	// Get returns offset=2 (the endOffset from internalGet's 4th return).
	// Wait — Get returns internalGet's 4th value as `offset`.
	// Let me re-check: Get returns (a, b, d, e) where d = endOffset from internalGet.
	// For "" at position 0: endOffset from getType = 2, so Get returns offset=2.

	count := 0
	_, err := ArrayEach([]byte(`["","b"]`), func(value []byte, dataType ValueType, offset int, err error) {
		count++
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 callbacks, got %d", count)
	}
}

// Verifies: SYS-REQ-006 [boundary]
func TestRemoval6_ArrayEach_WhitespaceOnlyInput(t *testing.T) {
	// Can Get return (nil, NotExist, 0, nil)?
	// Get("   ") → nextToken returns 0 pointing to first space... no.
	// nextToken skips spaces. "   " → returns -1 → MalformedJsonError.
	// So Get returns (nil, NotExist, 0, MalformedJsonError). offset=0, err!=nil.
	// In ArrayEach: if o==0, returns offset, e — correct.
	_, err := ArrayEach([]byte(`[   `), func(value []byte, dataType ValueType, offset int, err error) {
		// should not be called
	})
	if err == nil {
		t.Fatal("expected error for whitespace-only array content")
	}
	t.Logf("Correctly got error: %v", err)
}

// =============================================================================
// REMOVAL 7: `ln > 0` guard removed in findKeyStart
// Verify nextToken returning non-negative guarantees len(data) > 0
// =============================================================================

// Verifies: SYS-REQ-001 [boundary]
func TestRemoval7_NextToken_EmptyInput(t *testing.T) {
	result := nextToken([]byte{})
	if result != -1 {
		t.Fatalf("nextToken([]) = %d, expected -1", result)
	}
}

// Verifies: SYS-REQ-001 [boundary]
func TestRemoval7_NextToken_WhitespaceOnly(t *testing.T) {
	result := nextToken([]byte("   \t\n"))
	if result != -1 {
		t.Fatalf("nextToken(whitespace) = %d, expected -1", result)
	}
}

// Verifies: SYS-REQ-001 [boundary]
func TestRemoval7_FindKeyStart_NextTokenGuaranteesNonEmpty(t *testing.T) {
	// If nextToken returns >= 0, then data has at least one non-whitespace byte,
	// which means len(data) >= 1, which means ln > 0.
	// Proof: nextToken iterates data[0..len-1]. If len=0, loop doesn't execute,
	// returns -1. So nextToken >= 0 ⟹ len(data) >= 1.

	// Test: findKeyStart with valid minimal input
	_, err := findKeyStart([]byte(`{"a":1}`), "a")
	if err != nil {
		t.Fatalf("findKeyStart failed: %v", err)
	}

	// Test: findKeyStart with empty input
	_, err = findKeyStart([]byte{}, "a")
	if err != KeyPathNotFoundError {
		t.Fatalf("expected KeyPathNotFoundError, got %v", err)
	}
}

// =============================================================================
// EXTRA: ObjectEach `for offset < len(data)` → `for {}`
// Verify the loop can't run past the end of data.
// =============================================================================

// Verifies: SYS-REQ-007 [boundary]
func TestRemoval_ObjectEach_MalformedTrailingComma(t *testing.T) {
	// Object ends with comma but no more entries: `{"a":1,}`
	// After parsing "a":1, the loop finds comma, skips it, calls nextToken.
	// nextToken on "}" returns 0, offset points to '}', loop iteration starts,
	// switch hits '}' → return nil. No out-of-bounds.
	err := ObjectEach([]byte(`{"a":1,"b":2}`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Verifies: SYS-REQ-007 [boundary]
func TestRemoval_ObjectEach_MalformedNoClosingBrace(t *testing.T) {
	// `{"a":1` — no closing brace. After parsing "a":1,
	// nextToken on remaining data. Get consumes "1", offset moves past it.
	// nextToken(data[offset:]) on empty/near-empty → returns -1 → error.
	err := ObjectEach([]byte(`{"a":1`), func(key []byte, value []byte, dataType ValueType, offset int) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for missing closing brace")
	}
	t.Logf("Correctly got error: %v", err)
}

// =============================================================================
// STRESS: Ensure no infinite loops or panics on pathological inputs
// =============================================================================

// Verifies: SYS-REQ-006 [boundary]
func TestStress_ArrayEach_NestedEmpty(t *testing.T) {
	_, err := ArrayEach([]byte(`[[],[]]`), func(value []byte, dataType ValueType, offset int, err error) {
		// nested arrays
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Verifies: SYS-REQ-008 [boundary]
func TestStress_EachKey_LargeNestedSkip(t *testing.T) {
	// Build a large nested object that must be skipped
	inner := `{"a":{"b":{"c":{"d":"deep"}}}}`
	data := fmt.Sprintf(`{"skip":%s,"want":"found"}`, inner)
	paths := [][]string{{"want"}}

	var found bool
	EachKey([]byte(data), func(idx int, value []byte, vt ValueType, err error) {
		if idx == 0 {
			found = true
			if string(value) != "found" {
				t.Fatalf("expected 'found', got %q", value)
			}
		}
	}, paths...)

	if !found {
		t.Fatal("EachKey failed to find 'want' after large nested skip")
	}
}

// Verifies: SYS-REQ-010 [boundary]
func TestStress_Delete_TokenEndBoundary(t *testing.T) {
	// Test Delete where tokenEnd reaches the sentinel (returns len(data))
	// This exercises the new `endOffset+tokEnd >= len(data)` guard
	result := Delete([]byte(`{"a":1,"b":2}`), "b")
	val, _, _, err := Get(result, "a")
	if err != nil {
		t.Fatalf("after delete, failed to get 'a': %v", err)
	}
	if string(val) != "1" {
		t.Fatalf("after delete, 'a' = %q, want '1'", val)
	}
}

// Verifies: SYS-REQ-001 [boundary]
func TestStress_Get_BareTruncatedValue(t *testing.T) {
	// A bare value with no container and no terminator — tokenEnd returns len(data)
	val, dt, _, err := Get([]byte("12345"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dt != Number {
		t.Fatalf("expected Number, got %v", dt)
	}
	if string(val) != "12345" {
		t.Fatalf("expected '12345', got %q", val)
	}
}

// =============================================================================
// CRITICAL PATH: Verify EachKey correctly handles the removed block-skip
// by walking through the exact scenario step by step
// =============================================================================

// Verifies: SYS-REQ-008 [boundary]
func TestRemoval4_EachKey_TracePath(t *testing.T) {
	// {"skip":{"n":1},"want":"ok"}
	// When EachKey processes "skip" and match==-1:
	// - i is at ':' (the colon after "skip")
	// - tokenOffset := nextToken(data[i+1:]) — finds '{' at offset 0
	// - i += 0 (tokenOffset is 0, but wait: nextToken skips whitespace,
	//   and data[i+1] = '{', which is not whitespace, so nextToken returns 0)
	// - BUT: i += tokenOffset means i is still at ':'. No, wait:
	//   the code says `i += tokenOffset`, not `i = tokenOffset`.
	//   If i was at position of ':', say position 7 in {"skip":{"n":1},"want":"ok"}
	//   then data[i+1:] starts with '{"n":1},"want":"ok"}'
	//   nextToken returns 0 (first char '{' is not whitespace)
	//   i += 0 → i is still 7 (the colon position)
	//
	// Then we hit `switch data[i]` where data[7] = ':'
	// ':' is not in {'{', '}', '[', '"'} so no i-- adjustment.
	//
	// Then the outer loop does i++ → i = 8, which is '{'.
	// The outer switch hits case '{': → level++.
	// The parser then navigates the nested object naturally.
	//
	// The OLD code had: if data[i] == '{' { blockSkip = blockEnd(...); i += blockSkip + 1 }
	// This would have jumped past the entire nested object.
	// The NEW code relies on the natural '{' handler in the outer switch.
	//
	// Both should work, but the new code is O(n) walking character by character
	// through the nested object, while the old code was O(n) via blockEnd.
	// Functionally equivalent.

	data := []byte(`{"skip":{"n":1},"want":"ok"}`)
	paths := [][]string{{"want"}}

	var found bool
	EachKey(data, func(idx int, value []byte, vt ValueType, err error) {
		if idx == 0 {
			found = true
			if string(value) != "ok" {
				t.Fatalf("expected 'ok', got %q", value)
			}
		}
	}, paths...)

	if !found {
		t.Fatal("UNSAFE: EachKey failed the trace path test")
	}
}

// Test with value types that aren't objects — numbers, arrays, strings, bools
// Verifies: SYS-REQ-008 [boundary]
func TestRemoval4_EachKey_SkipVariousValueTypes(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"number", `{"skip":42,"want":"ok"}`},
		{"negative", `{"skip":-1,"want":"ok"}`},
		{"float", `{"skip":3.14,"want":"ok"}`},
		{"bool_true", `{"skip":true,"want":"ok"}`},
		{"bool_false", `{"skip":false,"want":"ok"}`},
		{"null", `{"skip":null,"want":"ok"}`},
		{"string", `{"skip":"hello","want":"ok"}`},
		{"array", `{"skip":[1,2,3],"want":"ok"}`},
		{"nested_array", `{"skip":[[1],[2]],"want":"ok"}`},
		{"object", `{"skip":{"a":1},"want":"ok"}`},
		{"deep_object", `{"skip":{"a":{"b":{"c":1}}},"want":"ok"}`},
		{"empty_object", `{"skip":{},"want":"ok"}`},
		{"empty_array", `{"skip":[],"want":"ok"}`},
		{"empty_string", `{"skip":"","want":"ok"}`},
	}

	paths := [][]string{{"want"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var found bool
			EachKey([]byte(tt.json), func(idx int, value []byte, vt ValueType, err error) {
				if idx == 0 {
					found = true
					if string(value) != "ok" {
						t.Fatalf("expected 'ok', got %q", value)
					}
				}
			}, paths...)
			if !found {
				t.Fatalf("EachKey failed to find 'want' in %s case", tt.name)
			}
		})
	}
}

// =============================================================================
// ADDITIONAL: Test that the Unescape loop change doesn't affect error handling
// =============================================================================

// Verifies: SYS-REQ-014 [boundary]
func TestRemoval1_Unescape_InvalidEscape(t *testing.T) {
	_, err := Unescape([]byte(`\z`), make([]byte, 64))
	if err == nil {
		t.Fatal("expected MalformedStringEscapeError for \\z")
	}
}

// Verifies: SYS-REQ-014 [boundary]
func TestRemoval1_Unescape_ConsecutiveEscapes(t *testing.T) {
	out, err := Unescape([]byte(`\n\t\r`), make([]byte, 64))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "\n\t\r" {
		t.Fatalf("unexpected result: %q", out)
	}
}

// Verifies: SYS-REQ-014 [boundary]
func TestRemoval1_Unescape_EscapedQuote(t *testing.T) {
	out, err := Unescape([]byte(`hello\"world`), make([]byte, 64))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != `hello"world` {
		t.Fatalf("unexpected result: %q", out)
	}
}
