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
	/*
		The variable w.headerWritten is checked twice to avoid what is known as a "race condition".
		A race condition can occur when two or more goroutines try to access a shared resource
		(in this case, the csvWriter instance) concurrently, and the outcome of the program depends on
		the interleaving of their execution.

		Imagine the following scenario:

		1. Goroutine A reads the value of w.headerWritten as false.
		2. Goroutine B reads the value of w.headerWritten as false.
		3. Goroutine A acquires the mutex lock and sets w.headerWritten to true.
		4. Goroutine B acquires the mutex lock and sets w.headerWritten to true.

		If this happens, the header will be written twice, which is not the desired behavior.
		By checking w.headerWritten twice, once before acquiring the mutex lock and once after acquiring the lock,
		the function can ensure that only one goroutine enters the critical section and writes the header.

		Here's how the function works:

		1. Goroutine A reads the value of w.headerWritten as false.
		2. Goroutine A acquires the mutex lock.
		3. Goroutine A re-checks the value of w.headerWritten and finds it to be false.
		4. Goroutine A sets w.headerWritten to true and writes the header.
		5. Goroutine A releases the mutex lock.

		If Goroutine B tries to enter the critical section at any point after step 2,
		it will have to wait until Goroutine A releases the lock in step 5. Once the lock is released,
		Goroutine B will re-check the value of w.headerWritten and find it to be true,
		so it will not write the header again.
	*/

	if w.headerWritten {
		return nil
	}
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
