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
	"io"
	"sync"
	"testing"

	"github.com/ossf/criticality_score/internal/collector/signal"
)

func Test_textWriter_writeRecord(t *testing.T) {
	type fields struct {
		w                  io.Writer
		fields             []string
		firstRecordWritten bool
	}
	tests := []struct { //nolint:govet
		name    string
		fields  fields
		values  map[string]string
		wantErr bool
	}{
		{
			name: "regular",
			fields: fields{
				w: io.Writer(&mockWriter{
					written: []byte{},
				}),
				fields:             []string{"test"},
				firstRecordWritten: false,
			},
			values: map[string]string{
				"test": "test",
			},
		},
		{
			name: "first record written and w is invalid",
			fields: fields{
				w:                  &mockWriter{},
				fields:             []string{"test"},
				firstRecordWritten: true,
			},
			values: map[string]string{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := &textWriter{
				w:                  test.fields.w,
				fields:             test.fields.fields,
				firstRecordWritten: test.fields.firstRecordWritten,
				mu:                 sync.Mutex{},
			}

			if err := w.writeRecord(test.values); (err != nil) != test.wantErr {
				t.Errorf("writeRecord() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

type mockWriter struct {
	written []byte
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}

func Test_textWriter_WriteSignals(t *testing.T) {
	type fields struct {
		w                  io.Writer
		fields             []string
		firstRecordWritten bool
	}
	type args struct {
		signals []signal.Set
		extra   []Field
	}
	tests := []struct { //nolint:govet
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "regular",
			fields: fields{
				w: io.Writer(&mockWriter{
					written: []byte{},
				}),
				fields:             []string{"test"},
				firstRecordWritten: true,
			},
			args: args{
				signals: []signal.Set{},
				extra:   []Field{},
			},
		},
		{
			name: "error while marshaling with extra",
			args: args{
				signals: []signal.Set{},
				extra:   []Field{{"test", []string{"invalid"}}},
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := &textWriter{
				w:                  test.fields.w,
				fields:             test.fields.fields,
				firstRecordWritten: test.fields.firstRecordWritten,
				mu:                 sync.Mutex{},
			}
			if err := w.WriteSignals(test.args.signals, test.args.extra...); (err != nil) != test.wantErr {
				t.Errorf("WriteSignals() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
