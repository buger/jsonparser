/*
gjson (path-based, like jsonparser) and sonic (encoding/json drop-in)
benchmarks for the small/medium/large payloads.
*/
package benchmark

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/tidwall/gjson"
)

/*
   github.com/tidwall/gjson
*/
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkGjsonSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		gjson.GetBytes(smallFixture, "uuid")
		gjson.GetBytes(smallFixture, "tz").Int()
		gjson.GetBytes(smallFixture, "ua")
		gjson.GetBytes(smallFixture, "st").Int()
		nothing()
	}
}

// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkGjsonMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		gjson.GetBytes(mediumFixture, "person.name.fullName")
		gjson.GetBytes(mediumFixture, "person.github.followers").Int()
		gjson.GetBytes(mediumFixture, "company")

		gjson.GetBytes(mediumFixture, "person.gravatar.avatars").ForEach(func(_, value gjson.Result) bool {
			value.Get("url")
			return true
		})

		nothing()
	}
}

// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkGjsonLarge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		gjson.GetBytes(largeFixture, "users").ForEach(func(_, value gjson.Result) bool {
			value.Get("username")
			return true
		})

		gjson.GetBytes(largeFixture, "topics.topics").ForEach(func(_, value gjson.Result) bool {
			value.Get("id").Int()
			value.Get("slug")
			return true
		})

		nothing()
	}
}

/*
   github.com/bytedance/sonic
*/
// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkSonicSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var data encodingJSONSmallPayload
		sonic.Unmarshal(smallFixture, &data)
		nothing(data.Uuid, data.Tz, data.Ua, data.St)
	}
}

// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkSonicMedium(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var data encodingJSONMediumPayload
		sonic.Unmarshal(mediumFixture, &data)

		nothing(data.Person.Name.FullName, data.Person.Github.Followers, data.Company)

		for _, el := range data.Person.Gravatar.Avatars {
			nothing(el.Url)
		}
	}
}

// reqproof:proptest:skip performance benchmark; measures wall-clock time and allocations, output is non-deterministic and not comparable to an independent reference
func BenchmarkSonicLarge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var data encodingJSONLargePayload
		sonic.Unmarshal(largeFixture, &data)

		for _, u := range data.Users {
			nothing(u.Username)
		}

		for _, t := range data.Topics.Topics {
			nothing(t.Id, t.Slug)
		}
	}
}
