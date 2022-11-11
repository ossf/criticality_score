// Copyright 2022 Criticality Score Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package signalio

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/ossf/criticality_score/internal/collector/signal"
)

type jsonWriter struct {
	encoder *json.Encoder

	// Prevents concurrent writes to w.
	mu sync.Mutex
}

func JSONWriter(w io.Writer) Writer {
	e := json.NewEncoder(w)
	e.SetIndent("", "")
	return &jsonWriter{
		encoder: e,
	}
}

// WriteSignals implements the Writer interface.
func (w *jsonWriter) WriteSignals(signals []signal.Set, extra ...Field) error {
	data := make(map[string]any)
	for _, s := range signals {
		m := signal.SetAsMapWithNamespace(s)

		// Merge m with data
		for ns, innerM := range m {
			d, ok := data[ns]
			if !ok {
				d = make(map[string]any)
				data[ns] = d
			}
			nsData, ok := d.(map[string]any)
			if !ok {
				return fmt.Errorf("failed to get map for namespace: %s", ns)
			}
			for k, v := range innerM {
				nsData[k] = v
			}
		}
	}
	for _, f := range extra {
		data[f.Key] = f.Value
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.encoder.Encode(data)
}
