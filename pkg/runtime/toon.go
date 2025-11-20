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

	"github.com/toon-format/toon-go"
)

// CompressToToon converts JSON bytes to TOON format for more efficient LLM token usage
// TOON (Token-Oriented Object Notation) is a compact format that reduces token count by ~40%
func CompressToToon(jsonData []byte) (string, error) {
	// Parse JSON into a generic structure
	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return "", err
	}

	// Encode to TOON format with length markers for better structure
	toonData, err := toon.Marshal(data, toon.WithLengthMarkers(true))
	if err != nil {
		return "", err
	}

	return string(toonData), nil
}
