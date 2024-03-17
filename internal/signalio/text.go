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
	"fmt"
	"io"
	"sync"

	"github.com/ossf/criticality_score/v2/internal/collector/signal"
)

type textWriter struct {
	w                  io.Writer
	fields             []string
	firstRecordWritten bool

	// Prevents concurrent writes to w.
	mu sync.Mutex
}

func TextWriter(w io.Writer, emptySets []signal.Set, extra ...string) Writer {
	return &textWriter{
		w:      w,
		fields: fieldsFromSignalSets(emptySets, extra),
	}
}

// WriteSignals implements the Writer interface.
func (w *textWriter) WriteSignals(signals []signal.Set, extra ...Field) error {
	values, err := marshalToMap(signals, extra...)
	if err != nil {
		return err
	}
	return w.writeRecord(values)
}

func (w *textWriter) writeRecord(values map[string]string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Output a newline between records if this isn't the first record.
	if w.firstRecordWritten {
		_, err := fmt.Fprintln(w.w, "")
		if err != nil {
			return err
		}
	} else {
		w.firstRecordWritten = true
	}

	for _, field := range w.fields {
		val := values[field]
		_, err := fmt.Fprintf(w.w, "%s: %s\n", field, val)
		if err != nil {
			return err
		}
	}
	return nil
}
