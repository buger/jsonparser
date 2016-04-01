/*
   Each test should process 2.4kb json record (based on Clearbit API)
   It should read multiple nested fields and 1 array
*/
package benchmark

import (
	"encoding/json"
	"github.com/Jeffail/gabs"
	"github.com/antonholmquist/jason"
	"github.com/bitly/go-simplejson"
	"github.com/buger/jsonparser"
	jlexer "github.com/mailru/easyjson/jlexer"
	"github.com/mreiferson/go-ujson"
	"github.com/pquerna/ffjson/ffjson"
	"github.com/ugorji/go/codec"
	"testing"
	// "fmt"
)

/*
   github.com/buger/jsonparser
*/
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

/*
   encoding/json
*/
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
   github.com/ugorji/go/codec
*/
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
