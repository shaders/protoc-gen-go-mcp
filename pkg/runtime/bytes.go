package runtime

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// maxBase64WalkDepth bounds the schema walk. Request schemas are shallow; the cap
// only guards against a pathological $ref chain that never descends into a value.
const maxBase64WalkDepth = 64

// NormalizeBase64BytesFields walks message against toolSchema and base64-encodes
// every value sitting where the schema declares a protobuf bytes field
// ("format":"byte" or "contentEncoding":"base64"). LLM clients routinely put raw
// content there — a service-account JSON, a PEM key — which protojson then
// rejects with `invalid value for bytes field X`.
//
// Idempotent: a value that already decodes as base64 is left untouched. Call it
// before the oneOf transform, while message still mirrors the schema shape.
func NormalizeBase64BytesFields(message map[string]interface{}, toolSchema string) {
	if len(message) == 0 || toolSchema == "" {
		return
	}
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(toolSchema), &schema); err != nil {
		return
	}
	defs, _ := schema["$defs"].(map[string]interface{})
	normalizeBase64Value(message, schema, defs, 0)
}

// normalizeBase64Value returns value with its bytes-field leaves encoded. It
// returns a value instead of only mutating in place so that a scalar rewrite
// found inside a oneOf variant propagates back to the parent.
func normalizeBase64Value(value interface{}, schema, defs map[string]interface{}, depth int) interface{} {
	if value == nil || schema == nil || depth > maxBase64WalkDepth {
		return value
	}

	if ref, ok := schema["$ref"].(string); ok && ref != "" {
		if resolved := resolveLocalDef(ref, defs); resolved != nil {
			return normalizeBase64Value(value, resolved, defs, depth+1)
		}
		return value
	}

	if isBase64Schema(schema) {
		return ensureBase64(value)
	}

	// A union describes the same value from several angles, so every variant gets
	// a walk; at most one of them matches the value actually supplied.
	for _, key := range []string{"oneOf", "anyOf"} {
		variants, _ := schema[key].([]interface{})
		for _, v := range variants {
			if variant, ok := v.(map[string]interface{}); ok {
				value = normalizeBase64Value(value, variant, defs, depth+1)
			}
		}
	}

	switch v := value.(type) {
	case map[string]interface{}:
		props, _ := schema["properties"].(map[string]interface{})
		for key, raw := range v {
			fieldSchema, ok := props[key].(map[string]interface{})
			if !ok {
				// map<string, bytes> declares its value schema here instead.
				fieldSchema, ok = schema["additionalProperties"].(map[string]interface{})
				if !ok {
					continue
				}
			}
			v[key] = normalizeBase64Value(raw, fieldSchema, defs, depth+1)
		}
	case []interface{}:
		items, ok := schema["items"].(map[string]interface{})
		if !ok {
			return value
		}
		for i := range v {
			v[i] = normalizeBase64Value(v[i], items, defs, depth+1)
		}
	}

	return value
}

// resolveLocalDef resolves "#/$defs/Name" against defs. Anything else — a remote
// URL, an arbitrary JSON Pointer — is not resolved.
func resolveLocalDef(ref string, defs map[string]interface{}) map[string]interface{} {
	const prefix = "#/$defs/"
	if !strings.HasPrefix(ref, prefix) {
		return nil
	}
	def, _ := defs[strings.TrimPrefix(ref, prefix)].(map[string]interface{})
	return def
}

func isBase64Schema(schema map[string]interface{}) bool {
	if format, ok := schema["format"].(string); ok && format == "byte" {
		return true
	}
	encoding, ok := schema["contentEncoding"].(string)
	return ok && encoding == "base64"
}

// ensureBase64 returns a base64 form of v when v looks like raw, not-yet-encoded
// data; otherwise v unchanged.
func ensureBase64(v interface{}) interface{} {
	switch x := v.(type) {
	case nil:
		return v
	case string:
		if looksLikeBase64(x) {
			return x
		}
		return base64.StdEncoding.EncodeToString([]byte(x))
	default:
		// Objects and arrays are inline content the model wrote out (a
		// service-account JSON, typically); encode the canonical text form, as a
		// compliant client would have sent it.
		b, err := json.Marshal(v)
		if err != nil {
			return v
		}
		return base64.StdEncoding.EncodeToString(b)
	}
}

// looksLikeBase64 reports whether s uses only the base64 alphabet (standard or
// URL-safe) and decodes cleanly. Empty counts as encoded. Padding is required:
// the unpadded encodings accept short raw words like "abc" and would leave them
// unencoded.
func looksLikeBase64(s string) bool {
	if s == "" {
		return true
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z':
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		case c == '+', c == '/', c == '=':
		case c == '-', c == '_':
		default:
			return false
		}
	}
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.URLEncoding} {
		if _, err := enc.DecodeString(s); err == nil {
			return true
		}
	}
	return false
}
