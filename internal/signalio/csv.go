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
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ossf/criticality_score/internal/collector/signal"
)

type csvWriter struct {
	w             *csv.Writer
	header        []string
	headerWritten bool

	// Prevents concurrent writes to w, and headerWritten.
	mu sync.Mutex
}

func headerFromSignalSets(sets []signal.Set) []string {
	var hs []string
	for _, s := range sets {
		if err := signal.ValidateSet(s); err != nil {
			panic(err)
		}
		hs = append(hs, signal.SetFields(s, true)...)
	}
	return hs
}

func CsvWriter(w io.Writer, emptySets []signal.Set) Writer {
	return &csvWriter{
		header: headerFromSignalSets(emptySets),
		w:      csv.NewWriter(w),
	}
}

// WriteSignals implements the Writer interface.
func (w *csvWriter) WriteSignals(signals []signal.Set) error {
	values := make(map[string]string)
	for _, s := range signals {
		// Get all of the signal data from the set and serialize it.
		for k, v := range signal.SetAsMap(s, true) {
			if s, err := marshalValue(v); err != nil {
				return fmt.Errorf("failed to write field %s: %w", k, err)
			} else {
				values[k] = s
			}
		}
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

func marshalValue(value any) (string, error) {
	switch v := value.(type) {
	case bool, int, int16, int32, int64, uint, uint16, uint32, uint64, byte, float32, float64, string:
		return fmt.Sprintf("%v", value), nil
	case time.Time:
		return v.Format(time.RFC3339), nil
	case nil:
		return "", nil
	default:
		return "", fmt.Errorf("%w: %T", ErrorMarshalFailure, value)
	}
}
