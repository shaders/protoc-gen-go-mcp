package runtime

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"
)

const pemKey = "-----BEGIN PRIVATE KEY-----\nMIIBpayload\n-----END PRIVATE KEY-----\n"

func TestNormalizeBase64BytesFields(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		message map[string]interface{}
		want    map[string]interface{}
	}{
		{
			name:    "raw string in bytes field is encoded",
			schema:  `{"properties":{"key_file":{"type":"string","format":"byte","contentEncoding":"base64"}}}`,
			message: map[string]interface{}{"key_file": pemKey},
			want:    map[string]interface{}{"key_file": base64.StdEncoding.EncodeToString([]byte(pemKey))},
		},
		{
			name:    "contentEncoding alone is enough",
			schema:  `{"properties":{"blob":{"type":"string","contentEncoding":"base64"}}}`,
			message: map[string]interface{}{"blob": "hello world"},
			want:    map[string]interface{}{"blob": "aGVsbG8gd29ybGQ="},
		},
		{
			name:    "already encoded value is left as is",
			schema:  `{"properties":{"blob":{"type":"string","format":"byte"}}}`,
			message: map[string]interface{}{"blob": "aGVsbG8gd29ybGQ="},
			want:    map[string]interface{}{"blob": "aGVsbG8gd29ybGQ="},
		},
		{
			name:    "unpadded base64-looking value is treated as raw",
			schema:  `{"properties":{"blob":{"type":"string","format":"byte"}}}`,
			message: map[string]interface{}{"blob": "abc"},
			want:    map[string]interface{}{"blob": "YWJj"},
		},
		{
			name:    "padded hex digest is indistinguishable from base64 and passes through",
			schema:  `{"properties":{"blob":{"type":"string","format":"byte"}}}`,
			message: map[string]interface{}{"blob": "d41d8cd98f00b204e9800998ecf8427e"},
			want:    map[string]interface{}{"blob": "d41d8cd98f00b204e9800998ecf8427e"},
		},
		{
			name:    "unpadded alphanumeric string is treated as raw",
			schema:  `{"properties":{"blob":{"type":"string","format":"byte"}}}`,
			message: map[string]interface{}{"blob": "deadbee"},
			want:    map[string]interface{}{"blob": "ZGVhZGJlZQ=="},
		},
		{
			name:    "string field is not touched",
			schema:  `{"properties":{"name":{"type":"string"}}}`,
			message: map[string]interface{}{"name": "some name"},
			want:    map[string]interface{}{"name": "some name"},
		},
		{
			name:   "inline object is marshaled then encoded",
			schema: `{"properties":{"service_account":{"type":"string","format":"byte"}}}`,
			message: map[string]interface{}{
				"service_account": map[string]interface{}{"type": "service_account", "project_id": "p1"},
			},
			want: map[string]interface{}{
				"service_account": base64.StdEncoding.EncodeToString(
					[]byte(`{"project_id":"p1","type":"service_account"}`)),
			},
		},
		{
			name:    "null value is left alone",
			schema:  `{"properties":{"blob":{"type":"string","format":"byte"}}}`,
			message: map[string]interface{}{"blob": nil},
			want:    map[string]interface{}{"blob": nil},
		},
		{
			name:   "nested message via $ref",
			schema: `{"$defs":{"Config":{"type":"object","properties":{"cert":{"type":"string","format":"byte"}}}},"properties":{"config":{"$ref":"#/$defs/Config","type":"object"}}}`,
			message: map[string]interface{}{
				"config": map[string]interface{}{"cert": "raw cert"},
			},
			want: map[string]interface{}{
				"config": map[string]interface{}{"cert": "cmF3IGNlcnQ="},
			},
		},
		{
			name:   "repeated bytes",
			schema: `{"properties":{"chunks":{"type":"array","items":{"type":"string","format":"byte"}}}}`,
			message: map[string]interface{}{
				"chunks": []interface{}{"one", "dHdv"},
			},
			want: map[string]interface{}{
				"chunks": []interface{}{"b25l", "dHdv"},
			},
		},
		{
			name:   "map of bytes via additionalProperties",
			schema: `{"properties":{"files":{"type":"object","propertyNames":{"type":"string"},"additionalProperties":{"type":"string","format":"byte"}}}}`,
			message: map[string]interface{}{
				"files": map[string]interface{}{"a.pem": "raw a", "b.pem": "cmF3IGI="},
			},
			want: map[string]interface{}{
				"files": map[string]interface{}{"a.pem": "cmF3IGE=", "b.pem": "cmF3IGI="},
			},
		},
		{
			name:   "bytes inside a oneOf variant",
			schema: `{"$defs":{"IOSConfiguration":{"type":"object","properties":{"ios_key_file":{"type":"string","format":"byte"}}}},"properties":{"kindOneOfType":{"type":"object","oneOf":[{"type":"object","properties":{"object_type":{"const":"ios_configuration","type":"string"},"ios_configuration":{"$ref":"#/$defs/IOSConfiguration","type":"object"}}},{"type":"object","properties":{"object_type":{"const":"android_configuration","type":"string"},"android_configuration":{"type":"object","properties":{"api_key":{"type":"string"}}}}}]}}}`,
			message: map[string]interface{}{
				"kindOneOfType": map[string]interface{}{
					"object_type":       "ios_configuration",
					"ios_configuration": map[string]interface{}{"ios_key_file": pemKey},
				},
			},
			want: map[string]interface{}{
				"kindOneOfType": map[string]interface{}{
					"object_type": "ios_configuration",
					"ios_configuration": map[string]interface{}{
						"ios_key_file": base64.StdEncoding.EncodeToString([]byte(pemKey)),
					},
				},
			},
		},
		{
			name:    "primitive bytes directly under a oneOf",
			schema:  `{"properties":{"payloadOneOfType":{"type":"object","oneOf":[{"type":"string","format":"byte"},{"type":"integer"}]}}}`,
			message: map[string]interface{}{"payloadOneOfType": "raw payload"},
			want:    map[string]interface{}{"payloadOneOfType": "cmF3IHBheWxvYWQ="},
		},
		{
			name:    "unknown field without schema is skipped",
			schema:  `{"properties":{"blob":{"type":"string","format":"byte"}}}`,
			message: map[string]interface{}{"blob": "raw", "extra": "kept"},
			want:    map[string]interface{}{"blob": "cmF3", "extra": "kept"},
		},
		{
			name:    "invalid schema JSON leaves the message untouched",
			schema:  `{not json`,
			message: map[string]interface{}{"blob": "raw"},
			want:    map[string]interface{}{"blob": "raw"},
		},
		{
			name:    "unresolvable $ref leaves the value untouched",
			schema:  `{"properties":{"config":{"$ref":"#/$defs/Missing","type":"object"}}}`,
			message: map[string]interface{}{"config": map[string]interface{}{"cert": "raw"}},
			want:    map[string]interface{}{"config": map[string]interface{}{"cert": "raw"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			NormalizeBase64BytesFields(tt.message, tt.schema)
			g.Expect(tt.message).To(Equal(tt.want))
		})
	}
}

