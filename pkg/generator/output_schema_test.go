package generator

import (
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	. "github.com/onsi/gomega"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"google.golang.org/protobuf/types/pluginpb"

	mcpoptions "github.com/shaders/protoc-gen-go-mcp/pkg/options"
	"github.com/shaders/protoc-gen-go-mcp/pkg/runtime"
)

// buildResponseFile compiles a file whose Resp message covers every case where
// the response schema differs from the argument schema.
func buildResponseFile(t *testing.T) (*FileGenerator, *protogen.Message) {
	t.Helper()

	pageOpts := &descriptorpb.FieldOptions{}
	proto.SetExtension(pageOpts, mcpoptions.E_ZeroBasedPagination, true)
	field := func(name string, num int32, typ descriptorpb.FieldDescriptorProto_Type, label descriptorpb.FieldDescriptorProto_Label) *descriptorpb.FieldDescriptorProto {
		return &descriptorpb.FieldDescriptorProto{Name: proto.String(name), Number: proto.Int32(num), Type: typ.Enum(), Label: label.Enum(), JsonName: proto.String(name)}
	}
	optional := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	repeated := descriptorpb.FieldDescriptorProto_LABEL_REPEATED

	count := field("count", 1, descriptorpb.FieldDescriptorProto_TYPE_INT64, optional)
	ids := field("ids", 2, descriptorpb.FieldDescriptorProto_TYPE_UINT64, repeated)
	text := field("text", 3, descriptorpb.FieldDescriptorProto_TYPE_STRING, optional)
	text.OneofIndex = proto.Int32(0)
	sub := field("sub", 4, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, optional)
	sub.TypeName = proto.String(".test.pkg.Sub")
	sub.OneofIndex = proto.Int32(0)
	page := field("page", 5, descriptorpb.FieldDescriptorProto_TYPE_INT32, optional)
	page.Options = pageOpts
	wrapped := field("wrapped", 6, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, optional)
	wrapped.TypeName = proto.String(".google.protobuf.Int64Value")

	fdp := &descriptorpb.FileDescriptorProto{
		Name:       proto.String("test/resp.proto"),
		Package:    proto.String("test.pkg"),
		Syntax:     proto.String("proto3"),
		Dependency: []string{"google/protobuf/wrappers.proto", "mcp/options/options.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name:      proto.String("Resp"),
				Field:     []*descriptorpb.FieldDescriptorProto{count, ids, text, sub, page, wrapped},
				OneofDecl: []*descriptorpb.OneofDescriptorProto{{Name: proto.String("kind")}},
			},
			{Name: proto.String("Sub")},
		},
		Options: &descriptorpb.FileOptions{GoPackage: proto.String("example.com/test/pkg;pkg")},
	}

	gen, err := protogen.Options{}.New(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"test/resp.proto"},
		ProtoFile: []*descriptorpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(descriptorpb.File_google_protobuf_descriptor_proto),
			protodesc.ToFileDescriptorProto(wrapperspb.File_google_protobuf_wrappers_proto),
			protodesc.ToFileDescriptorProto(mcpoptions.File_mcp_options_options_proto),
			fdp,
		},
	})
	if err != nil {
		t.Fatalf("protogen.New: %v", err)
	}
	file := gen.FilesByPath["test/resp.proto"]
	return NewFileGenerator(file, gen), file.Messages[0]
}

func TestResponseSchemaDescribesProtojsonOutput(t *testing.T) {
	g := NewWithT(t)
	fg, msg := buildResponseFile(t)

	schema := fg.responseSchemaWithDefs(msg.Desc, msg)
	props := schema["properties"].(map[string]any)

	g.Expect(props["count"]).To(HaveKeyWithValue("type", "string"), "protojson writes int64 as a string")
	g.Expect(props["ids"].(map[string]any)["items"]).To(HaveKeyWithValue("type", "string"))
	g.Expect(props["wrapped"]).To(HaveKeyWithValue("type", "string"))
	g.Expect(props).To(HaveKey("text"), "oneof members are plain fields in the response")
	g.Expect(props).To(HaveKey("sub"))
	g.Expect(props).ToNot(HaveKey("kindOneOfType"))
	g.Expect(props["page"]).ToNot(HaveKey("minimum"), "responses are not shifted to 1-based pages")
	g.Expect(schema["required"]).To(BeEmpty())

	args := fg.messageSchemaWithDefs(msg.Desc, msg)
	argProps := args["properties"].(map[string]any)
	g.Expect(argProps).To(HaveKey("kindOneOfType"), "argument schema is untouched")
	g.Expect(argProps["count"]).To(HaveKeyWithValue("type", "integer"))
	g.Expect(argProps["page"]).To(HaveKeyWithValue("minimum", 1))
}

func TestToolMetaPublishesOutputSchema(t *testing.T) {
	g := NewWithT(t)

	g.Expect(runtime.ToolMeta(runtime.Tool{Name: "x"})).To(BeNil())

	tool := mcp.Tool{
		Name:           "x",
		RawInputSchema: json.RawMessage(`{"type":"object"}`),
		Meta:           runtime.ToolMeta(runtime.Tool{OutputSchema: `{"type":"object"}`}),
	}
	raw, err := json.Marshal(tool)
	g.Expect(err).ToNot(HaveOccurred())

	var decoded map[string]any
	g.Expect(json.Unmarshal(raw, &decoded)).To(Succeed())
	g.Expect(decoded).ToNot(HaveKey("outputSchema"))
	g.Expect(decoded["_meta"]).To(HaveKeyWithValue(runtime.OutputSchemaMetaKey, map[string]any{"type": "object"}))
}

func TestOneOfRequiredOrderIsStable(t *testing.T) {
	g := NewWithT(t)
	fg := &FileGenerator{}
	oneOf := map[string][]map[string]any{"zeta": nil, "alpha": nil, "mid": nil}

	for range 20 {
		required := fg.addOneOfConstraints(map[string]any{}, oneOf, nil)
		g.Expect(required).To(Equal([]string{"alphaOneOfType", "midOneOfType", "zetaOneOfType"}))
	}
}
