/*
   Each test should process 190 byte http log like json record
   It should read multiple fields
*/
package benchmark

import (
	"encoding/json"
	"github.com/Jeffail/gabs"
	"github.com/antonholmquist/jason"
	"github.com/bitly/go-simplejson"
	"github.com/buger/jsonparser"
	"github.com/pquerna/ffjson/ffjson"
	"testing"
	// "fmt"
)

// Just for emulating field access, so it will not throw "evaluated but not used"
func nothing(_ ...interface{}) {}

/*
   github.com/buger/jsonparser
*/
func BenchmarkJsonParserSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		jsonparser.Get(smallFixture, "uuid")
		jsonparser.GetNumber(smallFixture, "tz")
		jsonparser.Get(smallFixture, "ua")
		jsonparser.GetNumber(smallFixture, "st")

		nothing()
	}
}

/*
   encoding/json
*/
func BenchmarkEncodingJsonStructSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var data SmallPayload
		json.Unmarshal(smallFixture, &data)

		nothing(data.Uuid, data.Tz, data.Ua, data.St)
	}
}

func BenchmarkEncodingJsonInterfaceSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var data interface{}
		json.Unmarshal(smallFixture, &data)
		m := data.(map[string]interface{})

		nothing(m["uuid"].(string), m["tz"].(float64), m["ua"].(string), m["st"].(float64))
	}
}

/*
   github.com/Jeffail/gabs
*/

func BenchmarkGabsSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		json, _ := gabs.ParseJSON(smallFixture)

		nothing(
			json.Path("uuid").Data().(string),
			json.Path("tz").Data().(float64),
			json.Path("ua").Data().(string),
			json.Path("st").Data().(float64),
		)
	}
}

/*
   github.com/bitly/go-simplejson
*/

func BenchmarkGoSimplejsonSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		json, _ := simplejson.NewJson(smallFixture)

		json.Get("uuid").String()
		json.Get("tz").Float64()
		json.Get("ua").String()
		json.Get("st").Float64()

		nothing()
	}
}

/*
   github.com/pquerna/ffjson
*/

func BenchmarkFFJsonSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var data SmallPayload
		ffjson.Unmarshal(smallFixture, &data)

		nothing(data.Uuid, data.Tz, data.Ua, data.St)
	}
}

/*
   github.com/bitly/go-simplejson
*/

func BenchmarkJasonSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		json, _ := jason.NewObjectFromBytes(smallFixture)

		json.GetString("uuid")
		json.GetFloat64("tz")
		json.GetString("ua")
		json.GetFloat64("st")

		nothing()
	}
}
