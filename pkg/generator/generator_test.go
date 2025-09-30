// Copyright 2025 Redpanda Data, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generator

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	testdata "github.com/shaders/protoc-gen-go-mcp/pkg/testdata/gen/go/testdata"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestGetTypeStandard(t *testing.T) {
	tests := []struct {
		name       string
		setupField func() protoreflect.FieldDescriptor
		wantSchema func(*WithT, map[string]any)
	}{
		// Map field tests
		{
			name: "map field in standard mode",
			setupField: func() protoreflect.FieldDescriptor {
				// Use the test proto's map field
				msg := &testdata.MapTestMessage{}
				return msg.ProtoReflect().Descriptor().Fields().ByName("string_map")
			},
			wantSchema: func(g *WithT, schema map[string]any) {
				g.Expect(schema["type"]).To(Equal("object"))
				g.Expect(schema).To(HaveKey("additionalProperties"))
				g.Expect(schema).To(HaveKey("propertyNames"))
			},
		},
		// Well-known types
		{
			name: "google.protobuf.Struct in standard mode",
			setupField: func() protoreflect.FieldDescriptor {
				msg := &testdata.WktTestMessage{}
				return msg.ProtoReflect().Descriptor().Fields().ByName("struct_field")
			},
			wantSchema: func(g *WithT, schema map[string]any) {
				g.Expect(schema["type"]).To(Equal("object"))
				g.Expect(schema["additionalProperties"]).To(Equal(true))
			},
		},
		{
			name: "google.protobuf.Value in standard mode",
			setupField: func() protoreflect.FieldDescriptor {
				msg := &testdata.WktTestMessage{}
				return msg.ProtoReflect().Descriptor().Fields().ByName("value_field")
			},
			wantSchema: func(g *WithT, schema map[string]any) {
				g.Expect(schema["description"]).To(ContainSubstring("dynamic JSON value"))
				g.Expect(schema).ToNot(HaveKey("type")) // Any type
			},
		},
		{
			name: "google.protobuf.ListValue in standard mode",
			setupField: func() protoreflect.FieldDescriptor {
				msg := &testdata.WktTestMessage{}
				return msg.ProtoReflect().Descriptor().Fields().ByName("list_value")
			},
			wantSchema: func(g *WithT, schema map[string]any) {
				g.Expect(schema["type"]).To(Equal("array"))
				g.Expect(schema).To(HaveKey("items"))
				g.Expect(schema["description"]).To(ContainSubstring("JSON array"))
			},
		},
		// Timestamp field
		{
			name: "timestamp field",
			setupField: func() protoreflect.FieldDescriptor {
				msg := &testdata.WktTestMessage{}
				return msg.ProtoReflect().Descriptor().Fields().ByName("timestamp")
			},
			wantSchema: func(g *WithT, schema map[string]any) {
				g.Expect(schema["type"]).To(Equal([]string{"string", "null"}))
				g.Expect(schema["format"]).To(Equal("date-time"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			fg := &FileGenerator{}

			field := tt.setupField()
			schema := fg.getType(field)

			tt.wantSchema(g, schema)
		})
	}
}

func TestMessageSchemaStandard(t *testing.T) {
	g := NewWithT(t)

	fg := &FileGenerator{}

	msgDesc := (&testdata.WktTestMessage{}).ProtoReflect().Descriptor()
	schema := fg.messageSchema(msgDesc)

	g.Expect(schema["type"]).To(Equal("object"))
	g.Expect(schema).To(HaveKey("properties"))
	g.Expect(schema).To(HaveKey("required"))
	// Object schemas should have additionalProperties: false
	g.Expect(schema).To(HaveKey("additionalProperties"))
	g.Expect(schema["additionalProperties"]).To(Equal(false))
}

func TestKindToType(t *testing.T) {
	tests := []struct {
		kind protoreflect.Kind
		want string
	}{
		{protoreflect.BoolKind, "boolean"},
		{protoreflect.StringKind, "string"},
		{protoreflect.Int32Kind, "integer"},
		{protoreflect.Int64Kind, "string"}, // encoded as string for safety
		{protoreflect.FloatKind, "number"},
		{protoreflect.DoubleKind, "number"},
		{protoreflect.BytesKind, "string"},
		{protoreflect.EnumKind, "string"},
	}

	for _, tt := range tests {
		t.Run(tt.kind.String(), func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(kindToType(tt.kind)).To(Equal(tt.want))
		})
	}
}

func TestSchemaMarshaling(t *testing.T) {
	g := NewWithT(t)

	fg := &FileGenerator{}

	// Test that generated schemas can be marshaled to JSON
	msg := &testdata.WktTestMessage{}
	schema := fg.messageSchema(msg.ProtoReflect().Descriptor())

	marshaled, err := json.Marshal(schema)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(marshaled).ToNot(BeEmpty())

	// Verify it's valid JSON
	var unmarshaled map[string]any
	err = json.Unmarshal(marshaled, &unmarshaled)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestOptionalKeywordSupport(t *testing.T) {
	tests := []struct {
		name                       string
		optionalKeywordSupport     bool
		fieldName                  string
		fieldHasOptionalKeyword    bool
		fieldHasRequiredAnnotation bool
		wantRequired               bool
	}{
		{
			name:                       "regular field without optional support",
			optionalKeywordSupport:     false,
			fieldName:                  "thumbnail", // bytes field from test proto
			fieldHasOptionalKeyword:    false,
			fieldHasRequiredAnnotation: false,
			wantRequired:               false,
		},
		{
			name:                       "regular field with optional support",
			optionalKeywordSupport:     true,
			fieldName:                  "thumbnail", // bytes field from test proto
			fieldHasOptionalKeyword:    false,
			fieldHasRequiredAnnotation: false,
			wantRequired:               true,
		},
		{
			name:                       "optional field without optional support",
			optionalKeywordSupport:     false,
			fieldName:                  "description",
			fieldHasOptionalKeyword:    true,
			fieldHasRequiredAnnotation: false,
			wantRequired:               false,
		},
		{
			name:                       "optional field with optional support",
			optionalKeywordSupport:     true,
			fieldName:                  "description",
			fieldHasOptionalKeyword:    true,
			fieldHasRequiredAnnotation: false,
			wantRequired:               false,
		},
		{
			name:                       "annotated required field without optional support",
			optionalKeywordSupport:     false,
			fieldName:                  "name",
			fieldHasOptionalKeyword:    false,
			fieldHasRequiredAnnotation: true,
			wantRequired:               true,
		},
		{
			name:                       "annotated required field with optional support",
			optionalKeywordSupport:     true,
			fieldName:                  "name",
			fieldHasOptionalKeyword:    false,
			fieldHasRequiredAnnotation: true,
			wantRequired:               true,
		},
		{
			name:                       "repeated field with optional support",
			optionalKeywordSupport:     true,
			fieldName:                  "tags", // repeated field from test proto
			fieldHasOptionalKeyword:    false,
			fieldHasRequiredAnnotation: false,
			wantRequired:               false,
		},
		{
			name:                       "map field with optional support",
			optionalKeywordSupport:     true,
			fieldName:                  "labels", // map field from test proto
			fieldHasOptionalKeyword:    false,
			fieldHasRequiredAnnotation: false,
			wantRequired:               false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			fg := &FileGenerator{optionalKeywordSupport: tt.optionalKeywordSupport}

			// Use CreateItemRequest from test data which has various field types
			msg := &testdata.CreateItemRequest{}
			msgDesc := msg.ProtoReflect().Descriptor()

			// Find the field we want to test
			var field protoreflect.FieldDescriptor
			for i := 0; i < msgDesc.Fields().Len(); i++ {
				fd := msgDesc.Fields().Get(i)
				if string(fd.Name()) == tt.fieldName {
					field = fd
					break
				}
			}

			// Skip test if field not found in the test message
			if field == nil {
				t.Skipf("Field %s not found in test message", tt.fieldName)
			}

			isRequired := fg.isFieldRequiredWithOptionalSupport(field)
			g.Expect(isRequired).To(Equal(tt.wantRequired),
				"Expected field %s to have required=%v with optionalKeywordSupport=%v",
				tt.fieldName, tt.wantRequired, tt.optionalKeywordSupport)
		})
	}
}

func TestMessageSchemaWithOptionalSupport(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name                   string
		optionalKeywordSupport bool
		wantRequiredFields     []string
	}{
		{
			name:                   "without optional keyword support",
			optionalKeywordSupport: false,
			wantRequiredFields:     []string{"name"}, // Only annotated field
		},
		{
			name:                   "with optional keyword support",
			optionalKeywordSupport: true,
			wantRequiredFields:     []string{"name", "thumbnail", "item_typeOneOfType"}, // All non-optional, non-repeated, non-map fields
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fg := &FileGenerator{optionalKeywordSupport: tt.optionalKeywordSupport}

			// Use CreateItemRequest which has mix of required, optional, and annotated fields
			msg := &testdata.CreateItemRequest{}
			schema := fg.messageSchemaFromDescriptor(msg.ProtoReflect().Descriptor(), nil)

			g.Expect(schema).To(HaveKey("required"))
			requiredFields, ok := schema["required"].([]string)
			g.Expect(ok).To(BeTrue(), "required field should be a string slice")

			// Check that all expected required fields are present
			for _, expectedField := range tt.wantRequiredFields {
				g.Expect(requiredFields).To(ContainElement(expectedField),
					"Field %s should be required with optionalKeywordSupport=%v",
					expectedField, tt.optionalKeywordSupport)
			}

			// In optional keyword support mode, these should NOT be required
			if tt.optionalKeywordSupport {
				g.Expect(requiredFields).ToNot(ContainElement("description"),
					"Optional field 'description' should not be required")
				g.Expect(requiredFields).ToNot(ContainElement("tags"),
					"Repeated field 'tags' should never be required")
				g.Expect(requiredFields).ToNot(ContainElement("labels"),
					"Map field 'labels' should never be required")
			}
		})
	}
}

