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
	"encoding/json"
	"io"
	"testing"

	"github.com/ossf/criticality_score/internal/collector/signal"
)

type testJSONWriterSet struct { //nolint:govet
	UpdatedCount signal.Field[int]
	Field        string
}

func (t testJSONWriterSet) Namespace() signal.Namespace {
	return "test"
}

type mockWriterJSON struct{}

func (m *mockWriterJSON) Write(p []byte) (n int, err error) {
	return 0, nil
}

func Test_jsonWriter_WriteSignals(t *testing.T) {
	type args struct {
		signals []signal.Set
		extra   []Field
	}
	test := struct {
		name    string
		encoder *json.Encoder
		args    args
		wantErr bool
	}{
		name:    "default",
		encoder: json.NewEncoder(io.Writer(&mockWriterJSON{})),
		args: args{
			signals: []signal.Set{
				&testJSONWriterSet{},
			},
			extra: []Field{
				{
					Key:   "extra",
					Value: "value",
				},
			},
		},
	}

	w := JSONWriter(&mockWriterJSON{})
	if err := w.WriteSignals(test.args.signals, test.args.extra...); (err != nil) != test.wantErr {
		t.Errorf("WriteSignals() error = %v, wantErr %v", err, test.wantErr)
	}
}
