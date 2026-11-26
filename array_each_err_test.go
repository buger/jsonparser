package jsonparser

import (
	"errors"
	"io"
	"reflect"
	"testing"
)

// Verifies: SYS-REQ-004 (ArrayEach)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestArrayEachErr(t *testing.T) {
	t.Run("normal iteration", func(t *testing.T) {
		var values []string
		count, err := ArrayEachErr([]byte(`[1, "two", true, null]`), func(value []byte, dataType ValueType, offset int, err error) error {
			if err != nil {
				t.Fatalf("callback error = %v, want nil", err)
			}
			values = append(values, string(value))
			return nil
		})

		if err != nil {
			t.Fatalf("ArrayEachErr() error = %v, want nil", err)
		}
		if count != 4 {
			t.Fatalf("ArrayEachErr() count = %d, want 4", count)
		}
		if want := []string{"1", "two", "true", "null"}; !reflect.DeepEqual(values, want) {
			t.Fatalf("ArrayEachErr() values = %q, want %q", values, want)
		}
	})

	t.Run("callback error stops iteration", func(t *testing.T) {
		wantErr := errors.New("stop after third element")
		var values []string
		count, err := ArrayEachErr([]byte(`[1, 2, 3, 4, 5]`), func(value []byte, dataType ValueType, offset int, err error) error {
			values = append(values, string(value))
			if len(values) == 3 {
				return wantErr
			}
			return nil
		})

		if !errors.Is(err, wantErr) {
			t.Fatalf("ArrayEachErr() error = %v, want %v", err, wantErr)
		}
		if count != 3 {
			t.Fatalf("ArrayEachErr() count = %d, want 3", count)
		}
		if want := []string{"1", "2", "3"}; !reflect.DeepEqual(values, want) {
			t.Fatalf("ArrayEachErr() values = %q, want %q", values, want)
		}
	})

	t.Run("io EOF stops gracefully", func(t *testing.T) {
		var values []string
		count, err := ArrayEachErr([]byte(`[1, 2, 3, 4]`), func(value []byte, dataType ValueType, offset int, err error) error {
			values = append(values, string(value))
			if len(values) == 2 {
				return io.EOF
			}
			return nil
		})

		if err != nil {
			t.Fatalf("ArrayEachErr() error = %v, want nil", err)
		}
		if count != 2 {
			t.Fatalf("ArrayEachErr() count = %d, want 2", count)
		}
		if want := []string{"1", "2"}; !reflect.DeepEqual(values, want) {
			t.Fatalf("ArrayEachErr() values = %q, want %q", values, want)
		}
	})

	t.Run("empty array", func(t *testing.T) {
		count, err := ArrayEachErr([]byte(`[]`), func(value []byte, dataType ValueType, offset int, err error) error {
			t.Fatal("callback invoked for empty array")
			return nil
		})

		if err != nil {
			t.Fatalf("ArrayEachErr() error = %v, want nil", err)
		}
		if count != 0 {
			t.Fatalf("ArrayEachErr() count = %d, want 0", count)
		}
	})

	t.Run("nested key path", func(t *testing.T) {
		data := []byte(`{"outer":{"items":["first","second","third"]}}`)
		var values []string
		count, err := ArrayEachErr(data, func(value []byte, dataType ValueType, offset int, err error) error {
			values = append(values, string(value))
			return nil
		}, "outer", "items")

		if err != nil {
			t.Fatalf("ArrayEachErr() error = %v, want nil", err)
		}
		if count != 3 {
			t.Fatalf("ArrayEachErr() count = %d, want 3", count)
		}
		if want := []string{"first", "second", "third"}; !reflect.DeepEqual(values, want) {
			t.Fatalf("ArrayEachErr() values = %q, want %q", values, want)
		}
	})

	t.Run("parser error is passed to callback", func(t *testing.T) {
		var callbackErr error
		count, err := ArrayEachErr([]byte(`[1, nope, 3]`), func(value []byte, dataType ValueType, offset int, err error) error {
			if err != nil {
				callbackErr = err
				return err
			}
			return nil
		})

		if callbackErr == nil {
			t.Fatal("callback did not receive malformed element error")
		}
		if err != callbackErr {
			t.Fatalf("ArrayEachErr() error = %v, want callback error %v", err, callbackErr)
		}
		if count != 2 {
			t.Fatalf("ArrayEachErr() count = %d, want 2", count)
		}
	})
}

