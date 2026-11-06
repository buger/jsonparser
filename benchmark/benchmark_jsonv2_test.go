//go:build go1.27 && goexperiment.jsonv2

/*
encoding/json/v2 benchmarks for the small/medium/large payloads.

encoding/json/v2 ships enabled by default in Go 1.27 (GOEXPERIMENT=jsonv2).
This file builds only on Go 1.27+ with the experiment on, so older toolchains
or GOEXPERIMENT=nojsonv2 skip it:

	go test -bench 'JsonV2' -benchmem ./benchmark/

The payload structs have no json tags. v1 matches field names
case-insensitively by default; v2 does not, so MatchCaseInsensitiveNames(true)
is passed to keep the comparison with encoding/json like-for-like.
*/
package benchmark

import (
	jsonv2 "encoding/json/v2"
	"testing"
)

var jsonV2Opts = jsonv2.MatchCaseInsensitiveNames(true)

// Guards against the benchmarks silently decoding nothing (for example if
// case-insensitive matching were dropped).
// reqproof:proptest:skip benchmark sanity check for an external decoder; not a jsonparser behavior
func TestJsonV2DecodesFixtures(t *testing.T) {
	var small encodingJSONSmallPayload
	if err := jsonv2.Unmarshal(smallFixture, &small, jsonV2Opts); err != nil {
		t.Fatalf("small: %v", err)
	}
	if small.Uuid == "" || small.Ua == "" || small.Tz == 0 || small.St == 0 {
		t.Fatalf("small: fields not populated: %+v", small)
	}

	var medium encodingJSONMediumPayload
	if err := jsonv2.Unmarshal(mediumFixture, &medium, jsonV2Opts); err != nil {
		t.Fatalf("medium: %v", err)
	}
	if medium.Person == nil || medium.Person.Name.FullName == "" ||
		medium.Person.Github.Followers == 0 || len(medium.Person.Gravatar.Avatars) == 0 {
		t.Fatalf("medium: fields not populated")
	}

	var large encodingJSONLargePayload
	if err := jsonv2.Unmarshal(largeFixture, &large, jsonV2Opts); err != nil {
		t.Fatalf("large: %v", err)
	}
	if len(large.Users) == 0 || large.Users[0].Username == "" ||
		large.Topics == nil || len(large.Topics.Topics) == 0 || large.Topics.Topics[0].Slug == "" {
		t.Fatalf("large: fields not populated")
	}
}

// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkEncodingJsonV2StructSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var data encodingJSONSmallPayload
		jsonv2.Unmarshal(smallFixture, &data, jsonV2Opts)

		nothing(data.Uuid, data.Tz, data.Ua, data.St)
	}
}

// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkEncodingJsonV2InterfaceSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var data any
		jsonv2.Unmarshal(smallFixture, &data)
		m := data.(map[string]any)

		nothing(m["uuid"].(string), m["tz"].(float64), m["ua"].(string), m["st"].(float64))
	}
}

// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkEncodingJsonV2StructMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var data encodingJSONMediumPayload
		jsonv2.Unmarshal(mediumFixture, &data, jsonV2Opts)

		nothing(data.Person.Name.FullName, data.Person.Github.Followers, data.Company)

		for _, el := range data.Person.Gravatar.Avatars {
			nothing(el.Url)
		}
	}
}

// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkEncodingJsonV2InterfaceMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var data any
		jsonv2.Unmarshal(mediumFixture, &data)
		m := data.(map[string]any)

		person := m["person"].(map[string]any)
		name := person["name"].(map[string]any)
		github := person["github"].(map[string]any)
		company := m["company"]
		gravatar := person["gravatar"].(map[string]any)
		avatars := gravatar["avatars"].([]any)

		nothing(name["fullName"].(string), github["followers"].(float64), company)
		for _, a := range avatars {
			nothing(a.(map[string]any)["url"])
		}
	}
}

// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkEncodingJsonV2StructLarge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var data encodingJSONLargePayload
		jsonv2.Unmarshal(largeFixture, &data, jsonV2Opts)

		for _, u := range data.Users {
			nothing(u.Username)
		}

		for _, t := range data.Topics.Topics {
			nothing(t.Id, t.Slug)
		}
	}
}

// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkEncodingJsonV2InterfaceLarge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var data any
		jsonv2.Unmarshal(largeFixture, &data)
		m := data.(map[string]any)

		users := m["users"].([]any)
		for _, u := range users {
			nothing(u.(map[string]any)["username"].(string))
		}

		topics := m["topics"].(map[string]any)["topics"].([]any)
		for _, t := range topics {
			tI := t.(map[string]any)
			nothing(tI["id"].(float64), tI["slug"].(string))
		}
	}
}
