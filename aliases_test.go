package jsonparser

import (
	"errors"
	"reflect"
	"testing"
)

type arrayAliasObservation struct {
	value    string
	dataType ValueType
	offset   int
	err      string
}

type objectAliasObservation struct {
	key      string
	value    string
	dataType ValueType
	offset   int
}

type wildcardAliasObservation struct {
	index    int
	value    string
	dataType ValueType
	offset   int
	err      string
}

func aliasErrorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// Verifies: SYS-REQ-006 (array iteration alias)
// reqproof:proptest:skip parity test for a thin compatibility wrapper
func TestEachArrayAlias(t *testing.T) {
	data := []byte(`{"payload":{"items":[1,"two",true,null]}}`)
	var original, alias []arrayAliasObservation

	_, originalErr := ArrayEach(data, func(value []byte, dataType ValueType, offset int, err error) {
		original = append(original, arrayAliasObservation{
			value:    string(value),
			dataType: dataType,
			offset:   offset,
			err:      aliasErrorString(err),
		})
	}, "payload", "items")
	if originalErr != nil {
		t.Fatalf("ArrayEach() error = %v, want nil", originalErr)
	}

	EachArray(data, func(value []byte, dataType ValueType, offset int, err error) {
		alias = append(alias, arrayAliasObservation{
			value:    string(value),
			dataType: dataType,
			offset:   offset,
			err:      aliasErrorString(err),
		})
	}, "payload", "items")

	if !reflect.DeepEqual(alias, original) {
		t.Fatalf("EachArray() observations = %#v, ArrayEach() observations = %#v", alias, original)
	}
}

// Verifies: SYS-REQ-007 (object iteration alias)
// reqproof:proptest:skip parity test for a thin compatibility wrapper
func TestEachObjectAlias(t *testing.T) {
	data := []byte(`{"payload":{"first":1,"second":"two","third":true}}`)
	stop := errors.New("stop")
	var original, alias []objectAliasObservation

	originalErr := ObjectEach(data, func(key []byte, value []byte, dataType ValueType, offset int) error {
		original = append(original, objectAliasObservation{
			key:      string(key),
			value:    string(value),
			dataType: dataType,
			offset:   offset,
		})
		if len(original) == 2 {
			return stop
		}
		return nil
	}, "payload")

	aliasErr := EachObject(data, func(key []byte, value []byte, dataType ValueType, offset int) error {
		alias = append(alias, objectAliasObservation{
			key:      string(key),
			value:    string(value),
			dataType: dataType,
			offset:   offset,
		})
		if len(alias) == 2 {
			return stop
		}
		return nil
	}, "payload")

	if !errors.Is(aliasErr, originalErr) {
		t.Fatalf("EachObject() error = %v, ObjectEach() error = %v", aliasErr, originalErr)
	}
	if !reflect.DeepEqual(alias, original) {
		t.Fatalf("EachObject() observations = %#v, ObjectEach() observations = %#v", alias, original)
	}
}

// Verifies: SYS-REQ-004 (error-returning array iteration alias)
// reqproof:proptest:skip parity test for a thin compatibility wrapper
func TestEachArrayErrAlias(t *testing.T) {
	data := []byte(`{"payload":{"items":[1,"two",true,null]}}`)
	stop := errors.New("stop")
	var original, alias []arrayAliasObservation

	originalCount, originalErr := ArrayEachErr(data, func(value []byte, dataType ValueType, offset int, err error) error {
		original = append(original, arrayAliasObservation{
			value:    string(value),
			dataType: dataType,
			offset:   offset,
			err:      aliasErrorString(err),
		})
		if len(original) == 3 {
			return stop
		}
		return nil
	}, "payload", "items")

	aliasCount, aliasErr := EachArrayErr(data, func(value []byte, dataType ValueType, offset int, err error) error {
		alias = append(alias, arrayAliasObservation{
			value:    string(value),
			dataType: dataType,
			offset:   offset,
			err:      aliasErrorString(err),
		})
		if len(alias) == 3 {
			return stop
		}
		return nil
	}, "payload", "items")

	if aliasCount != originalCount {
		t.Fatalf("EachArrayErr() count = %d, ArrayEachErr() count = %d", aliasCount, originalCount)
	}
	if !errors.Is(aliasErr, originalErr) {
		t.Fatalf("EachArrayErr() error = %v, ArrayEachErr() error = %v", aliasErr, originalErr)
	}
	if !reflect.DeepEqual(alias, original) {
		t.Fatalf("EachArrayErr() observations = %#v, ArrayEachErr() observations = %#v", alias, original)
	}
}

// Verifies: SYS-REQ-113 (wildcard array iteration alias)
// reqproof:proptest:skip parity test for a thin compatibility wrapper
func TestEachArrayWildcardAlias(t *testing.T) {
	data := []byte(`{"payload":{"items":[1,"two",true,null]}}`)
	var original, alias []wildcardAliasObservation

	originalCount, originalErr := ArrayEachWildcard(data, func(index int, value []byte, dataType ValueType, offset int, err error) error {
		original = append(original, wildcardAliasObservation{
			index:    index,
			value:    string(value),
			dataType: dataType,
			offset:   offset,
			err:      aliasErrorString(err),
		})
		return nil
	}, "payload", "items", "[*]")

	aliasCount, aliasErr := EachArrayWildcard(data, func(index int, value []byte, dataType ValueType, offset int, err error) error {
		alias = append(alias, wildcardAliasObservation{
			index:    index,
			value:    string(value),
			dataType: dataType,
			offset:   offset,
			err:      aliasErrorString(err),
		})
		return nil
	}, "payload", "items", "[*]")

	if aliasCount != originalCount {
		t.Fatalf("EachArrayWildcard() count = %d, ArrayEachWildcard() count = %d", aliasCount, originalCount)
	}
	if !errors.Is(aliasErr, originalErr) {
		t.Fatalf("EachArrayWildcard() error = %v, ArrayEachWildcard() error = %v", aliasErr, originalErr)
	}
	if !reflect.DeepEqual(alias, original) {
		t.Fatalf("EachArrayWildcard() observations = %#v, ArrayEachWildcard() observations = %#v", alias, original)
	}
}
