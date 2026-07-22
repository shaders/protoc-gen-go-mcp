package generator

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	mcpoptions "github.com/shaders/protoc-gen-go-mcp/pkg/options"
)

// buildServices compiles a file descriptor with the given services, each
// carrying the given tool options per method (nil entries mean "no
// annotation"), and returns every protogen method across all services.
func buildServices(t *testing.T, services map[string]map[string]*mcpoptions.ToolOptions) []*protogen.Method {
	t.Helper()

	sdps := make([]*descriptorpb.ServiceDescriptorProto, 0, len(services))
	for svcName, opts := range services {
		methods := make([]*descriptorpb.MethodDescriptorProto, 0, len(opts))
		for name, opt := range opts {
			mo := &descriptorpb.MethodOptions{}
			if opt != nil {
				proto.SetExtension(mo, mcpoptions.E_Tool, opt)
			}
			methods = append(methods, &descriptorpb.MethodDescriptorProto{
				Name:       proto.String(name),
				InputType:  proto.String(".test.pkg.Req"),
				OutputType: proto.String(".test.pkg.Resp"),
				Options:    mo,
			})
		}
		sdps = append(sdps, &descriptorpb.ServiceDescriptorProto{Name: proto.String(svcName), Method: methods})
	}

	fdp := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test/svc.proto"),
		Package: proto.String("test.pkg"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String("Req")},
			{Name: proto.String("Resp")},
		},
		Service: sdps,
		Options: &descriptorpb.FileOptions{GoPackage: proto.String("example.com/test/pkg;pkg")},
	}

	gen, err := protogen.Options{}.New(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: []string{"test/svc.proto"},
		ProtoFile:      []*descriptorpb.FileDescriptorProto{fdp},
	})
	if err != nil {
		t.Fatalf("protogen.New: %v", err)
	}
	var all []*protogen.Method
	for _, svc := range gen.Files[0].Services {
		all = append(all, svc.Methods...)
	}
	return all
}

// buildMethod compiles a single-service file descriptor carrying the given tool
// options (nil entries mean "no annotation") and returns the protogen methods.
func buildMethod(t *testing.T, opts map[string]*mcpoptions.ToolOptions) []*protogen.Method {
	t.Helper()
	return buildServices(t, map[string]map[string]*mcpoptions.ToolOptions{"Svc": opts})
}

func methodNamed(methods []*protogen.Method, name string) *protogen.Method {
	for _, m := range methods {
		if string(m.Desc.Name()) == name {
			return m
		}
	}
	return nil
}

// resolve runs resolveToolName for the named method, extracting its
// (mcp.options.tool) annotation the same way the generation loop does.
func resolve(g *FileGenerator, methods []*protogen.Method, name string) (string, error) {
	m := methodNamed(methods, name)
	return g.resolveToolName(m, methodToolOptions(m))
}

func TestResolveToolName_Strict_Happy(t *testing.T) {
	methods := buildMethod(t, map[string]*mcpoptions.ToolOptions{
		"GetItem": {Name: "get_item", Title: "Get item", ReadOnly: proto.Bool(true)},
	})
	g := &FileGenerator{requireToolAnnotation: true, seenToolNames: ToolNameRegistry{}}

	got, err := resolve(g, methods, "GetItem")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "get_item" {
		t.Fatalf("name = %q, want %q", got, "get_item")
	}
}

