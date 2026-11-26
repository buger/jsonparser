package jsonparser

import (
	"errors"
	"reflect"
	"testing"
)

// Verifies: SYS-REQ-113 (wildcard path support)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
// SYS-REQ-113:nominal:nominal
// SYS-REQ-113:boundary:nominal
// SYS-REQ-113:malformed_input:nominal
// SYS-REQ-113:malformed_input:negative
func TestEachKeyWildcard(t *testing.T) {
	t.Run("single wildcard at end", func(t *testing.T) {
		data := []byte(`{"values":[1,"two",true,null]}`)
		var indices []int
		var values []string
		var types []ValueType

		err := EachKeyWildcard(data, func(idx int, value []byte, vt ValueType, err error) {
			if err != nil {
				t.Fatalf("callback error = %v, want nil", err)
			}
			indices = append(indices, idx)
			values = append(values, string(value))
			types = append(types, vt)
		}, "values", "[*]")

		if err != nil {
			t.Fatalf("EachKeyWildcard() error = %v, want nil", err)
		}
		if want := []int{0, 1, 2, 3}; !reflect.DeepEqual(indices, want) {
			t.Fatalf("indices = %v, want %v", indices, want)
		}
		if want := []string{"1", "two", "true", "null"}; !reflect.DeepEqual(values, want) {
			t.Fatalf("values = %q, want %q", values, want)
		}
		if want := []ValueType{Number, String, Boolean, Null}; !reflect.DeepEqual(types, want) {
			t.Fatalf("types = %v, want %v", types, want)
		}
	})

	t.Run("wildcard in middle", func(t *testing.T) {
		data := []byte(`{"users":[{"name":"Alice"},{"name":"Bob"}]}`)
		var names []string

		err := EachKeyWildcard(data, func(_ int, value []byte, _ ValueType, err error) {
			if err != nil {
				t.Fatalf("callback error = %v, want nil", err)
			}
			names = append(names, string(value))
		}, "users", "[*]", "name")

		if err != nil {
			t.Fatalf("EachKeyWildcard() error = %v, want nil", err)
		}
		if want := []string{"Alice", "Bob"}; !reflect.DeepEqual(names, want) {
			t.Fatalf("names = %q, want %q", names, want)
		}
	})

	t.Run("multiple wildcards in nested arrays", func(t *testing.T) {
		data := []byte(`{"matrix":[[1,2],[3,4]]}`)
		var values []string

		err := EachKeyWildcard(data, func(_ int, value []byte, _ ValueType, err error) {
			if err != nil {
				t.Fatalf("callback error = %v, want nil", err)
			}
			values = append(values, string(value))
		}, "matrix", "[*]", "[*]")

		if err != nil {
			t.Fatalf("EachKeyWildcard() error = %v, want nil", err)
		}
		if want := []string{"1", "2", "3", "4"}; !reflect.DeepEqual(values, want) {
			t.Fatalf("values = %q, want %q", values, want)
		}
	})

	t.Run("wildcard requires array", func(t *testing.T) {
		calls := 0
		err := EachKeyWildcard([]byte(`{"users":{"name":"Alice"}}`), func(_ int, _ []byte, _ ValueType, _ error) {
			calls++
		}, "users", "[*]", "name")

		if !errors.Is(err, MalformedArrayError) {
			t.Fatalf("EachKeyWildcard() error = %v, want %v", err, MalformedArrayError)
		}
		if calls != 0 {
			t.Fatalf("callback calls = %d, want 0", calls)
		}
	})

	t.Run("empty array", func(t *testing.T) {
		calls := 0
		err := EachKeyWildcard([]byte(`{"users":[]}`), func(_ int, _ []byte, _ ValueType, _ error) {
			calls++
		}, "users", "[*]", "name")

		if err != nil {
			t.Fatalf("EachKeyWildcard() error = %v, want nil", err)
		}
		if calls != 0 {
			t.Fatalf("callback calls = %d, want 0", calls)
		}
	})
}

// Verifies: SYS-REQ-113 (wildcard path support)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
// SYS-REQ-113:nominal:nominal
func TestArrayEachWildcard(t *testing.T) {
	data := []byte(`{"values":[10,20,30]}`)
	var indices []int
	var values []string
	var offsets []int

	count, err := ArrayEachWildcard(data, func(idx int, value []byte, _ ValueType, offset int, err error) error {
		if err != nil {
			return err
		}
		indices = append(indices, idx)
		values = append(values, string(value))
		offsets = append(offsets, offset)
		return nil
	}, "values", "[*]")

	if err != nil {
		t.Fatalf("ArrayEachWildcard() error = %v, want nil", err)
	}
	if count != 3 {
		t.Fatalf("ArrayEachWildcard() count = %d, want 3", count)
	}
	if want := []int{0, 1, 2}; !reflect.DeepEqual(indices, want) {
		t.Fatalf("indices = %v, want %v", indices, want)
	}
	if want := []string{"10", "20", "30"}; !reflect.DeepEqual(values, want) {
		t.Fatalf("values = %q, want %q", values, want)
	}
	if !(offsets[0] < offsets[1] && offsets[1] < offsets[2]) {
		t.Fatalf("offsets = %v, want strictly increasing offsets", offsets)
	}
}

// Verifies: SYS-REQ-113 (wildcard path support)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
// SYS-REQ-113:nominal:nominal
// SYS-REQ-113:malformed_input:nominal
// SYS-REQ-113:malformed_input:negative
func TestSetWildcard(t *testing.T) {
	t.Run("sets field on every item", func(t *testing.T) {
		data := []byte(`{"items":[{"id":1},{"id":2}]}`)
		got, err := SetWildcard(data, []byte(`true`), "items", "[*]", "flag")
		if err != nil {
			t.Fatalf("SetWildcard() error = %v, want nil", err)
		}

		want := `{"items":[{"id":1,"flag":true},{"id":2,"flag":true}]}`
		if string(got) != want {
			t.Fatalf("SetWildcard() = %s, want %s", got, want)
		}
	})

	t.Run("multiple wildcards", func(t *testing.T) {
		data := []byte(`{"groups":[{"items":[{"flag":false},{"flag":false}]},{"items":[{"flag":false}]}]}`)
		got, err := SetWildcard(data, []byte(`true`), "groups", "[*]", "items", "[*]", "flag")
		if err != nil {
			t.Fatalf("SetWildcard() error = %v, want nil", err)
		}

		want := `{"groups":[{"items":[{"flag":true},{"flag":true}]},{"items":[{"flag":true}]}]}`
		if string(got) != want {
			t.Fatalf("SetWildcard() = %s, want %s", got, want)
		}
	})

	t.Run("empty array is unchanged", func(t *testing.T) {
		data := []byte(`{"items":[]}`)
		got, err := SetWildcard(data, []byte(`true`), "items", "[*]", "flag")
		if err != nil {
			t.Fatalf("SetWildcard() error = %v, want nil", err)
		}
		if string(got) != string(data) {
			t.Fatalf("SetWildcard() = %s, want unchanged %s", got, data)
		}
	})

	t.Run("wildcard requires array", func(t *testing.T) {
		_, err := SetWildcard([]byte(`{"items":{"flag":false}}`), []byte(`true`), "items", "[*]", "flag")
		if !errors.Is(err, MalformedArrayError) {
			t.Fatalf("SetWildcard() error = %v, want %v", err, MalformedArrayError)
		}
	})
}
