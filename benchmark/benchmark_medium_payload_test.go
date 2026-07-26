/*
Each test should process 2.4kb json record (based on Clearbit API)
It should read multiple nested fields and 1 array
*/
package benchmark

import (
	"encoding/json"
	"testing"

	"github.com/Jeffail/gabs"
	"github.com/a8m/djson"
	"github.com/antonholmquist/jason"
	"github.com/bitly/go-simplejson"
	"github.com/buger/jsonparser"
	jlexer "github.com/mailru/easyjson/jlexer"
	"github.com/mreiferson/go-ujson"
	"github.com/pquerna/ffjson/ffjson"
	"github.com/ugorji/go/codec"
	// "fmt"
	"bytes"
	"errors"
)

/*
   github.com/buger/jsonparser
*/
// Verifies: STK-REQ-001
// MCDC STK-REQ-001: N/A
// Verifies: STK-REQ-003
// MCDC STK-REQ-003: N/A
// Verifies: STK-REQ-004
// MCDC STK-REQ-004: N/A
// Verifies: STK-REQ-005
// MCDC STK-REQ-005: N/A
// Verifies: STK-REQ-007
// MCDC STK-REQ-007: N/A
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkJsonParserMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		jsonparser.Get(mediumFixture, "person", "name", "fullName")
		jsonparser.GetInt(mediumFixture, "person", "github", "followers")
		jsonparser.Get(mediumFixture, "company")

		jsonparser.ArrayEach(mediumFixture, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			jsonparser.Get(value, "url")
			nothing()
		}, "person", "gravatar", "avatars")
	}
}

// Verifies: STK-REQ-001
// MCDC STK-REQ-001: N/A
// Verifies: STK-REQ-003
// MCDC STK-REQ-003: N/A
// Verifies: STK-REQ-004
// MCDC STK-REQ-004: N/A
// Verifies: STK-REQ-005
// MCDC STK-REQ-005: N/A
// Verifies: STK-REQ-007
// MCDC STK-REQ-007: N/A
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkJsonParserDeleteMedium(b *testing.B) {
	fixture := make([]byte, 0, len(mediumFixture))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fixture = append(fixture[:0], mediumFixture...)
		fixture = jsonparser.Delete(fixture, "person", "name", "fullName")
		fixture = jsonparser.Delete(fixture, "person", "github", "followers")
		fixture = jsonparser.Delete(fixture, "company")

		nothing()
	}
}

// Verifies: STK-REQ-001
// MCDC STK-REQ-001: N/A
// Verifies: STK-REQ-003
// MCDC STK-REQ-003: N/A
// Verifies: STK-REQ-004
// MCDC STK-REQ-004: N/A
// Verifies: STK-REQ-005
// MCDC STK-REQ-005: N/A
// Verifies: STK-REQ-007
// MCDC STK-REQ-007: N/A
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkJsonParserEachKeyManualMedium(b *testing.B) {
	paths := [][]string{
		[]string{"person", "name", "fullName"},
		[]string{"person", "github", "followers"},
		[]string{"company"},
		[]string{"person", "gravatar", "avatars"},
	}

	for i := 0; i < b.N; i++ {
		jsonparser.EachKey(mediumFixture, func(idx int, value []byte, vt jsonparser.ValueType, err error) {
			switch idx {
			case 0:
			// jsonparser.ParseString(value)
			case 1:
				jsonparser.ParseInt(value)
			case 2:
			// jsonparser.ParseString(value)
			case 3:
				jsonparser.ArrayEach(value, func(avalue []byte, dataType jsonparser.ValueType, offset int, err error) {
					jsonparser.Get(avalue, "url")
				})
			}
		}, paths...)
	}
}