// Verifies: SYS-REQ-008 (EachKey)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestEachKeyErr(t *testing.T) {
	t.Run("normal iteration", func(t *testing.T) {
		data := []byte(`{"first":1,"nested":{"second":"two"},"third":true}`)
		paths := [][]string{{"first"}, {"nested", "second"}, {"third"}}
		var indices []int
		var values []string

		err := EachKeyErr(data, func(idx int, value []byte, vt ValueType, err error) error {
			indices = append(indices, idx)
			values = append(values, string(value))
			return nil
		}, paths...)

		if err != nil {
			t.Fatalf("EachKeyErr() error = %v, want nil", err)
		}
		if want := []int{0, 1, 2}; !reflect.DeepEqual(indices, want) {
			t.Fatalf("EachKeyErr() indices = %v, want %v", indices, want)
		}
		if want := []string{"1", "two", "true"}; !reflect.DeepEqual(values, want) {
			t.Fatalf("EachKeyErr() values = %q, want %q", values, want)
		}
	})

	t.Run("callback error stops iteration", func(t *testing.T) {
		wantErr := errors.New("stop after second key")
		data := []byte(`{"first":1,"second":2,"third":3}`)
		paths := [][]string{{"first"}, {"second"}, {"third"}}
		var calls int

		err := EachKeyErr(data, func(idx int, value []byte, vt ValueType, err error) error {
			calls++
			if calls == 2 {
				return wantErr
			}
			return nil
		}, paths...)

		if !errors.Is(err, wantErr) {
			t.Fatalf("EachKeyErr() error = %v, want %v", err, wantErr)
		}
		if calls != 2 {
			t.Fatalf("EachKeyErr() callbacks = %d, want 2", calls)
		}
	})

	t.Run("io EOF stops gracefully", func(t *testing.T) {
		data := []byte(`{"first":1,"second":2}`)
		paths := [][]string{{"first"}, {"second"}}
		var calls int

		err := EachKeyErr(data, func(idx int, value []byte, vt ValueType, err error) error {
			calls++
			return io.EOF
		}, paths...)

		if err != nil {
			t.Fatalf("EachKeyErr() error = %v, want nil", err)
		}
		if calls != 1 {
			t.Fatalf("EachKeyErr() callbacks = %d, want 1", calls)
		}
	})

	t.Run("stops within array path iteration", func(t *testing.T) {
		wantErr := errors.New("stop in array")
		data := []byte(`{"items":[{"id":1},{"id":2}],"after":3}`)
		paths := [][]string{{"items", "[0]", "id"}, {"items", "[1]", "id"}, {"after"}}
		var indices []int

		err := EachKeyErr(data, func(idx int, value []byte, vt ValueType, err error) error {
			indices = append(indices, idx)
			return wantErr
		}, paths...)

		if !errors.Is(err, wantErr) {
			t.Fatalf("EachKeyErr() error = %v, want %v", err, wantErr)
		}
		if want := []int{0}; !reflect.DeepEqual(indices, want) {
			t.Fatalf("EachKeyErr() indices = %v, want %v", indices, want)
		}
	})

	t.Run("parser error is passed to callback", func(t *testing.T) {
		var callbackErr error
		err := EachKeyErr([]byte(`{"broken":nope}`), func(idx int, value []byte, vt ValueType, err error) error {
			callbackErr = err
			return err
		}, []string{"broken"})

		if callbackErr == nil {
			t.Fatal("callback did not receive malformed value error")
		}
		if err != callbackErr {
			t.Fatalf("EachKeyErr() error = %v, want callback error %v", err, callbackErr)
		}
	})
}