var updateGolden = flag.Bool("update-golden", false, "Update golden files")

func TestFullGeneration(t *testing.T) {
	g := NewWithT(t)

	// Get current directory and change to testdata
	originalDir, err := os.Getwd()
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = os.Chdir(originalDir) }()

	testdataDir := filepath.Join("..", "testdata")
	err = os.Chdir(testdataDir)
	g.Expect(err).ToNot(HaveOccurred())

	if *updateGolden {
		// Generate golden files
		t.Logf("Generating golden files...")
		cmd := exec.Command("buf", "generate", "--template", "buf.gen.golden.yaml")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to generate golden files: %v\nOutput: %s", err, output)
		}

		// Also generate googleapis golden files
		t.Logf("Generating googleapis golden files...")
		cmd = exec.Command("buf", "generate", "buf.build/googleapis/googleapis", "--template", "buf.gen.golden.yaml")
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to generate googleapis golden files: %v\nOutput: %s", err, output)
		}

		t.Logf("Updated golden files")
		return
	}

	// Generate current files
	t.Logf("Generating current files...")
	cmd := exec.Command("buf", "generate")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to generate current files: %v\nOutput: %s", err, output)
	}

	// Also generate googleapis files
	t.Logf("Generating googleapis files...")
	cmd = exec.Command("buf", "generate", "buf.build/googleapis/googleapis")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to generate googleapis files: %v\nOutput: %s", err, output)
	}

	cmd = exec.Command("../../taskw", "fmt")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to format non-generated files: %v\nOutput: %s", err, output)
	}

	// Format generated files like the generate task does
	cmd = exec.Command("go", "run", "mvdan.cc/gofumpt@latest", "-l", "-w", ".")
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to format generated files: %v\nOutput: %s", err, output)
	}

	// Find all .pb.mcp.go files in gen/go and compare with golden/
	err = filepath.Walk("gen/go", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only check .pb.mcp.go files
		if !strings.HasSuffix(path, ".pb.mcp.go") {
			return nil
		}

		// Get corresponding golden file path
		goldenPath := strings.Replace(path, "gen/go/", "gen/go-golden/", 1)

		// Check that golden file exists
		if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
			t.Fatalf("Golden file %s missing\n", goldenPath)
		}

		// Read and compare files
		generatedContent, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		expectedContent, err := os.ReadFile(goldenPath)
		if err != nil {
			return err
		}

		// Compare content
		if !bytes.Equal(bytes.ReplaceAll(expectedContent, []byte("gen/go-golden"), []byte("gen/go")), generatedContent) {
			t.Errorf("Generated content differs from golden file.\n"+
				"Generated: %s\n"+
				"Golden: %s\n"+
				"To update golden files, run: go test -update-golden\n"+
				"Expected length: %d, Got length: %d",
				path, goldenPath,
				len(expectedContent), len(generatedContent))
		}

		return nil
	})

	g.Expect(err).ToNot(HaveOccurred())
}
