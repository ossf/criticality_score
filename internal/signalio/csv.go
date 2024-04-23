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
	"encoding/csv"
	"io"
	"sync"

	"github.com/ossf/criticality_score/v2/internal/collector/signal"
)

type csvWriter struct {
	w             *csv.Writer
	header        []string
	headerWritten bool

	// Prevents concurrent writes to w, and headerWritten.
	mu sync.Mutex
}

func CSVWriter(w io.Writer, emptySets []signal.Set, extra ...string) Writer {
	return &csvWriter{
		header: fieldsFromSignalSets(emptySets, extra),
		w:      csv.NewWriter(w),
	}
}

// WriteSignals implements the Writer interface.
func (w *csvWriter) WriteSignals(signals []signal.Set, extra ...Field) error {
	values, err := marshalToMap(signals, extra...)
	if err != nil {
		return err
	}
	return w.writeRecord(values)
}

func (w *csvWriter) maybeWriteHeader() error {
	// Check headerWritten without the lock to avoid holding the lock if the
	// header has already been written.
	if w.headerWritten {
		return nil
	}
	// Grab the lock and re-check headerWritten just in case another goroutine
	// entered the same critical section. Also prevent concurrent writes to w.
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.headerWritten {
		return nil
	}
	w.headerWritten = true
	return w.w.Write(w.header)
}

func (w *csvWriter) writeRecord(values map[string]string) error {
	if err := w.maybeWriteHeader(); err != nil {
		return err
	}
	var rec []string
	for _, k := range w.header {
		rec = append(rec, values[k])
	}
	// Grab the lock when we're ready to write the record to prevent
	// concurrent writes to w.
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.w.Write(rec); err != nil {
		return err
	}
	w.w.Flush()
	return w.w.Error()
}
