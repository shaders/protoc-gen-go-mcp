// Copyright 2025 Redpanda Data, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runtime

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"
)

func TestCompressToToon(t *testing.T) {
	g := NewWithT(t)

	t.Run("compress simple object", func(t *testing.T) {
		input := map[string]interface{}{
			"name":   "test",
			"value":  42,
			"active": true,
		}
		jsonData, err := json.Marshal(input)
		g.Expect(err).ToNot(HaveOccurred())

		toonData, err := CompressToToon(jsonData)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(toonData).ToNot(BeEmpty())

		// TOON should be more compact than JSON for structured data
		t.Logf("Original JSON length: %d", len(jsonData))
		t.Logf("TOON length: %d", len(toonData))
		t.Logf("TOON output:\n%s", toonData)
	})

	t.Run("compress array of objects", func(t *testing.T) {
		input := []map[string]interface{}{
			{"id": 1, "name": "Alice", "score": 95},
			{"id": 2, "name": "Bob", "score": 87},
			{"id": 3, "name": "Charlie", "score": 92},
		}
		jsonData, err := json.Marshal(input)
		g.Expect(err).ToNot(HaveOccurred())

		toonData, err := CompressToToon(jsonData)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(toonData).ToNot(BeEmpty())

		// For uniform arrays, TOON should provide significant compression
		t.Logf("Original JSON length: %d", len(jsonData))
		t.Logf("TOON length: %d", len(toonData))
		t.Logf("Compression ratio: %.2f%%", float64(len(toonData))/float64(len(jsonData))*100)
		t.Logf("TOON output:\n%s", toonData)
	})

	t.Run("handle invalid JSON", func(t *testing.T) {
		invalidJSON := []byte("{invalid json")
		_, err := CompressToToon(invalidJSON)
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("compress nested object", func(t *testing.T) {
		input := map[string]interface{}{
			"user": map[string]interface{}{
				"name":  "John",
				"email": "john@example.com",
				"preferences": map[string]interface{}{
					"theme":         "dark",
					"notifications": true,
				},
			},
			"items": []string{"item1", "item2", "item3"},
		}
		jsonData, err := json.Marshal(input)
		g.Expect(err).ToNot(HaveOccurred())

		toonData, err := CompressToToon(jsonData)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(toonData).ToNot(BeEmpty())

		t.Logf("Original JSON length: %d", len(jsonData))
		t.Logf("TOON length: %d", len(toonData))
		t.Logf("TOON output:\n%s", toonData)
	})
}
