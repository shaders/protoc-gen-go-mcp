package generator

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/shaders/protoc-gen-go-mcp/pkg/runtime"
	testdata "github.com/shaders/protoc-gen-go-mcp/pkg/testdata/gen/go/testdata"
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

// TestBase64AlphabetValueKeepsProtojsonBehaviour pins the documented trade-off of
// looksLikeBase64: a value that is valid padded base64 by alphabet — a hex digest
// is the interesting case, being equally readable as raw text — is passed through,
// so protojson resolves it to the same bytes with and without normalization.
func TestBase64AlphabetValueKeepsProtojsonBehaviour(t *testing.T) {
	schema := testdatamcp.TestService_CreateItemTool.JSONSchema

	for _, digest := range []string{
		"deadbeef",
		"da39a3ee5e6b4b0d3255bfef95601890afd80709", // sha1 hex
		"d41d8cd98f00b204e9800998ecf8427e",         // md5 hex
	} {
		var withNormalization, without testdata.CreateItemRequest

		m := map[string]interface{}{"thumbnail": digest}
		runtime.NormalizeBase64BytesFields(m, schema)
		if m["thumbnail"] != digest {
			t.Fatalf("thumbnail = %v, want %q passed through", m["thumbnail"], digest)
		}

		payload := []byte(fmt.Sprintf(`{"thumbnail":%q}`, digest))
		if err := protojson.Unmarshal(payload, &withNormalization); err != nil {
			t.Fatalf("normalized %q: %v", digest, err)
		}
		if err := protojson.Unmarshal(payload, &without); err != nil {
			t.Fatalf("raw %q: %v", digest, err)
		}
		if !bytes.Equal(withNormalization.Thumbnail, without.Thumbnail) {
			t.Fatalf("%q decodes differently after normalization", digest)
		}
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