// Verifies: STK-REQ-001
// MCDC STK-REQ-001: N/A
// Verifies: STK-REQ-003
// MCDC STK-REQ-003: N/A
// Verifies: STK-REQ-004
// MCDC STK-REQ-004: N/A
// Verifies: STK-REQ-005
// MCDC STK-REQ-005: N/A
// Verifies: STK-REQ-007
// MCDC STK-REQ-007: N/A
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkJsonParserEachKeyStructMedium(b *testing.B) {
	paths := [][]string{
		[]string{"person", "name", "fullName"},
		[]string{"person", "github", "followers"},
		[]string{"company"},
		[]string{"person", "gravatar", "avatars"},
	}

	for i := 0; i < b.N; i++ {
		data := MediumPayload{
			Person: &CBPerson{
				Name:     &CBName{},
				Github:   &CBGithub{},
				Gravatar: &CBGravatar{},
			},
		}

		jsonparser.EachKey(mediumFixture, func(idx int, value []byte, vt jsonparser.ValueType, err error) {
			switch idx {
			case 0:
				data.Person.Name.FullName, _ = jsonparser.ParseString(value)
			case 1:
				v, _ := jsonparser.ParseInt(value)
				data.Person.Github.Followers = int(v)
			case 2:
				json.Unmarshal(value, &data.Company) // we don't have a JSON -> map[string]interface{} function yet, so use standard encoding/json here
			case 3:
				var avatars []*CBAvatar
				jsonparser.ArrayEach(value, func(avalue []byte, dataType jsonparser.ValueType, offset int, err error) {
					url, _ := jsonparser.ParseString(avalue)
					avatars = append(avatars, &CBAvatar{Url: url})
				})
				data.Person.Gravatar.Avatars = avatars
			}
		}, paths...)
	}
}

// Verifies: STK-REQ-001
// MCDC STK-REQ-001: N/A
// Verifies: STK-REQ-003
// MCDC STK-REQ-003: N/A
// Verifies: STK-REQ-004
// MCDC STK-REQ-004: N/A
// Verifies: STK-REQ-005
// MCDC STK-REQ-005: N/A
// Verifies: STK-REQ-007
// MCDC STK-REQ-007: N/A
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkJsonParserObjectEachStructMedium(b *testing.B) {
	nameKey, githubKey, gravatarKey := []byte("name"), []byte("github"), []byte("gravatar")
	errStop := errors.New("stop")

	for i := 0; i < b.N; i++ {
		data := MediumPayload{
			Person: &CBPerson{
				Name:     &CBName{},
				Github:   &CBGithub{},
				Gravatar: &CBGravatar{},
			},
		}

		missing := 3

		jsonparser.ObjectEach(mediumFixture, func(k, v []byte, vt jsonparser.ValueType, o int) error {
			switch {
			case bytes.Equal(k, nameKey):
				data.Person.Name.FullName, _ = jsonparser.GetString(v, "fullName")
				missing--
			case bytes.Equal(k, githubKey):
				x, _ := jsonparser.GetInt(v, "followers")
				data.Person.Github.Followers = int(x)
				missing--
			case bytes.Equal(k, gravatarKey):
				var avatars []*CBAvatar
				jsonparser.ArrayEach(v, func(avalue []byte, dataType jsonparser.ValueType, offset int, err error) {
					url, _ := jsonparser.ParseString(avalue)
					avatars = append(avatars, &CBAvatar{Url: url})
				}, "avatars")
				data.Person.Gravatar.Avatars = avatars
				missing--
			}

			if missing == 0 {
				return errStop
			} else {
				return nil
			}
		}, "person")

		cv, _, _, _ := jsonparser.Get(mediumFixture, "company")
		json.Unmarshal(cv, &data.Company)
	}
}

/*
   encoding/json
*/
// Verifies: STK-REQ-001
// MCDC STK-REQ-001: N/A
// Verifies: STK-REQ-003
// MCDC STK-REQ-003: N/A
// Verifies: STK-REQ-004
// MCDC STK-REQ-004: N/A
// Verifies: STK-REQ-005
// MCDC STK-REQ-005: N/A
// Verifies: STK-REQ-007
// MCDC STK-REQ-007: N/A
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkEncodingJsonStructMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var data MediumPayload
		json.Unmarshal(mediumFixture, &data)

		nothing(data.Person.Name.FullName, data.Person.Github.Followers, data.Company)

		for _, el := range data.Person.Gravatar.Avatars {
			nothing(el.Url)
		}
	}
}