func TestResolveToolName_Strict_MissingAnnotation(t *testing.T) {
	methods := buildMethod(t, map[string]*mcpoptions.ToolOptions{"GetItem": nil})
	g := &FileGenerator{requireToolAnnotation: true, seenToolNames: ToolNameRegistry{}}

	_, err := resolve(g, methods, "GetItem")
	if err == nil {
		t.Fatal("expected error for missing annotation, got nil")
	}
	if !strings.Contains(err.Error(), "without a (mcp.options.tool) name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveToolName_Strict_EmptyName(t *testing.T) {
	methods := buildMethod(t, map[string]*mcpoptions.ToolOptions{
		"GetItem": {Title: "no name here"},
	})
	g := &FileGenerator{requireToolAnnotation: true, seenToolNames: ToolNameRegistry{}}

	if _, err := resolve(g, methods, "GetItem"); err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

func TestResolveToolName_InvalidName(t *testing.T) {
	for _, bad := range []string{"Get-Item", "GetItem", "1get", "a", "with space", strings.Repeat("x", 65)} {
		methods := buildMethod(t, map[string]*mcpoptions.ToolOptions{
			"GetItem": {Name: bad},
		})
		g := &FileGenerator{requireToolAnnotation: true, seenToolNames: ToolNameRegistry{}}
		if _, err := resolve(g, methods, "GetItem"); err == nil {
			t.Fatalf("expected error for invalid name %q, got nil", bad)
		}
	}
}

func TestResolveToolName_TitleLength(t *testing.T) {
	// 60 characters is the documented maximum; 61 must fail. Runes count,
	// not bytes.
	okTitle := strings.Repeat("x", 59) + "й"
	methods := buildMethod(t, map[string]*mcpoptions.ToolOptions{
		"GetItem": {Name: "get_item", Title: okTitle},
	})
	g := &FileGenerator{requireToolAnnotation: true, seenToolNames: ToolNameRegistry{}}
	if _, err := resolve(g, methods, "GetItem"); err != nil {
		t.Fatalf("60-character title should be accepted: %v", err)
	}

	methods = buildMethod(t, map[string]*mcpoptions.ToolOptions{
		"GetItem": {Name: "get_item", Title: strings.Repeat("x", 61)},
	})
	g = &FileGenerator{requireToolAnnotation: true, seenToolNames: ToolNameRegistry{}}
	_, err := resolve(g, methods, "GetItem")
	if err == nil {
		t.Fatal("expected error for 61-character title, got nil")
	}
	if !strings.Contains(err.Error(), "title of 61 characters") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveToolName_DuplicateName(t *testing.T) {
	methods := buildMethod(t, map[string]*mcpoptions.ToolOptions{
		"First":  {Name: "shared_name"},
		"Second": {Name: "shared_name"},
	})
	g := &FileGenerator{requireToolAnnotation: true, seenToolNames: ToolNameRegistry{}}

	if _, err := resolve(g, methods, "First"); err != nil {
		t.Fatalf("first resolve should succeed: %v", err)
	}
	_, err := resolve(g, methods, "Second")
	if err == nil {
		t.Fatal("expected duplicate-name error, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate MCP tool name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveToolName_NonStrictLegacyFallback(t *testing.T) {
	methods := buildMethod(t, map[string]*mcpoptions.ToolOptions{"GetItem": nil})
	g := &FileGenerator{requireToolAnnotation: false, seenToolNames: ToolNameRegistry{}}

	got, err := resolve(g, methods, "GetItem")
	if err != nil {
		t.Fatalf("non-strict should not error: %v", err)
	}
	if got != "test_pkg_Svc_GetItem" {
		t.Fatalf("legacy name = %q, want %q", got, "test_pkg_Svc_GetItem")
	}
}

func TestResolveToolName_AnnotatedCollidesWithLegacy(t *testing.T) {
	// The legacy mangle of test.pkg.svc.get is "test_pkg_svc_get", which is a
	// valid annotation name another method can claim.
	methods := buildServices(t, map[string]map[string]*mcpoptions.ToolOptions{
		"svc":   {"get": nil},
		"Other": {"GetItem": {Name: "test_pkg_svc_get"}},
	})
	g := &FileGenerator{requireToolAnnotation: false, seenToolNames: ToolNameRegistry{}}

	if _, err := resolve(g, methods, "get"); err != nil {
		t.Fatalf("legacy resolve should succeed: %v", err)
	}
	_, err := resolve(g, methods, "GetItem")
	if err == nil {
		t.Fatal("expected duplicate-name error for annotated name colliding with legacy name, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate MCP tool name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestOptionsProtoCopiesInSync guards against drift between the canonical
// options proto (vendored by consumers) and the testdata copy that
// pkg/options is generated from.
func TestOptionsProtoCopiesInSync(t *testing.T) {
	canonical, err := os.ReadFile("../../proto/mcp/options/options.proto")
	if err != nil {
		t.Fatalf("read canonical options.proto: %v", err)
	}
	testdata, err := os.ReadFile("../testdata/proto/mcp/options/options.proto")
	if err != nil {
		t.Fatalf("read testdata options.proto: %v", err)
	}
	if !bytes.Equal(canonical, testdata) {
		t.Fatal("proto/mcp/options/options.proto and pkg/testdata/proto/mcp/options/options.proto have drifted apart; copy the canonical file over the testdata one")
	}
}

func TestResolveToolName_LegacyLegacyCollisionStaysSilent(t *testing.T) {
	// test.pkg.a_b.c and test.pkg.a.b_c both mangle to "test_pkg_a_b_c".
	// Two unannotated methods keep the historic silent behavior.
	methods := buildServices(t, map[string]map[string]*mcpoptions.ToolOptions{
		"a_b": {"c": nil},
		"a":   {"b_c": nil},
	})
	g := &FileGenerator{requireToolAnnotation: false, seenToolNames: ToolNameRegistry{}}

	first, err := resolve(g, methods, "c")
	if err != nil {
		t.Fatalf("first legacy resolve should succeed: %v", err)
	}
	second, err := resolve(g, methods, "b_c")
	if err != nil {
		t.Fatalf("second legacy resolve should stay silent on collision: %v", err)
	}
	if first != second {
		t.Fatalf("expected identical mangled names, got %q and %q", first, second)
	}
}
