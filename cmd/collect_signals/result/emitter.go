package result

import (
	"errors"

	"github.com/ossf/criticality_score/cmd/collect_signals/signal"
)

var (
	MarshalError = errors.New("failed to marshal value")
)

type RecordWriter interface {
	// WriteSignalSet is used to output the value for a signal.Set for a record.
	WriteSignalSet(signal.Set) error

	// Done indicates that all the fields for the record have been written and
	// record is complete.
	Done() error
}

type Writer interface {
	//WriteAll([]signal.Set) error

	// Record returns a RecordWriter that can be used to write a new record.
	Record() RecordWriter
}
