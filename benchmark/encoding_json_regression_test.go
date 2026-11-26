package benchmark

import (
	"encoding/json"
	"testing"
)

// Verifies: STK-REQ-001 [regression, issue #126]
func TestEncodingJSONBenchmarkPayloadTypesUseReflection(t *testing.T) {
	payloads := []struct {
		name  string
		value interface{}
	}{
		{"small", (*encodingJSONSmallPayload)(nil)},
		{"medium", (*encodingJSONMediumPayload)(nil)},
		{"medium person", (*encodingJSONCBPerson)(nil)},
		{"medium name", (*encodingJSONCBName)(nil)},
		{"medium github", (*encodingJSONCBGithub)(nil)},
		{"medium gravatar", (*encodingJSONCBGravatar)(nil)},
		{"medium avatar", (*encodingJSONCBAvatar)(nil)},
		{"large", (*encodingJSONLargePayload)(nil)},
		{"large user", (*encodingJSONDSUser)(nil)},
		{"large topics list", (*encodingJSONDSTopicsList)(nil)},
		{"large topic", (*encodingJSONDSTopic)(nil)},
	}

	for _, payload := range payloads {
		t.Run(payload.name, func(t *testing.T) {
			if _, ok := payload.value.(json.Unmarshaler); ok {
				t.Fatalf("%T implements json.Unmarshaler; encoding/json benchmark would use generated code", payload.value)
			}
			if _, ok := payload.value.(json.Marshaler); ok {
				t.Fatalf("%T implements json.Marshaler; encoding/json benchmark would use generated code", payload.value)
			}
		})
	}
}
