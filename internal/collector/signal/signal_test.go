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

package signal_test

import (
	"reflect"
	"testing"

	"github.com/ossf/criticality_score/internal/collector/signal"
)

type validateSet1 struct{}

func (t validateSet1) Namespace() signal.Namespace {
	return "te-st"
}

type validateSet2 struct {
	F signal.Field[int] `signal:"ab-bc"`
}

func (t validateSet2) Namespace() signal.Namespace {
	return "test"
}

type validateSet3 struct {
	F signal.Field[int]
}

func (t validateSet3) Namespace() signal.Namespace {
	return "test"
}

func TestValidateSet(t *testing.T) {
	tests := []struct { //nolint:govet
		name    string
		s       signal.Set
		wantErr bool
	}{
		{
			name:    "namespace contains invalid characters",
			s:       validateSet1{},
			wantErr: true,
		},
		{
			name: "invalid name",
			s: &validateSet2{
				F: signal.Val(4),
			},
			wantErr: true,
		},
		{
			name: "valid",
			s: &validateSet3{
				F: signal.Val(4),
			},
			wantErr: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := signal.ValidateSet(test.s)

			if (err != nil) != test.wantErr {
				t.Errorf("ValidateSet() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

type signalSet struct { //nolint:govet
	UpdatedCount signal.Field[int]
	Field        string
}

func (t signalSet) Namespace() signal.Namespace {
	return "test-"
}

type testSetFields struct {
	UpdatedCount signal.Field[int] `signal:"legacy"`
}

func (t testSetFields) Namespace() signal.Namespace {
	return "test-"
}

func TestSetFields(t *testing.T) {
	tests := []struct { //nolint:govet
		name      string
		s         signal.Set
		namespace bool
		want      []string
	}{
		{
			name: "valid",
			s: &signalSet{
				UpdatedCount: signal.Val(1),
			},
			namespace: false,
			want:      []string{"updated_count"},
		},
		{
			name: "valid with namespace",
			s: &signalSet{
				UpdatedCount: signal.Val(1),
			},
			namespace: true,
			want:      []string{"test-.updated_count"},
		},
		{
			name: "valid with namespace and legacy",
			s: &testSetFields{
				UpdatedCount: signal.Val(1),
			},
			namespace: true,
			want:      []string{"legacy.updated_count"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := signal.SetFields(test.s, test.namespace); !reflect.DeepEqual(got, test.want) {
				t.Errorf("SetFields() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSetValues(t *testing.T) {
	tests := []struct {
		name string
		s    signal.Set
		want []any
	}{
		{
			name: "valid",
			s: &testSetFields{
				UpdatedCount: signal.Val(1),
			},
			want: []any{1},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := signal.SetValues(test.s); !reflect.DeepEqual(got, test.want) {
				t.Errorf("SetValues() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSetAsMap(t *testing.T) {
	tests := []struct { //nolint:govet
		name      string
		s         signal.Set
		namespace bool
		want      map[string]any
	}{
		{
			name: "valid",
			s: &testSetFields{
				UpdatedCount: signal.Val(1),
			},
			namespace: false,
			want:      map[string]any{"updated_count": 1},
		},
		{
			name: "valid with namespace",
			s: &testSetFields{
				UpdatedCount: signal.Val(1),
			},
			namespace: true,
			want:      map[string]any{"legacy.updated_count": 1},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := signal.SetAsMap(test.s, test.namespace); !reflect.DeepEqual(got, test.want) {
				t.Errorf("SetAsMap() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSetAsMapWithNamespace(t *testing.T) {
	test := struct { //nolint:govet
		name string
		s    signal.Set
		want map[string]map[string]any
	}{
		name: "valid",
		s: &testSetFields{
			UpdatedCount: signal.Val(1),
		},
		want: map[string]map[string]any{"legacy": {"updated_count": 1}},
	}

	if got := signal.SetAsMapWithNamespace(test.s); !reflect.DeepEqual(got, test.want) {
		t.Errorf("SetAsMapWithNamespace() = %v, want %v", got, test.want)
	}
}

func TestField_Get(t *testing.T) {
	type testCase[T signal.SupportedType] struct { //nolint:govet
		name string
		s    signal.Field[T]
		want T
	}
	tests := []testCase[int]{
		{
			name: "valid",
			s:    signal.Val(1),
			want: 1,
		},
		{
			name: "not set",
			s:    signal.Field[int]{},
			want: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.s.Get(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Get() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestField_Unset(t *testing.T) {
	type testCase[T signal.SupportedType] struct {
		name        string
		s           signal.Field[T]
		shouldUnset bool
	}
	tests := []testCase[int]{
		{
			name:        "valid", // should we reset the value to T when we unset?
			s:           signal.Val(1),
			shouldUnset: true,
		},
		{
			name: "not set",
			s:    signal.Field[int]{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.shouldUnset {
				test.s.Unset()
			}
			if test.shouldUnset && test.s.IsSet() {
				t.Errorf("Field.Unset() was set")
			}
		})
	}
}

func TestVal(t *testing.T) {
	type testCase[T signal.SupportedType] struct {
		name string
		v    T
		want signal.Field[T]
	}
	test := testCase[int]{
		name: "valid",
		v:    1,
		want: signal.Val(1),
	}
	if got := signal.Val(test.v); !reflect.DeepEqual(got, test.want) {
		t.Errorf("Val() = %v, want %v", got, test.want)
	}
}
