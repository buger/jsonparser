package jsonparser

import (
	"bytes"
	"reflect"
	"testing"
)

// Verifies: SYS-REQ-114 (JSONPath compiled paths)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
// SYS-REQ-114:nominal:nominal
// SYS-REQ-114:malformed_input:nominal
func TestParsePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "dot notation",
			path: "person.name.fullName",
			want: []string{"person", "name", "fullName"},
		},
		{
			name: "optional root prefix",
			path: "$.person.name.fullName",
			want: []string{"person", "name", "fullName"},
		},
		{
			name: "array index",
			path: "users[0].name",
			want: []string{"users", "[0]", "name"},
		},
		{
			name: "root array and multiple brackets",
			path: "[0][1]",
			want: []string{"[0]", "[1]"},
		},
		{
			name: "root array with dollar",
			path: "$[0][1]",
			want: []string{"[0]", "[1]"},
		},
		{
			name: "wildcard",
			path: "data.items[*].id",
			want: []string{"data", "items", "[*]", "id"},
		},
		{
			name: "quoted key",
			path: `$."my key".value`,
			want: []string{"my key", "value"},
		},
		{
			name: "quoted syntax characters",
			path: `$."key.with[syntax]".value`,
			want: []string{"key.with[syntax]", "value"},
		},
		{
			name: "quoted escaped key",
			path: `$."say \"hello\"".value`,
			want: []string{`say "hello"`, "value"},
		},
		{
			name: "root",
			path: "$",
			want: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParsePath(test.path)
			if err != nil {
				t.Fatalf("ParsePath(%q) returned error: %v", test.path, err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("ParsePath(%q) = %#v, want %#v", test.path, got, test.want)
			}
		})
	}
}

// Verifies: SYS-REQ-114 (JSONPath compiled paths)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
// SYS-REQ-114:boundary:negative
// SYS-REQ-114:malformed_input:negative
func TestParsePathErrors(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "empty", path: ""},
		{name: "root prefix only", path: "$."},
		{name: "trailing dot", path: "person.name."},
		{name: "leading dot", path: ".person"},
		{name: "double dot", path: "person..name"},
		{name: "unclosed bracket", path: "users[0"},
		{name: "empty bracket", path: "users[]"},
		{name: "non numeric bracket", path: "users[first]"},
		{name: "malformed wildcard", path: "users[**]"},
		{name: "stray closing bracket", path: "users]"},
		{name: "unterminated quote", path: `$."my key.value`},
		{name: "characters after quote", path: `$."my key"value`},
		{name: "characters after bracket", path: "users[0]name"},
		{name: "dollar without separator", path: "$person.name"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got, err := ParsePath(test.path); err == nil {
				t.Fatalf("ParsePath(%q) = %#v, want error", test.path, got)
			}
		})
	}
}

// Verifies: SYS-REQ-001 (Get)
// Verifies: SYS-REQ-114 (JSONPath compiled paths)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestCompilePathGet(t *testing.T) {
	path, err := CompilePath("person.name")
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	tests := []struct {
		data string
		want string
	}{
		{data: `{"person":{"name":"Ada"}}`, want: "Ada"},
		{data: `{"person":{"name":"Grace"}}`, want: "Grace"},
	}

	for _, test := range tests {
		got, err := path.GetString([]byte(test.data))
		if err != nil {
			t.Fatalf("GetString(%s) returned error: %v", test.data, err)
		}
		if got != test.want {
			t.Fatalf("GetString(%s) = %q, want %q", test.data, got, test.want)
		}
	}
}

// Verifies: SYS-REQ-009 (Set)
// Verifies: SYS-REQ-114 (JSONPath compiled paths)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestCompilePathSet(t *testing.T) {
	path, err := CompilePath("users[0].name")
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	got, err := path.Set([]byte(`{"users":[{"name":"Ada"}]}`), []byte(`"Grace"`))
	if err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	want := []byte(`{"users":[{"name":"Grace"}]}`)
	if !bytes.Equal(got, want) {
		t.Fatalf("Set result = %s, want %s", got, want)
	}
}

