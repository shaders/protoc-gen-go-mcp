package generator

import (
	"testing"

	testdatamcp "github.com/shaders/protoc-gen-go-mcp/pkg/testdata/gen/go/testdata/testdatamcp"
)

// TestNormalizeParsesJSONStringForValueField pins a deliberate side effect of
// google.protobuf.Value carrying the full JSON-type union: the union contains
// "object", so the generated NormalizeTopLevelJSONStrings re-parses a
// JSON-looking string argument for a Value-typed field into the parsed value
// (previously the field had no "type" at all and strings were left as-is).
func TestNormalizeParsesJSONStringForValueField(t *testing.T) {
	schema := testdatamcp.TestService_ProcessWellKnownTypesTool.JSONSchema

	m := map[string]interface{}{"config": `{"nested": {"a": 1}}`}
	if changed := testdatamcp.TestServiceNormalizeTopLevelJSONStrings(m, schema); !changed {
		t.Fatal("expected a JSON-object string for a Value field to be re-parsed")
	}
	obj, ok := m["config"].(map[string]interface{})
	if !ok {
		t.Fatalf("config = %T(%v), want parsed object", m["config"], m["config"])
	}
	if _, ok := obj["nested"]; !ok {
		t.Fatalf("parsed object lost content: %v", obj)
	}

	// A string that does not parse as JSON stays untouched.
	m = map[string]interface{}{"config": "plain string"}
	if changed := testdatamcp.TestServiceNormalizeTopLevelJSONStrings(m, schema); changed {
		t.Fatal("plain string must stay untouched")
	}
	if m["config"] != "plain string" {
		t.Fatalf("config = %v, want unchanged string", m["config"])
	}
}
