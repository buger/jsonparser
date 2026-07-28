package jsonparser

import (
	"errors"
	"testing"
)

// Verifies: SYS-REQ-112 (container length accessor).
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
// SYS-REQ-112:nominal:nominal
// SYS-REQ-112:boundary:nominal
// SYS-REQ-112:malformed_input:nominal
// SYS-REQ-112:malformed_input:negative
func TestGetArrayLen(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		keys    []string
		want    int
		wantErr error
	}{
		{
			name: "empty array",
			data: `[]`,
			want: 0,
		},
		{
			name: "one element",
			data: `{"values":[1]}`,
			keys: []string{"values"},
			want: 1,
		},
		{
			name: "five elements",
			data: `[1,2,3,4,5]`,
			want: 5,
		},
		{
			name: "nested array inner length",
			data: `{"values":[[1,2,3],["ignored"]]}`,
			keys: []string{"values", "[0]"},
			want: 3,
		},
		{
			name: "mixed types",
			data: `[1,true,null,{"nested":[1,2]},["x","y"]]`,
			want: 5,
		},
		{
			name: "array of strings",
			data: `["one,two","three","four,five,six"]`,
			want: 3,
		},
		{
			name:    "malformed array",
			data:    `[1,2`,
			wantErr: MalformedArrayError,
		},
		{
			name:    "addressed value is not an array",
			data:    `{"values":1}`,
			keys:    []string{"values"},
			wantErr: KeyPathNotFoundError,
		},
		{
			name:    "missing path",
			data:    `{"values":[]}`,
			keys:    []string{"missing"},
			wantErr: KeyPathNotFoundError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := GetArrayLen([]byte(test.data), test.keys...)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("GetArrayLen() error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetArrayLen() unexpected error: %v", err)
			}
			if got != test.want {
				t.Fatalf("GetArrayLen() = %d, want %d", got, test.want)
			}
		})
	}
}

// Verifies: SYS-REQ-112 (container length accessor).
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
// SYS-REQ-112:nominal:nominal
// SYS-REQ-112:boundary:nominal
// SYS-REQ-112:malformed_input:nominal
// SYS-REQ-112:malformed_input:negative
func TestGetObjectLen(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		keys    []string
		want    int
		wantErr error
	}{
		{
			name: "empty object",
			data: `{}`,
			want: 0,
		},
		{
			name: "one key",
			data: `{"one":1}`,
			want: 1,
		},
		{
			name: "five keys",
			data: `{"one":1,"two":2,"three":3,"four":4,"five":5}`,
			want: 5,
		},
		{
			name: "nested object",
			data: `{"outer":{"one":1,"two":{"nested":true},"three":3}}`,
			keys: []string{"outer"},
			want: 3,
		},
		{
			name: "string values containing commas",
			data: `{"one":"a,b,c","two":"d,e","three":"no comma"}`,
			want: 3,
		},
		{
			name:    "addressed value is not an object",
			data:    `{"value":[]}`,
			keys:    []string{"value"},
			wantErr: KeyPathNotFoundError,
		},
		{
			name:    "missing path",
			data:    `{"value":{}}`,
			keys:    []string{"missing"},
			wantErr: KeyPathNotFoundError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := GetObjectLen([]byte(test.data), test.keys...)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("GetObjectLen() error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetObjectLen() unexpected error: %v", err)
			}
			if got != test.want {
				t.Fatalf("GetObjectLen() = %d, want %d", got, test.want)
			}
		})
	}
}

// Verifies: SYS-REQ-003 (GetInt/typed accessors).
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestGetUint64(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		keys    []string
		want    uint64
		wantErr error
		anyErr  bool
	}{
		{
			name: "zero",
			data: `{"value":0}`,
			keys: []string{"value"},
			want: 0,
		},
		{
			name: "one",
			data: `{"value":1}`,
			keys: []string{"value"},
			want: 1,
		},
		{
			name: "max uint64",
			data: `{"value":18446744073709551615}`,
			keys: []string{"value"},
			want: ^uint64(0),
		},
		{
			name:    "negative",
			data:    `{"value":-1}`,
			keys:    []string{"value"},
			wantErr: MalformedValueError,
		},
		{
			name: "max int64",
			data: `{"value":9223372036854775807}`,
			keys: []string{"value"},
			want: uint64(9223372036854775807),
		},
		{
			name:    "max uint64 plus one",
			data:    `{"value":18446744073709551616}`,
			keys:    []string{"value"},
			wantErr: OverflowIntegerError,
		},
		{
			name:   "string value",
			data:   `{"value":"1"}`,
			keys:   []string{"value"},
			anyErr: true,
		},
		{
			name:    "missing key",
			data:    `{"value":1}`,
			keys:    []string{"missing"},
			wantErr: KeyPathNotFoundError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := GetUint64([]byte(test.data), test.keys...)
			if test.anyErr {
				if err == nil {
					t.Fatal("GetUint64() error = nil, want a type error")
				}
				return
			}
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("GetUint64() error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetUint64() unexpected error: %v", err)
			}
			if got != test.want {
				t.Fatalf("GetUint64() = %d, want %d", got, test.want)
			}
		})
	}
}

// Verifies: SYS-REQ-112 (container length accessor).
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestGetArrayLenRoundTrip(t *testing.T) {
	data := []byte(`{"values":[1,"two,three",{"four":[4,5]},[6,7],null]}`)

	got, err := GetArrayLen(data, "values")
	if err != nil {
		t.Fatalf("GetArrayLen() unexpected error: %v", err)
	}

	callbacks := 0
	_, err = ArrayEach(data, func(_ []byte, _ ValueType, _ int, callbackErr error) {
		if callbackErr != nil {
			t.Errorf("ArrayEach callback error: %v", callbackErr)
			return
		}
		callbacks++
	}, "values")
	if err != nil {
		t.Fatalf("ArrayEach() unexpected error: %v", err)
	}
	if got != callbacks {
		t.Fatalf("GetArrayLen() = %d, ArrayEach callbacks = %d", got, callbacks)
	}
}

// Verifies: SYS-REQ-112 (container length accessor).
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestGetObjectLenRoundTrip(t *testing.T) {
	data := []byte(`{"value":{"one":1,"two":"a,b","three":{"nested":true},"four":[1,2]}}`)

	got, err := GetObjectLen(data, "value")
	if err != nil {
		t.Fatalf("GetObjectLen() unexpected error: %v", err)
	}

	callbacks := 0
	err = ObjectEach(data, func(_ []byte, _ []byte, _ ValueType, _ int) error {
		callbacks++
		return nil
	}, "value")
	if err != nil {
		t.Fatalf("ObjectEach() unexpected error: %v", err)
	}
	if got != callbacks {
		t.Fatalf("GetObjectLen() = %d, ObjectEach callbacks = %d", got, callbacks)
	}
}
