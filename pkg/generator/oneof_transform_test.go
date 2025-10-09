package generator

import (
	"encoding/json"
	"strings"
	"testing"
)

// transformOneOfFieldsRecursive is a copy of the generated function for testing
func transformOneOfFieldsRecursive(obj interface{}) {
	switch v := obj.(type) {
	case map[string]interface{}:
		// Transform oneOf fields in this object
		for key, value := range v {
			// Check if this looks like a oneOf discriminated union (must have OneOfType postfix)
			if strings.HasSuffix(key, "OneOfType") {
				if unionObj, ok := value.(map[string]interface{}); ok {
					if typeField, hasType := unionObj["object_type"]; hasType {
						if typeStr, ok := typeField.(string); ok {
							// First try to extract the field that matches the object_type
							// (for message types with $ref)
							if fieldValue, hasField := unionObj[typeStr]; hasField {
								// Move the field value directly to the parent level
								v[typeStr] = fieldValue
								delete(v, key)
							} else {
								// Fall back to old logic: create object without object_type field
								// (for primitive types or inline objects)
								variantObj := make(map[string]interface{})
								for k, val := range unionObj {
									if k != "object_type" {
										variantObj[k] = val
									}
								}
								// Replace the union object with the variant object
								v[typeStr] = variantObj
								delete(v, key)
							}
						}
					}
				}
			}
		}

		// Recursively process all values
		for _, value := range v {
			transformOneOfFieldsRecursive(value)
		}
	case []interface{}:
		// Process array elements
		for _, item := range v {
			transformOneOfFieldsRecursive(item)
		}
	}
}

func TestOneOfTransformation(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "transform oneOf with nested message",
			input: map[string]interface{}{
				"kindOneOfType": map[string]interface{}{
					"object_type": "device_data_applications",
					"device_data_applications": map[string]interface{}{
						"application_code": "test_app",
					},
				},
			},
			expected: map[string]interface{}{
				"device_data_applications": map[string]interface{}{
					"application_code": "test_app",
				},
			},
		},
		{
			name: "transform oneOf with scalar field",
			input: map[string]interface{}{
				"someOneOfType": map[string]interface{}{
					"object_type":  "string_value",
					"string_value": "hello",
				},
			},
			expected: map[string]interface{}{
				"string_value": "hello",
			},
		},
		{
			name: "nested oneOf transformation",
			input: map[string]interface{}{
				"outer": map[string]interface{}{
					"innerOneOfType": map[string]interface{}{
						"object_type": "option_a",
						"option_a": map[string]interface{}{
							"value": "test",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"outer": map[string]interface{}{
					"option_a": map[string]interface{}{
						"value": "test",
					},
				},
			},
		},
		{
			name: "fallback to old logic when field doesn't match object_type",
			input: map[string]interface{}{
				"primitiveOneOfType": map[string]interface{}{
					"object_type": "string_option",
					"value":       "hello world",
					"extra_field": 123,
				},
			},
			expected: map[string]interface{}{
				"string_option": map[string]interface{}{
					"value":       "hello world",
					"extra_field": 123,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of input to avoid modifying the original
			input := deepCopyMap(tt.input)

			// Apply the transformation
			transformOneOfFieldsRecursive(input)

			// Compare the result
			inputJSON, _ := json.MarshalIndent(input, "", "  ")
			expectedJSON, _ := json.MarshalIndent(tt.expected, "", "  ")

			if string(inputJSON) != string(expectedJSON) {
				t.Errorf("OneOf transformation failed\nGot:\n%s\nExpected:\n%s", inputJSON, expectedJSON)
			}
		})
	}
}

func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = deepCopyMap(val)
		case []interface{}:
			arr := make([]interface{}, len(val))
			copy(arr, val)
			result[k] = arr
		default:
			result[k] = v
		}
	}
	return result
}
