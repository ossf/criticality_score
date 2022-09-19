package repowriter

import (
	"bytes"
	"errors"
	"io"
)

type WriterType int

const (
	// WriterTypeText corresponds to the Writer returned by Text.
	WriterTypeText = WriterType(iota)

	// WriterTypeScorecard corresponds to the Writer returned by Scorecard.
	WriterTypeScorecard
)

var ErrorUnknownRepoWriterType = errors.New("unknown repo writer type")

// String implements the fmt.Stringer interface.
func (t WriterType) String() string {
	text, err := t.MarshalText()
	if err != nil {
		return ""
	}
	return string(text)
}

// MarshalText implements the encoding.TextMarshaler interface.
func (t WriterType) MarshalText() ([]byte, error) {
	switch t {
	case WriterTypeText:
		return []byte("text"), nil
	case WriterTypeScorecard:
		return []byte("scorecard"), nil
	default:
		return []byte{}, ErrorUnknownRepoWriterType
	}
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (t *WriterType) UnmarshalText(text []byte) error {
	switch {
	case bytes.Equal(text, []byte("text")):
		*t = WriterTypeText
	case bytes.Equal(text, []byte("scorecard")):
		*t = WriterTypeScorecard
	default:
		return ErrorUnknownRepoWriterType
	}
	return nil
}

// New will return a new instance of the corresponding implementation of
// Writer for the given WriterType.
func (t *WriterType) New(w io.Writer) Writer {
	switch *t {
	case WriterTypeText:
		return Text(w)
	case WriterTypeScorecard:
		return Scorecard(w)
	default:
		return nil
	}
}
