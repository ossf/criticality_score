package result

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/ossf/criticality_score/cmd/collect_signals/signal"
)

type csvWriter struct {
	header        []string
	w             *csv.Writer
	headerWritten bool
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
		return "", nil
	}
}