// Verifies: SYS-REQ-114 (JSONPath compiled paths)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
// SYS-REQ-114:nominal:nominal
// SYS-REQ-114:boundary:nominal
// SYS-REQ-114:malformed_input:nominal
func TestCompilePathRoundTrip(t *testing.T) {
	data := []byte(`{"users":[{"profile":{"name":"Ada","age":36}}]}`)
	path, err := CompilePath("$.users[0].profile.age")
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	compiledValue, compiledType, compiledOffset, compiledErr := path.Get(data)
	directValue, directType, directOffset, directErr := Get(data, "users", "[0]", "profile", "age")
	if !bytes.Equal(compiledValue, directValue) ||
		compiledType != directType ||
		compiledOffset != directOffset ||
		compiledErr != directErr {
		t.Fatalf(
			"compiled Get = (%s, %v, %d, %v), direct Get = (%s, %v, %d, %v)",
			compiledValue, compiledType, compiledOffset, compiledErr,
			directValue, directType, directOffset, directErr,
		)
	}

	compiledSet, err := path.Set(data, []byte(`37`))
	if err != nil {
		t.Fatalf("compiled Set returned error: %v", err)
	}
	directSet, err := Set(data, []byte(`37`), "users", "[0]", "profile", "age")
	if err != nil {
		t.Fatalf("direct Set returned error: %v", err)
	}
	if !bytes.Equal(compiledSet, directSet) {
		t.Fatalf("compiled Set = %s, direct Set = %s", compiledSet, directSet)
	}
}

// Verifies: SYS-REQ-114 (JSONPath compiled paths)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestCompiledPathMethods(t *testing.T) {
	t.Run("GetInt", func(t *testing.T) {
		path, err := CompilePath("person.age")
		if err != nil {
			t.Fatal(err)
		}
		got, err := path.GetInt([]byte(`{"person":{"age":36}}`))
		if err != nil || got != 36 {
			t.Fatalf("GetInt = (%d, %v), want (36, nil)", got, err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		path, err := CompilePath("person.age")
		if err != nil {
			t.Fatal(err)
		}
		got := path.Delete([]byte(`{"person":{"name":"Ada","age":36}}`))
		want := []byte(`{"person":{"name":"Ada"}}`)
		if !bytes.Equal(got, want) {
			t.Fatalf("Delete result = %s, want %s", got, want)
		}
	})

	t.Run("ArrayEach", func(t *testing.T) {
		path, err := CompilePath("values")
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		count, err := path.ArrayEach([]byte(`{"values":[1,2,3]}`), func(value []byte, _ ValueType, _ int, err error) {
			if err != nil {
				t.Errorf("ArrayEach callback error: %v", err)
				return
			}
			got = append(got, string(value))
		})
		if err != nil {
			t.Fatalf("ArrayEach returned error: %v", err)
		}
		if count <= 0 || !reflect.DeepEqual(got, []string{"1", "2", "3"}) {
			t.Fatalf("ArrayEach = (%d, %#v), want three values", count, got)
		}
	})

	t.Run("EachKey", func(t *testing.T) {
		path, err := CompilePath("person.name")
		if err != nil {
			t.Fatal(err)
		}
		var got string
		err = path.EachKey([]byte(`{"person":{"name":"Ada"}}`), func(_ int, value []byte, _ ValueType, err error) {
			if err != nil {
				t.Errorf("EachKey callback error: %v", err)
				return
			}
			got = string(value)
		})
		if err != nil {
			t.Fatalf("EachKey returned error: %v", err)
		}
		if got != "Ada" {
			t.Fatalf("EachKey value = %q, want Ada", got)
		}
	})
}

// Verifies: SYS-REQ-114 (JSONPath compiled paths)
// reqproof:proptest:skip targeted witness/regression test; not a property-test subject
func TestCompiledPathPartsReturnsCopy(t *testing.T) {
	path, err := CompilePath("person.name")
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}

	parts := path.Parts()
	parts[0] = "changed"

	got, err := path.GetString([]byte(`{"person":{"name":"Ada"},"changed":{"name":"Grace"}}`))
	if err != nil {
		t.Fatalf("GetString returned error: %v", err)
	}
	if got != "Ada" {
		t.Fatalf("mutating Parts changed compiled path: got %q, want Ada", got)
	}
}

func BenchmarkParsePath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := ParsePath("users[0].profile.name"); err != nil {
			b.Fatal(err)
		}
	}
}
