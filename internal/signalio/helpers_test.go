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
	"reflect"
	"testing"

	"github.com/ossf/criticality_score/internal/collector/signal"
)

type testSet struct { //nolint:govet
	UpdatedCount signal.Field[int]
	Field        string
}

func (t testSet) Namespace() signal.Namespace {
	return "test"
}

func Test_marshalToMap(t *testing.T) {
	tests := []struct { //nolint:govet
		name    string
		extra   []Field
		signals []signal.Set
		want    map[string]string
		wantErr bool
	}{
		{
			name: "default",
			extra: []Field{
				{
					Key:   "test",
					Value: "1",
				},
				{
					Key:   "test2",
					Value: "2",
				},
			},
			signals: []signal.Set{
				&testSet{
					UpdatedCount: signal.Val(1),
				},
				&testSet{
					UpdatedCount: signal.Val(3),
				},
			},
			want: map[string]string{"test": "1", "test.updated_count": "3", "test2": "2"},
		},
		{
			name: "marshal error extra",
			extra: []Field{
				{
					Key:   "test",
					Value: []string{"1"},
				},
			},
			signals: []signal.Set{
				&testSet{
					UpdatedCount: signal.Val(1),
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty extra",
			signals: []signal.Set{
				&testSet{
					UpdatedCount: signal.Val(1),
				},
			},
			want: map[string]string{"test.updated_count": "1"},
		},
		{
			name: "empty signals",
			extra: []Field{
				{
					Key:   "test",
					Value: "1",
				},
			},
			want: map[string]string{"test": "1"},
		},
		{
			name: "empty signals and extra",
			want: map[string]string{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := marshalToMap(test.signals, test.extra...)
			if (err != nil) != test.wantErr {
				t.Errorf("marshalToMap() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("marshalToMap() got = %v, want %v", got, test.want)
			}
		})
	}
}

type testPanicSet struct { //nolint:govet
	UpdatedCount signal.Field[int]
	Field        string
}

func (t testPanicSet) Namespace() signal.Namespace {
	return "invalid-namespace"
}

func Test_fieldsFromSignalSets(t *testing.T) {
	tests := []struct {
		name      string
		extra     []string
		sets      []signal.Set
		want      []string
		wantPanic bool
	}{
		{
			name: "valid input",

			extra: []string{"test"},
			sets: []signal.Set{
				&testSet{
					UpdatedCount: signal.Val(1),
				},
				&testSet{
					UpdatedCount: signal.Val(3),
				},
			},
			want: []string{"test.updated_count", "test.updated_count", "test"},
		},
		{
			name: "empty extra",
			sets: []signal.Set{
				&testSet{
					UpdatedCount: signal.Val(1),
				},
			},
			want: []string{"test.updated_count"},
		},
		{
			name: "empty sets",
			extra: []string{
				"test",
			},
			want: []string{"test"},
		},
		{
			name: "panics",
			sets: []signal.Set{
				&testPanicSet{
					UpdatedCount: signal.Val(1),
				},
			},

			wantPanic: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if p := recover(); (p != nil) != test.wantPanic {
					t.Errorf("want panic %v, got %v", test.wantPanic, p)
				}
			}()

			if got := fieldsFromSignalSets(test.sets, test.extra); !reflect.DeepEqual(got, test.want) {
				t.Errorf("fieldsFromSignalSets() = %v, want %v", got, test.want)
			}
		})
	}
}
