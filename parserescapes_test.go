package jsonparser

import (
	"testing"
)

func TestH2I(t *testing.T) {
	hexChars := []byte{'0', '9', 'A', 'F', 'a', 'f', 'x', '\000'}
	hexValues := []int{0, 9, 10, 15, 10, 15, -1, -1}

	for i, c := range hexChars {
		if v := h2I(c); v != hexValues[i] {
			t.Errorf("h2I('%c') returned wrong value (obtained %d, expected %d)", c, v, hexValues[i])
		}
	}
}

func TestDecodeSingleUnicodeEscape(t *testing.T) {
	escapeSequences := []string{
		`\"`,
		`\\`,
		`\n`,
		`\t`,
		`\r`,
		`\/`,
		`\b`,
		`\f`,
	}

	runeValues := []struct {
		r  rune
		ok bool
	}{
		{'"', true},
		{'\\', true},
		{'\n', true},
		{'\t', true},
		{'/', true},
		{'\b', true},
		{'\f', true},
	}

	for i, esc := range escapeSequences {
		expected := runeValues[i]
		if r, ok := decodeSingleUnicodeEscape([]byte(esc)); ok != expected.ok {
			t.Errorf("decodeSingleUnicodeEscape(%s) returned 'ok' mismatch: expected %t, obtained %t", esc, expected.ok, ok)
		} else if r != expected.r {
			t.Errorf("decodeSingleUnicodeEscape(%s) returned rune mismatch: expected %x (%c), obtained %x (%c)", esc, expected.r, expected.r, r, r)
		}
	}
}
