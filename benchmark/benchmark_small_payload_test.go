/*
Each test should process 190 byte http log like json record
It should read multiple fields
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

// Just for emulating field access, so it will not throw "evaluated but not used"
// Benchmark helper for STK-REQ-001, STK-REQ-003, STK-REQ-004, STK-REQ-005, and STK-REQ-007.
// reqproof:proptest:skip no-op benchmark helper; performs no work, returns immediately, no behavioral variance to property-test
func nothing(_ ...interface{}) {}

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
func BenchmarkJsonParserSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		jsonparser.Get(smallFixture, "uuid")
		jsonparser.GetInt(smallFixture, "tz")
		jsonparser.Get(smallFixture, "ua")
		jsonparser.GetInt(smallFixture, "st")

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
func BenchmarkJsonParserEachKeyManualSmall(b *testing.B) {
	paths := [][]string{
		[]string{"uuid"},
		[]string{"tz"},
		[]string{"ua"},
		[]string{"st"},
	}

	for i := 0; i < b.N; i++ {
		jsonparser.EachKey(smallFixture, func(idx int, value []byte, vt jsonparser.ValueType, err error) {
			switch idx {
			case 0:
				// jsonparser.ParseString(value)
			case 1:
				jsonparser.ParseInt(value)
			case 2:
				// jsonparser.ParseString(value)
			case 3:
				jsonparser.ParseInt(value)
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
func BenchmarkJsonParserEachKeyStructSmall(b *testing.B) {
	paths := [][]string{
		[]string{"uuid"},
		[]string{"tz"},
		[]string{"ua"},
		[]string{"st"},
	}

	for i := 0; i < b.N; i++ {
		var data SmallPayload

		jsonparser.EachKey(smallFixture, func(idx int, value []byte, vt jsonparser.ValueType, err error) {
			switch idx {
			case 0:
				data.Uuid, _ = jsonparser.ParseString(value)
			case 1:
				v, _ := jsonparser.ParseInt(value)
				data.Tz = int(v)
			case 2:
				data.Ua, _ = jsonparser.ParseString(value)
			case 3:
				v, _ := jsonparser.ParseInt(value)
				data.St = int(v)
			}
		}, paths...)

		nothing(data.Uuid, data.Tz, data.Ua, data.St)
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
func BenchmarkJsonParserObjectEachStructSmall(b *testing.B) {
	uuidKey, tzKey, uaKey, stKey := []byte("uuid"), []byte("tz"), []byte("ua"), []byte("st")
	errStop := errors.New("stop")

	for i := 0; i < b.N; i++ {
		var data SmallPayload

		missing := 4

		jsonparser.ObjectEach(smallFixture, func(key, value []byte, vt jsonparser.ValueType, off int) error {
			switch {
			case bytes.Equal(key, uuidKey):
				data.Uuid, _ = jsonparser.ParseString(value)
				missing--
			case bytes.Equal(key, tzKey):
				v, _ := jsonparser.ParseInt(value)
				data.Tz = int(v)
				missing--
			case bytes.Equal(key, uaKey):
				data.Ua, _ = jsonparser.ParseString(value)
				missing--
			case bytes.Equal(key, stKey):
				v, _ := jsonparser.ParseInt(value)
				data.St = int(v)
				missing--
			}

			if missing == 0 {
				return errStop
			} else {
				return nil
			}
		})

		nothing(data.Uuid, data.Tz, data.Ua, data.St)
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
func BenchmarkJsonParserSetSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		jsonparser.Set(smallFixture, []byte(`"c90927dd-1588-4fe7-a14f-8a8950cfcbd8"`), "uuid")
		jsonparser.Set(smallFixture, []byte("-3"), "tz")
		jsonparser.Set(smallFixture, []byte(`"server_agent"`), "ua")
		jsonparser.Set(smallFixture, []byte("3"), "st")

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
func BenchmarkJsonParserDelSmall(b *testing.B) {
	fixture := make([]byte, 0, len(smallFixture))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fixture = append(fixture[:0], smallFixture...)
		fixture = jsonparser.Delete(fixture, "uuid")
		fixture = jsonparser.Delete(fixture, "tz")
		fixture = jsonparser.Delete(fixture, "ua")
		fixture = jsonparser.Delete(fixture, "stt")

		nothing()
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
func BenchmarkEncodingJsonStructSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var data encodingJSONSmallPayload
		json.Unmarshal(smallFixture, &data)

		nothing(data.Uuid, data.Tz, data.Ua, data.St)
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
func BenchmarkGoSimplejsonSetSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		json, _ := simplejson.NewJson(smallFixture)

		json.SetPath([]string{"uuid"}, "c90927dd-1588-4fe7-a14f-8a8950cfcbd8")
		json.SetPath([]string{"tz"}, -3)
		json.SetPath([]string{"ua"}, "server_agent")
		json.SetPath([]string{"st"}, 3)

		nothing()
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
func BenchmarkUjsonSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		json, _ := ujson.NewFromBytes(smallFixture)

		json.Get("uuid").String()
		json.Get("tz").Float64()
		json.Get("ua").String()
		json.Get("st").Float64()

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
func BenchmarkDjsonSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m, _ := djson.DecodeObject(smallFixture)
		nothing(m["uuid"].(string), m["tz"].(float64), m["ua"].(string), m["st"].(float64))
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
func BenchmarkUgirjiSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		decoder := codec.NewDecoderBytes(smallFixture, new(codec.JsonHandle))
		data := new(SmallPayload)
		data.CodecDecodeSelf(decoder)

		nothing(data.Uuid, data.Tz, data.Ua, data.St)
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
func BenchmarkEasyJsonSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		lexer := &jlexer.Lexer{Data: smallFixture}
		data := new(SmallPayload)
		data.UnmarshalEasyJSON(lexer)

		nothing(data.Uuid, data.Tz, data.Ua, data.St)
	}
}