func TestNormalizeBase64BytesFieldsIsIdempotent(t *testing.T) {
	g := NewWithT(t)

	const schema = `{"properties":{"key_file":{"type":"string","format":"byte"}}}`
	message := map[string]interface{}{"key_file": pemKey}

	NormalizeBase64BytesFields(message, schema)
	once := message["key_file"]
	NormalizeBase64BytesFields(message, schema)

	g.Expect(message["key_file"]).To(Equal(once))
	decoded, err := base64.StdEncoding.DecodeString(once.(string))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(decoded)).To(Equal(pemKey))
}

// A self-referencing $defs entry must not spin the walk forever.
func TestNormalizeBase64BytesFieldsRecursiveSchema(t *testing.T) {
	g := NewWithT(t)

	const schema = `{"$defs":{"Node":{"type":"object","properties":{"payload":{"type":"string","format":"byte"},"child":{"$ref":"#/$defs/Node","type":"object"}}}},"properties":{"root":{"$ref":"#/$defs/Node","type":"object"}}}`
	message := map[string]interface{}{
		"root": map[string]interface{}{
			"payload": "one",
			"child":   map[string]interface{}{"payload": "two"},
		},
	}

	NormalizeBase64BytesFields(message, schema)

	g.Expect(message).To(Equal(map[string]interface{}{
		"root": map[string]interface{}{
			"payload": "b25l",
			"child":   map[string]interface{}{"payload": "dHdv"},
		},
	}))
}

func TestNormalizeBase64BytesFieldsNoop(t *testing.T) {
	g := NewWithT(t)

	g.Expect(func() { NormalizeBase64BytesFields(nil, `{"properties":{}}`) }).ToNot(Panic())
	g.Expect(func() { NormalizeBase64BytesFields(map[string]interface{}{"a": "b"}, "") }).ToNot(Panic())
}

func TestLooksLikeBase64(t *testing.T) {
	g := NewWithT(t)

	for _, s := range []string{"", "aGVsbG8=", "aGVsbG8gd29ybGQ=", "-_-_", "AAAA"} {
		g.Expect(looksLikeBase64(s)).To(BeTrue(), "%q must be recognized as base64", s)
	}
	for _, s := range []string{"abc", pemKey, "hello world", `{"type":"service_account"}`, "a=b=c"} {
		g.Expect(looksLikeBase64(s)).To(BeFalse(), "%q must be treated as raw data", s)
	}
}

// protojson only accepts base64 for bytes fields, which is the failure this
// normalization exists to prevent; assert the output round-trips through it.
func TestNormalizedBytesDecodeAsJSONString(t *testing.T) {
	g := NewWithT(t)

	message := map[string]interface{}{"key_file": pemKey}
	NormalizeBase64BytesFields(message, `{"properties":{"key_file":{"type":"string","format":"byte"}}}`)

	marshaled, err := json.Marshal(message)
	g.Expect(err).ToNot(HaveOccurred())

	var out struct {
		KeyFile []byte `json:"key_file"`
	}
	g.Expect(json.Unmarshal(marshaled, &out)).To(Succeed())
	g.Expect(string(out.KeyFile)).To(Equal(pemKey))
}
