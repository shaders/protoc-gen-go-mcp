package generator

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"

	"github.com/shaders/protoc-gen-go-mcp/pkg/runtime"
	testdatamcp "github.com/shaders/protoc-gen-go-mcp/pkg/testdata/gen/go/testdata/testdatamcp"
)

// TestGeneratedSchemaNormalizesBytesField runs the normalization against a real
// generated schema: CreateItemRequest.thumbnail is a proto bytes field, so a raw
// value the model wrote there must come out base64-encoded.
func TestGeneratedSchemaNormalizesBytesField(t *testing.T) {
	schema := testdatamcp.TestService_CreateItemTool.JSONSchema

	m := map[string]interface{}{"name": "item", "thumbnail": "not encoded yet"}
	runtime.NormalizeBase64BytesFields(m, schema)

	want := base64.StdEncoding.EncodeToString([]byte("not encoded yet"))
	if m["thumbnail"] != want {
		t.Fatalf("thumbnail = %v, want %v", m["thumbnail"], want)
	}
	if m["name"] != "item" {
		t.Fatalf("name = %v, want it untouched", m["name"])
	}
}

// TestGeneratedHandlerNormalizesBytesBeforeOneOfTransform pins the call order in
// the emitted handler: the bytes walk matches the message against the schema, so
// it has to run while the oneOf wrappers are still in place.
func TestGeneratedHandlerNormalizesBytesBeforeOneOfTransform(t *testing.T) {
	src, err := os.ReadFile("../testdata/gen/go/testdata/testdatamcp/test_service.pb.mcp.go")
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}

	normalizeAt := strings.Index(string(src), "runtime.NormalizeBase64BytesFields(message,")
	if normalizeAt < 0 {
		t.Fatal("generated handler does not call runtime.NormalizeBase64BytesFields")
	}

	transformAt := strings.Index(string(src), "TransformOneOfFields(message)")
	if transformAt < 0 {
		t.Fatal("generated handler does not call TransformOneOfFields")
	}

	if normalizeAt > transformAt {
		t.Fatal("NormalizeBase64BytesFields must be called before TransformOneOfFields")
	}
}
