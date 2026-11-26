package jsonparser

import (
	"bytes"
	"errors"
	"testing"
)

// Verifies: SYS-REQ-110 (Set/array append)
// reqproof:proptest:skip
func TestAppendTopLevelArray(t *testing.T) {
	got, err := Append([]byte(`[1,2,3]`), []byte(`4`))
	if err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	if want := `[1,2,3,4]`; string(got) != want {
		t.Fatalf("Append result = %s, want %s", got, want)
	}
}

// Verifies: SYS-REQ-110 (Set/array append)
// reqproof:proptest:skip
func TestAppendEmptyArray(t *testing.T) {
	got, err := Append([]byte(`[]`), []byte(`1`))
	if err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	if want := `[1]`; string(got) != want {
		t.Fatalf("Append result = %s, want %s", got, want)
	}
}

// Verifies: SYS-REQ-110 (Set/array append)
// reqproof:proptest:skip
func TestAppendNestedArray(t *testing.T) {
	got, err := Append([]byte(`{"items":[1,2]}`), []byte(`3`), "items")
	if err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	if want := `{"items":[1,2,3]}`; string(got) != want {
		t.Fatalf("Append result = %s, want %s", got, want)
	}
}

// Verifies: SYS-REQ-110 (Set/array append)
// reqproof:proptest:skip
func TestAppendObject(t *testing.T) {
	got, err := Append([]byte(`{"users":[]}`), []byte(`{"name":"Bob"}`), "users")
	if err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	if want := `{"users":[{"name":"Bob"}]}`; string(got) != want {
		t.Fatalf("Append result = %s, want %s", got, want)
	}
}

// Verifies: SYS-REQ-110 (Set/array append)
// reqproof:proptest:skip
func TestAppendString(t *testing.T) {
	got, err := Append([]byte(`{"tags":["a"]}`), Escape("b"), "tags")
	if err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	if want := `{"tags":["a","b"]}`; string(got) != want {
		t.Fatalf("Append result = %s, want %s", got, want)
	}
}

// Verifies: SYS-REQ-110 (Set/array append)
// reqproof:proptest:skip
func TestAppendCreatePath(t *testing.T) {
	got, err := Append([]byte(`{"a":{}}`), []byte(`1`), "a", "items")
	if err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	if want := `{"a":{"items":[1]}}`; string(got) != want {
		t.Fatalf("Append result = %s, want %s", got, want)
	}
}

// Verifies: SYS-REQ-110 (Set/array append)
// reqproof:proptest:skip
func TestAppendNonArrayError(t *testing.T) {
	got, err := Append([]byte(`{"a":1}`), []byte(`2`), "a")
	if !errors.Is(err, MalformedArrayError) {
		t.Fatalf("Append error = %v, want %v", err, MalformedArrayError)
	}
	if got != nil {
		t.Fatalf("Append result = %s, want nil", got)
	}
}

// Verifies: SYS-REQ-110 (Set/array append)
// reqproof:proptest:skip
func TestAppendRoundTrip(t *testing.T) {
	got, err := Append([]byte(`[1,2]`), []byte(`{"id":3}`))
	if err != nil {
		t.Fatalf("Append returned error: %v", err)
	}

	last, dataType, _, err := Get(got, "[2]")
	if err != nil {
		t.Fatalf("Get appended value returned error: %v", err)
	}
	if dataType != Object {
		t.Fatalf("Get appended value type = %v, want %v", dataType, Object)
	}
	if want := `{"id":3}`; string(last) != want {
		t.Fatalf("Get appended value = %s, want %s", last, want)
	}
}

// Verifies: SYS-REQ-110 (Set/array append)
// reqproof:proptest:skip
func TestAppendMultiple(t *testing.T) {
	got := []byte(`[]`)
	var err error
	for _, value := range []string{"1", "2", "3"} {
		got, err = Append(got, []byte(value))
		if err != nil {
			t.Fatalf("Append %s returned error: %v", value, err)
		}
	}
	if want := `[1,2,3]`; string(got) != want {
		t.Fatalf("Append result = %s, want %s", got, want)
	}
}

// Verifies: SYS-REQ-110 (Set/array append)
// reqproof:proptest:skip
func TestAppendDoesNotMutateInput(t *testing.T) {
	input := []byte(`{"items":[1,2]}`)
	original := bytes.Clone(input)

	got, err := Append(input, []byte(`3`), "items")
	if err != nil {
		t.Fatalf("Append returned error: %v", err)
	}
	if !bytes.Equal(input, original) {
		t.Fatalf("Append mutated input: got %s, want %s", input, original)
	}
	if want := `{"items":[1,2,3]}`; string(got) != want {
		t.Fatalf("Append result = %s, want %s", got, want)
	}
}
