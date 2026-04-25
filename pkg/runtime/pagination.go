package runtime

import (
	"encoding/json"
	"strconv"
)

// AdjustZeroBasedPaginationFields walks the given message and decrements the
// integer value at each path by 1 so that a 1-based value sent by an MCP
// client becomes the 0-based value expected by the underlying gRPC API.
//
// Each path is a slice of map keys leading to the integer field. Missing
// fields and non-numeric values are left untouched. Values <= 0 are clamped
// to 0 to avoid producing negative pagination indices when the client ignores
// the schema's minimum=1.
func AdjustZeroBasedPaginationFields(message map[string]interface{}, paths [][]string) {
	if len(message) == 0 || len(paths) == 0 {
		return
	}
	for _, path := range paths {
		decrementAtPath(message, path)
	}
}

func decrementAtPath(m map[string]interface{}, path []string) {
	if len(path) == 0 {
		return
	}
	if len(path) == 1 {
		key := path[0]
		v, ok := m[key]
		if !ok {
			return
		}
		if adjusted, ok := decrementValue(v); ok {
			m[key] = adjusted
		}
		return
	}
	next, ok := m[path[0]].(map[string]interface{})
	if !ok {
		return
	}
	decrementAtPath(next, path[1:])
}

// decrementValue returns v decremented by 1 (clamped to 0) preserving the
// original numeric type. The second return value is false if v is not a
// recognized numeric kind, in which case the value is left untouched by the
// caller.
func decrementValue(v interface{}) (interface{}, bool) {
	switch n := v.(type) {
	case float64:
		if n >= 1 {
			return n - 1, true
		}
		return float64(0), true
	case float32:
		if n >= 1 {
			return n - 1, true
		}
		return float32(0), true
	case int:
		if n >= 1 {
			return n - 1, true
		}
		return 0, true
	case int32:
		if n >= 1 {
			return n - 1, true
		}
		return int32(0), true
	case int64:
		if n >= 1 {
			return n - 1, true
		}
		return int64(0), true
	case uint:
		if n >= 1 {
			return n - 1, true
		}
		return uint(0), true
	case uint32:
		if n >= 1 {
			return n - 1, true
		}
		return uint32(0), true
	case uint64:
		if n >= 1 {
			return n - 1, true
		}
		return uint64(0), true
	case json.Number:
		// Prefer integer math; fall back to float for fractional inputs.
		if i, err := n.Int64(); err == nil {
			if i >= 1 {
				return json.Number(strconv.FormatInt(i-1, 10)), true
			}
			return json.Number("0"), true
		}
		if f, err := n.Float64(); err == nil {
			if f >= 1 {
				return json.Number(strconv.FormatFloat(f-1, 'f', -1, 64)), true
			}
			return json.Number("0"), true
		}
		return v, false
	}
	return v, false
}
