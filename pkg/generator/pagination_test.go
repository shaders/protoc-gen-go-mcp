package generator

import (
	"testing"

	. "github.com/onsi/gomega"
	testdata "github.com/shaders/protoc-gen-go-mcp/pkg/testdata/gen/go/testdata"
)

func TestZeroBasedPaginationSchema(t *testing.T) {
	g := NewWithT(t)

	fg := &FileGenerator{}
	msgDesc := (&testdata.ListItemsRequest{}).ProtoReflect().Descriptor()

	pageField := msgDesc.Fields().ByName("page")
	g.Expect(pageField).ToNot(BeNil())

	g.Expect(isZeroBasedPagination(pageField)).To(BeTrue(),
		"page field must be detected as zero_based_pagination")

	pageSizeField := msgDesc.Fields().ByName("page_size")
	g.Expect(pageSizeField).ToNot(BeNil())
	g.Expect(isZeroBasedPagination(pageSizeField)).To(BeFalse(),
		"page_size must not be flagged as pagination")

	repeatedField := msgDesc.Fields().ByName("ignored_repeated_pages")
	g.Expect(repeatedField).ToNot(BeNil())
	g.Expect(isZeroBasedPagination(repeatedField)).To(BeFalse(),
		"annotation on repeated field must be ignored")

	stringField := msgDesc.Fields().ByName("ignored_string_page")
	g.Expect(stringField).ToNot(BeNil())
	g.Expect(isZeroBasedPagination(stringField)).To(BeFalse(),
		"annotation on non-integer field must be ignored")

	unsignedField := msgDesc.Fields().ByName("unsigned_page")
	g.Expect(unsignedField).ToNot(BeNil())
	g.Expect(isZeroBasedPagination(unsignedField)).To(BeTrue(),
		"annotation on uint32 field must be honored")

	defs := map[string]any{}
	visiting := map[string]bool{}
	pageSchema := fg.getTypeWithDefsAndComment(pageField, "Page number (0-based).", defs, visiting)

	g.Expect(pageSchema["minimum"]).To(Equal(1))
	g.Expect(pageSchema["description"]).To(ContainSubstring("1-based"))
	g.Expect(pageSchema["description"]).ToNot(ContainSubstring("0-based"))

	repeatedSchema := fg.getTypeWithDefsAndComment(repeatedField, "Annotation on a repeated field.", defs, visiting)
	g.Expect(repeatedSchema).ToNot(HaveKey("minimum"),
		"repeated field schema must not get minimum=1")
	g.Expect(repeatedSchema["type"]).To(Equal("array"))

	stringSchema := fg.getTypeWithDefsAndComment(stringField, "Annotation on a string field.", defs, visiting)
	g.Expect(stringSchema).ToNot(HaveKey("minimum"),
		"non-integer field schema must not get minimum=1")
	g.Expect(stringSchema["type"]).To(Equal("string"))
}

func TestCollectZeroBasedPaginationPaths(t *testing.T) {
	g := NewWithT(t)

	msgDesc := (&testdata.ListItemsRequest{}).ProtoReflect().Descriptor()
	paths := collectZeroBasedPaginationPaths(msgDesc)

	g.Expect(paths).To(ConsistOf(
		[]string{"page"},
		[]string{"query", "inner_page"},
		[]string{"unsigned_page"},
	))
}

func TestAdjustDescriptionForOneBased(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		contains string
	}{
		{"empty description", "", "Page number"},
		{"nil description", nil, "Page number"},
		{"contains 0-based", "Page index (0-based)", "1-based"},
		{"contains zero-based", "Page index, zero-based", "one-based"},
		{"generic description", "Some other doc", "1-based"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result := adjustDescriptionForOneBased(tt.input)
			g.Expect(result).To(ContainSubstring(tt.contains))
		})
	}
}
