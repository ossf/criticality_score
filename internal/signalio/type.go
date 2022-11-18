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
	"bytes"
	"errors"
	"io"

	"github.com/ossf/criticality_score/internal/collector/signal"
)

type WriterType int

const (
	WriterTypeCSV = WriterType(iota)
	WriterTypeJSON
	WriterTypeText
)

var ErrorUnknownWriterType = errors.New("unknown writer type")

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
	case WriterTypeCSV:
		return []byte("csv"), nil
	case WriterTypeJSON:
		return []byte("json"), nil
	case WriterTypeText:
		return []byte("text"), nil
	default:
		return []byte{}, ErrorUnknownWriterType
	}
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (t *WriterType) UnmarshalText(text []byte) error {
	switch {
	case bytes.Equal(text, []byte("csv")):
		*t = WriterTypeCSV
	case bytes.Equal(text, []byte("json")):
		*t = WriterTypeJSON
	case bytes.Equal(text, []byte("text")):
		*t = WriterTypeText
	default:
		return ErrorUnknownWriterType
	}
	return nil
}

// New will return a new instance of the corresponding implementation of
// Writer for the given WriterType.
func (t *WriterType) New(w io.Writer, emptySets []signal.Set, extra ...string) Writer {
	switch *t {
	case WriterTypeCSV:
		return CSVWriter(w, emptySets, extra...)
	case WriterTypeJSON:
		return JSONWriter(w)
	case WriterTypeText:
		return TextWriter(w, emptySets, extra...)
	default:
		return nil
	}
}
