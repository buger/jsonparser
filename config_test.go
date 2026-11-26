package jsonparser

import (
	"errors"
	"reflect"
	"testing"
)

// Verifies: SYS-REQ-115 (lenient Config parsing)
// SYS-REQ-115:nominal:nominal
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestConfigGet(t *testing.T) {
	data := []byte(`{'person':{'name':'Ada','bio':'brace } and double " quote'}}`)

	value, valueType, _, err := Lenient.Get(data, "person", "name")
	if err != nil {
		t.Fatalf("Lenient.Get returned error: %v", err)
	}
	if valueType != String {
		t.Fatalf("Lenient.Get type = %v, want String", valueType)
	}
	if got, want := string(value), "Ada"; got != want {
		t.Fatalf("Lenient.Get value = %q, want %q", got, want)
	}

	bio, _, _, err := Lenient.Get(data, "person", "bio")
	if err != nil {
		t.Fatalf("Lenient.Get nested string containing brace returned error: %v", err)
	}
	if got, want := string(bio), `brace } and double " quote`; got != want {
		t.Fatalf("Lenient.Get nested value = %q, want %q", got, want)
	}
}

// Verifies: SYS-REQ-115 (single-quoted string decoding)
// SYS-REQ-115:nominal:nominal
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestConfigGetString(t *testing.T) {
	data := []byte(`{'message':'it\'s\nlenient'}`)

	got, err := Lenient.GetString(data, "message")
	if err != nil {
		t.Fatalf("Lenient.GetString returned error: %v", err)
	}
	if want := "it's\nlenient"; got != want {
		t.Fatalf("Lenient.GetString = %q, want %q", got, want)
	}
}

// Verifies: SYS-REQ-115 (mutation of lenient input)
// SYS-REQ-115:nominal:nominal
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestConfigSet(t *testing.T) {
	data := []byte(`{'name':'old','keep':1}`)

	got, err := Lenient.Set(data, []byte(`"new"`), "name")
	if err != nil {
		t.Fatalf("Lenient.Set returned error: %v", err)
	}
	if want := `{'name':"new",'keep':1}`; string(got) != want {
		t.Fatalf("Lenient.Set = %s, want %s", got, want)
	}
}

// Verifies: SYS-REQ-115 (unknown escapes are emitted literally)
// SYS-REQ-115:encoding_safety:nominal
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestConfigAllowUnknownEscapes(t *testing.T) {
	data := []byte("{\"message\":\"hello\\`world\\x\"}")
	config := Config{AllowUnknownEscapes: true}

	raw, valueType, _, err := config.Get(data, "message")
	if err != nil {
		t.Fatalf("Config.Get returned error: %v", err)
	}
	if valueType != String || string(raw) != "hello\\`world\\x" {
		t.Fatalf("Config.Get = (%q, %v), want raw escaped string", raw, valueType)
	}

	got, err := config.GetString(data, "message")
	if err != nil {
		t.Fatalf("Config.GetString returned error: %v", err)
	}
	if want := "hello`worldx"; got != want {
		t.Fatalf("Config.GetString = %q, want %q", got, want)
	}
}

// Verifies: SYS-REQ-115 (zero-value Config remains strict)
// SYS-REQ-115:malformed_input:nominal
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestConfigStrictStillWorks(t *testing.T) {
	if _, _, _, err := DefaultConfig.Get([]byte(`{'value':'data'}`), "value"); err == nil {
		t.Fatal("DefaultConfig.Get accepted single-quoted JSON")
	}

	if _, err := DefaultConfig.GetString([]byte("{\"value\":\"bad\\`escape\"}"), "value"); err == nil {
		t.Fatal("DefaultConfig.GetString accepted an unknown escape")
	}
}

// Verifies: SYS-REQ-115 (package-level API remains strict)
// SYS-REQ-115:malformed_input:nominal
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestConfigBackwardCompat(t *testing.T) {
	if _, _, _, err := Get([]byte(`{'value':'data'}`), "value"); err == nil {
		t.Fatal("package-level Get accepted single-quoted JSON")
	}

	if _, err := GetString([]byte("{\"value\":\"bad\\`escape\"}"), "value"); err == nil {
		t.Fatal("package-level GetString accepted an unknown escape")
	}
}

// Verifies: SYS-REQ-115 (Get/Set round trip)
// SYS-REQ-115:nominal:nominal
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestConfigRoundTrip(t *testing.T) {
	source := []byte(`{'source':42,'target':0}`)
	value, valueType, _, err := Lenient.Get(source, "source")
	if err != nil {
		t.Fatalf("Lenient.Get returned error: %v", err)
	}
	if valueType != Number {
		t.Fatalf("Lenient.Get type = %v, want Number", valueType)
	}

	updated, err := Lenient.Set(source, value, "target")
	if err != nil {
		t.Fatalf("Lenient.Set returned error: %v", err)
	}
	got, _, _, err := Lenient.Get(updated, "target")
	if err != nil {
		t.Fatalf("Lenient.Get after Set returned error: %v", err)
	}
	if !reflect.DeepEqual(got, value) {
		t.Fatalf("round-trip value = %q, want %q", got, value)
	}
}

// Verifies: SYS-REQ-115 (lenient traversal and deletion)
// SYS-REQ-115:nominal:nominal
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestConfigTraversalAndDelete(t *testing.T) {
	data := []byte(`{'items':['one','two'],'object':{'a':'A','b':'B'}}`)

	var items []string
	if _, err := Lenient.ArrayEach(data, func(value []byte, _ ValueType, _ int, err error) {
		if err != nil {
			t.Errorf("Lenient.ArrayEach callback error: %v", err)
			return
		}
		items = append(items, string(value))
	}, "items"); err != nil {
		t.Fatalf("Lenient.ArrayEach returned error: %v", err)
	}
	if want := []string{"one", "two"}; !reflect.DeepEqual(items, want) {
		t.Fatalf("Lenient.ArrayEach values = %v, want %v", items, want)
	}

	entries := make(map[string]string)
	err := Lenient.ObjectEach(data, func(key, value []byte, _ ValueType, _ int) error {
		if _, duplicate := entries[string(key)]; duplicate {
			return errors.New("duplicate key")
		}
		entries[string(key)] = string(value)
		return nil
	}, "object")
	if err != nil {
		t.Fatalf("Lenient.ObjectEach returned error: %v", err)
	}
	if want := map[string]string{"a": "A", "b": "B"}; !reflect.DeepEqual(entries, want) {
		t.Fatalf("Lenient.ObjectEach entries = %v, want %v", entries, want)
	}

	deleted := Lenient.Delete(data, "object", "a")
	if got, want := string(deleted), `{'items':['one','two'],'object':{'b':'B'}}`; got != want {
		t.Fatalf("Lenient.Delete = %s, want %s", got, want)
	}
}
