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

package result

import (
	"errors"

	"github.com/ossf/criticality_score/cmd/collect_signals/signal"
)

var ErrorMarshalFailure = errors.New("failed to marshal value")

type RecordWriter interface {
	// WriteSignalSet is used to output the value for a signal.Set for a record.
	WriteSignalSet(signal.Set) error

	// Done indicates that all the fields for the record have been written and
	// record is complete.
	Done() error
}

type Writer interface {
	// Record returns a RecordWriter that can be used to write a new record.
	Record() RecordWriter
}