// Verifies: STK-REQ-001
// MCDC STK-REQ-001: N/A
// Verifies: STK-REQ-003
// MCDC STK-REQ-003: N/A
// Verifies: STK-REQ-004
// MCDC STK-REQ-004: N/A
// Verifies: STK-REQ-005
// MCDC STK-REQ-005: N/A
// Verifies: STK-REQ-007
// MCDC STK-REQ-007: N/A
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkEncodingJsonInterfaceMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var data interface{}
		json.Unmarshal(mediumFixture, &data)
		m := data.(map[string]interface{})

		person := m["person"].(map[string]interface{})
		name := person["name"].(map[string]interface{})
		github := person["github"].(map[string]interface{})
		company := m["company"]
		gravatar := person["gravatar"].(map[string]interface{})
		avatars := gravatar["avatars"].([]interface{})

		nothing(name["fullName"].(string), github["followers"].(float64), company)
		for _, a := range avatars {
			nothing(a.(map[string]interface{})["url"])
		}
	}
}

/*
   github.com/Jeffail/gabs
*/
// Verifies: STK-REQ-001
// MCDC STK-REQ-001: N/A
// Verifies: STK-REQ-003
// MCDC STK-REQ-003: N/A
// Verifies: STK-REQ-004
// MCDC STK-REQ-004: N/A
// Verifies: STK-REQ-005
// MCDC STK-REQ-005: N/A
// Verifies: STK-REQ-007
// MCDC STK-REQ-007: N/A
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkGabsMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		json, _ := gabs.ParseJSON(mediumFixture)
		person := json.Path("person")
		nothing(
			person.Path("name.fullName").Data().(string),
			person.Path("github.followers").Data().(float64),
		)

		json.Path("company").ChildrenMap()

		arr, _ := person.Path("gravatar.avatars.url").Children()
		for _, el := range arr {
			nothing(el.String())
		}
	}
}

/*
   github.com/bitly/go-simplejson
*/
// Verifies: STK-REQ-001
// MCDC STK-REQ-001: N/A
// Verifies: STK-REQ-003
// MCDC STK-REQ-003: N/A
// Verifies: STK-REQ-004
// MCDC STK-REQ-004: N/A
// Verifies: STK-REQ-005
// MCDC STK-REQ-005: N/A
// Verifies: STK-REQ-007
// MCDC STK-REQ-007: N/A
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkGoSimpleJsonMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		json, _ := simplejson.NewJson(mediumFixture)
		person := json.Get("person")
		person.Get("name").Get("fullName").String()
		person.Get("github").Get("followers").Float64()
		json.Get("company")
		arr, _ := person.Get("gravatar").Get("avatars").Array()

		for _, el := range arr {
			nothing(el.(map[string]interface{})["url"])
		}
	}
}

/*
   github.com/pquerna/ffjson
*/
// Verifies: STK-REQ-001
// MCDC STK-REQ-001: N/A
// Verifies: STK-REQ-003
// MCDC STK-REQ-003: N/A
// Verifies: STK-REQ-004
// MCDC STK-REQ-004: N/A
// Verifies: STK-REQ-005
// MCDC STK-REQ-005: N/A
// Verifies: STK-REQ-007
// MCDC STK-REQ-007: N/A
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkFFJsonMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var data MediumPayload
		ffjson.Unmarshal(mediumFixture, &data)

		nothing(data.Person.Name.FullName, data.Person.Github.Followers, data.Company)

		for _, el := range data.Person.Gravatar.Avatars {
			nothing(el.Url)
		}
	}
}

