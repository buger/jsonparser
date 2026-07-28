package jsonparser

import (
	"bytes"
	"testing"
)

func inputWithSpareCapacity(data string) ([]byte, []byte) {
	input := make([]byte, len(data), len(data)+100)
	copy(input, data)

	backing := input[:cap(input)]
	for i := len(input); i < len(backing); i++ {
		backing[i] = 0xa5
	}

	snapshot := append([]byte(nil), backing...)
	return input, snapshot
}

func assertBackingArrayUnchanged(t *testing.T, input, snapshot []byte) {
	t.Helper()

	if got := input[:cap(input)]; !bytes.Equal(got, snapshot) {
		t.Fatalf("input backing array changed:\n got: %v\nwant: %v", got, snapshot)
	}
}

// Verifies: SYS-REQ-009 [regression, issues #209 and #141]
func TestSetDoesNotMutateInputBackingArrayWhenGrowing(t *testing.T) {
	tests := []struct {
		name     string
		document string
		value    string
		path     []string
		want     string
	}{
		{
			name:     "insert top-level key",
			document: `{"a":1}`,
			value:    `2`,
			path:     []string{"b"},
			want:     `{"a":1,"b":2}`,
		},
		{
			name:     "grow missing nested path",
			document: `{"a":{}}`,
			value:    `"value"`,
			path:     []string{"a", "b", "c"},
			want:     `{"a":{"b":{"c":"value"}}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input, snapshot := inputWithSpareCapacity(tc.document)

			result, err := Set(input, []byte(tc.value), tc.path...)
			if err != nil {
				t.Fatalf("Set returned error: %v", err)
			}
			if got := string(result); got != tc.want {
				t.Fatalf("Set result = %s, want %s", got, tc.want)
			}
			assertBackingArrayUnchanged(t, input, snapshot)

			result[0] ^= 0xff
			assertBackingArrayUnchanged(t, input, snapshot)
		})
	}
}

// Verifies: SYS-REQ-010 [regression, issues #209 and #141]
func TestDeleteDoesNotAliasInputBackingArray(t *testing.T) {
	t.Run("delete existing key", func(t *testing.T) {
		input, snapshot := inputWithSpareCapacity(`{"a":1,"b":2}`)

		result := Delete(input, "a")
		if got, want := string(result), `{"b":2}`; got != want {
			t.Fatalf("Delete result = %s, want %s", got, want)
		}
		assertBackingArrayUnchanged(t, input, snapshot)

		result[0] ^= 0xff
		assertBackingArrayUnchanged(t, input, snapshot)
	})

	t.Run("delete root with empty path", func(t *testing.T) {
		input, snapshot := inputWithSpareCapacity(`{"a":1}`)

		result := Delete(input)
		if len(result) != 0 {
			t.Fatalf("Delete result = %q, want an empty document", result)
		}
		assertBackingArrayUnchanged(t, input, snapshot)

		result = append(result, 'x')
		assertBackingArrayUnchanged(t, input, snapshot)
	})
}

// Verifies: SYS-REQ-008 [regression, issue #232]
func TestEachKeyDescendsIntoArrayIndexAtEndOfPath(t *testing.T) {
	data := []byte(`{"request":{"headers":{"User-Agent":["abc"]}}}`)
	path := []string{"request", "headers", "User-Agent", "[0]"}

	getValue, getType, _, err := Get(data, path...)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	callbacks := 0
	EachKey(data, func(idx int, value []byte, valueType ValueType, err error) {
		callbacks++
		if err != nil {
			t.Errorf("EachKey callback returned error: %v", err)
		}
		if idx != 0 {
			t.Errorf("EachKey callback index = %d, want 0", idx)
		}
		if !bytes.Equal(value, getValue) {
			t.Errorf("EachKey value = %q, want Get value %q", value, getValue)
		}
		if valueType != getType {
			t.Errorf("EachKey type = %v, want Get type %v", valueType, getType)
		}
	}, path)

	if callbacks != 1 {
		t.Fatalf("EachKey callback count = %d, want 1", callbacks)
	}
}
