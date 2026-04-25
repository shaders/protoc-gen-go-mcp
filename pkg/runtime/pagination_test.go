package runtime

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"
)

func TestAdjustZeroBasedPaginationFields(t *testing.T) {
	tests := []struct {
		name    string
		message map[string]interface{}
		paths   [][]string
		want    map[string]interface{}
	}{
		{
			name:    "decrement top level page from 1 to 0",
			message: map[string]interface{}{"page": float64(1)},
			paths:   [][]string{{"page"}},
			want:    map[string]interface{}{"page": float64(0)},
		},
		{
			name:    "decrement top level page from 5 to 4",
			message: map[string]interface{}{"page": float64(5)},
			paths:   [][]string{{"page"}},
			want:    map[string]interface{}{"page": float64(4)},
		},
		{
			name:    "page=0 stays 0 (clamp, no negatives)",
			message: map[string]interface{}{"page": float64(0)},
			paths:   [][]string{{"page"}},
			want:    map[string]interface{}{"page": float64(0)},
		},
		{
			name:    "negative input clamps to 0",
			message: map[string]interface{}{"page": float64(-3)},
			paths:   [][]string{{"page"}},
			want:    map[string]interface{}{"page": float64(0)},
		},
		{
			name:    "missing field is left untouched",
			message: map[string]interface{}{"other": "ok"},
			paths:   [][]string{{"page"}},
			want:    map[string]interface{}{"other": "ok"},
		},
		{
			name: "nested path is decremented",
			message: map[string]interface{}{
				"query": map[string]interface{}{"inner_page": float64(2)},
			},
			paths: [][]string{{"query", "inner_page"}},
			want: map[string]interface{}{
				"query": map[string]interface{}{"inner_page": float64(1)},
			},
		},
		{
			name:    "nested path with missing parent is no-op",
			message: map[string]interface{}{"page_size": float64(10)},
			paths:   [][]string{{"query", "inner_page"}},
			want:    map[string]interface{}{"page_size": float64(10)},
		},
		{
			name:    "non-numeric value is left untouched",
			message: map[string]interface{}{"page": "first"},
			paths:   [][]string{{"page"}},
			want:    map[string]interface{}{"page": "first"},
		},
		{
			name:    "nil message is no-op",
			message: nil,
			paths:   [][]string{{"page"}},
			want:    nil,
		},
		{
			name: "multiple paths processed independently",
			message: map[string]interface{}{
				"page": float64(3),
				"nested": map[string]interface{}{
					"page": float64(7),
				},
			},
			paths: [][]string{{"page"}, {"nested", "page"}},
			want: map[string]interface{}{
				"page": float64(2),
				"nested": map[string]interface{}{
					"page": float64(6),
				},
			},
		},
		{
			name: "int values are decremented",
			message: map[string]interface{}{
				"page":   3,
				"page64": int64(4),
				"page32": int32(2),
			},
			paths: [][]string{{"page"}, {"page64"}, {"page32"}},
			want: map[string]interface{}{
				"page":   2,
				"page64": int64(3),
				"page32": int32(1),
			},
		},
		{
			name: "uint values are decremented and clamped at 0",
			message: map[string]interface{}{
				"u":   uint(3),
				"u32": uint32(1),
				"u64": uint64(0),
			},
			paths: [][]string{{"u"}, {"u32"}, {"u64"}},
			want: map[string]interface{}{
				"u":   uint(2),
				"u32": uint32(0),
				"u64": uint64(0),
			},
		},
		{
			name: "float32 is decremented",
			message: map[string]interface{}{
				"page": float32(2),
			},
			paths: [][]string{{"page"}},
			want: map[string]interface{}{
				"page": float32(1),
			},
		},
		{
			name: "json.Number integer is decremented",
			message: map[string]interface{}{
				"page": json.Number("5"),
			},
			paths: [][]string{{"page"}},
			want: map[string]interface{}{
				"page": json.Number("4"),
			},
		},
		{
			name: "json.Number zero is clamped",
			message: map[string]interface{}{
				"page": json.Number("0"),
			},
			paths: [][]string{{"page"}},
			want: map[string]interface{}{
				"page": json.Number("0"),
			},
		},
		{
			name: "deeply nested path (3 levels)",
			message: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"page": float64(7),
					},
				},
			},
			paths: [][]string{{"a", "b", "page"}},
			want: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"page": float64(6),
					},
				},
			},
		},
		{
			name: "nil intermediate map is no-op",
			message: map[string]interface{}{
				"query": nil,
			},
			paths: [][]string{{"query", "inner_page"}},
			want: map[string]interface{}{
				"query": nil,
			},
		},
		{
			name: "empty paths slice is no-op",
			message: map[string]interface{}{
				"page": float64(5),
			},
			paths: nil,
			want: map[string]interface{}{
				"page": float64(5),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			AdjustZeroBasedPaginationFields(tt.message, tt.paths)
			g.Expect(tt.message).To(Equal(tt.want))
		})
	}
}