/*
   github.com/bitly/go-simplejson
*/
// Verifies: STK-REQ-001
// MCDC STK-REQ-001: N/A
// Verifies: STK-REQ-003
// MCDC STK-REQ-003: N/A
// Verifies: STK-REQ-004
// MCDC STK-REQ-004: N/A
// Verifies: STK-REQ-005
// MCDC STK-REQ-005: N/A
// Verifies: STK-REQ-007
// MCDC STK-REQ-007: N/A
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkJasonMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		json, _ := jason.NewObjectFromBytes(mediumFixture)

		json.GetString("person.name.fullName")
		json.GetFloat64("person.github.followers")
		json.GetObject("company")
		arr, _ := json.GetObjectArray("person.gravatar.avatars")

		for _, el := range arr {
			el.GetString("url")
		}

		nothing()
	}
}

/*
   github.com/mreiferson/go-ujson
*/
// Verifies: STK-REQ-001
// MCDC STK-REQ-001: N/A
// Verifies: STK-REQ-003
// MCDC STK-REQ-003: N/A
// Verifies: STK-REQ-004
// MCDC STK-REQ-004: N/A
// Verifies: STK-REQ-005
// MCDC STK-REQ-005: N/A
// Verifies: STK-REQ-007
// MCDC STK-REQ-007: N/A
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkUjsonMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		json, _ := ujson.NewFromBytes(mediumFixture)

		person := json.Get("person")

		person.Get("name").Get("fullName").String()
		person.Get("github").Get("followers").Float64()
		json.Get("company").String()

		arr := person.Get("gravatar").Get("avatars").Array()
		for _, el := range arr {
			el.Get("url").String()
		}

		nothing()
	}
}

/*
   github.com/a8m/djson
*/
// Verifies: STK-REQ-001
// MCDC STK-REQ-001: N/A
// Verifies: STK-REQ-003
// MCDC STK-REQ-003: N/A
// Verifies: STK-REQ-004
// MCDC STK-REQ-004: N/A
// Verifies: STK-REQ-005
// MCDC STK-REQ-005: N/A
// Verifies: STK-REQ-007
// MCDC STK-REQ-007: N/A
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkDjsonMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m, _ := djson.DecodeObject(mediumFixture)
		person := m["person"].(map[string]interface{})
		name := person["name"].(map[string]interface{})
		github := person["github"].(map[string]interface{})
		company := m["company"]
		gravatar := person["gravatar"].(map[string]interface{})
		avatars := gravatar["avatars"].([]interface{})

		nothing(name["fullName"].(string), github["followers"].(float64), company)
		for _, a := range avatars {
			nothing(a.(map[string]interface{})["url"])
		}
	}
}

/*
   github.com/ugorji/go/codec
*/
// Verifies: STK-REQ-001
// MCDC STK-REQ-001: N/A
// Verifies: STK-REQ-003
// MCDC STK-REQ-003: N/A
// Verifies: STK-REQ-004
// MCDC STK-REQ-004: N/A
// Verifies: STK-REQ-005
// MCDC STK-REQ-005: N/A
// Verifies: STK-REQ-007
// MCDC STK-REQ-007: N/A
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkUgirjiMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		decoder := codec.NewDecoderBytes(mediumFixture, new(codec.JsonHandle))
		data := new(MediumPayload)
		json.Unmarshal(mediumFixture, &data)
		data.CodecDecodeSelf(decoder)

		nothing(data.Person.Name.FullName, data.Person.Github.Followers, data.Company)

		for _, el := range data.Person.Gravatar.Avatars {
			nothing(el.Url)
		}
	}
}

/*
   github.com/mailru/easyjson
*/
// Verifies: STK-REQ-001
// MCDC STK-REQ-001: N/A
// Verifies: STK-REQ-003
// MCDC STK-REQ-003: N/A
// Verifies: STK-REQ-004
// MCDC STK-REQ-004: N/A
// Verifies: STK-REQ-005
// MCDC STK-REQ-005: N/A
// Verifies: STK-REQ-007
// MCDC STK-REQ-007: N/A
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkEasyJsonMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		lexer := &jlexer.Lexer{Data: mediumFixture}
		data := new(MediumPayload)
		data.UnmarshalEasyJSON(lexer)

		nothing(data.Person.Name.FullName, data.Person.Github.Followers, data.Company)

		for _, el := range data.Person.Gravatar.Avatars {
			nothing(el.Url)
		}
	}
}
