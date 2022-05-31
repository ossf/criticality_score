package result

import (
	"encoding/csv"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ossf/criticality_score/cmd/collect_signals/signal"
)

type csvWriter struct {
	header        []string
	w             *csv.Writer
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

func NewCsvWriter(w io.Writer, emptySets []signal.Set) Writer {
	return &csvWriter{
		header: headerFromSignalSets(emptySets),
		w:      csv.NewWriter(w),
	}
}

func (w *csvWriter) Record() RecordWriter {
	return &csvRecord{
		values: make(map[string]string),
		sink:   w,
	}
}

func (s *csvWriter) maybeWriteHeader() error {
	// Check headerWritten without the lock to avoid holding the lock if the
	// header has already been written.
	if s.headerWritten {
		return nil
	}
	// Grab the lock and re-check headerWritten just in case another goroutine
	// entered the same critical section. Also prevent concurrent writes to w.
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.headerWritten {
		return nil
	}
	s.headerWritten = true
	return s.w.Write(s.header)
}

func (s *csvWriter) writeRecord(c *csvRecord) error {
	if err := s.maybeWriteHeader(); err != nil {
		return err
	}
	var rec []string
	for _, k := range s.header {
		rec = append(rec, c.values[k])
	}
	// Grab the lock when we're ready to write the record to prevent
	// concurrent writes to w.
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.w.Write(rec); err != nil {
		return err
	}
	s.w.Flush()
	return s.w.Error()
}

type csvRecord struct {
	values map[string]string
	sink   *csvWriter
}

func (r *csvRecord) WriteSignalSet(s signal.Set) error {
	data := signal.SetAsMap(s, true)
	for k, v := range data {
		if s, err := marshalValue(v); err != nil {
			return err
		} else {
			r.values[k] = s
		}
	}
	return nil
}

func (r *csvRecord) Done() error {
	return r.sink.writeRecord(r)
}

func marshalValue(value any) (string, error) {
	switch v := value.(type) {
	case bool, int, int16, int32, int64, uint, uint16, uint32, uint64, byte, float32, float64, string:
		return fmt.Sprintf("%v", value), nil
	case time.Time:
		return v.Format(time.RFC3339), nil
	default:
		return "", MarshalError
	}
}
