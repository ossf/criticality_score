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

package signal

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

type validateSet1 struct{}

func (t validateSet1) Namespace() Namespace {
	return "te-st"
}

type validateSet2 struct {
	F Field[int] `signal:"ab-bc"`
}

func (t validateSet2) Namespace() Namespace {
	return "test"
}

type validateSet3 struct {
	F Field[int]
}

func (t validateSet3) Namespace() Namespace {
	return "test"
}

func TestValidateSet(t *testing.T) {
	tests := []struct { //nolint:govet
		name       string
		s          Set
		wantErr    bool
		errMessage string
	}{
		{
			name:       "namespace contains invalid characters",
			s:          validateSet1{},
			wantErr:    true,
			errMessage: fmt.Sprintf("namespace '%s' contains invalid characters", validateSet1{}.Namespace()),
		},
		{
			name: "invalid name",
			s: &validateSet2{
				F: Field[int]{value: 4},
			},
			wantErr:    true,
			errMessage: fmt.Sprintf("field name '%s' contains invalid character", "ab-bc"),
		},
		{
			name: "valid",
			s: &validateSet3{
				F: Field[int]{value: 4},
			},
			wantErr: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateSet(test.s)

			if (err != nil) != test.wantErr {
				t.Errorf("ValidateSet() error = %v, wantErr %v", err, test.wantErr)
			}

			if test.wantErr && err.Error() != test.errMessage {
				t.Errorf("ValidateSet() error = %v, wantErr %v", err, test.errMessage)
			}
		})
	}
}

type testParseStructField struct{}

func (t testParseStructField) Value() any {
	return nil
}

func Test_parseStructField(t *testing.T) {
	type parseTypeTest struct {
		field testParseStructField //nolint
	}

	tests := []struct { //nolint:govet
		name string
		sf   reflect.StructField
		want *fieldConfig
	}{
		{
			name: "valid",
			sf: reflect.StructField{
				Name: "Test",
				Tag:  reflect.StructTag(`signal:"test"`),
				Type: reflect.TypeOf(testParseStructField{}),
			},
			want: &fieldConfig{
				name:   "test",
				legacy: false,
			},
		},
		{
			name: "tag equals fieldTagIgnore",
			sf: reflect.StructField{
				Name: "Test",
				Tag:  reflect.StructTag(`signal:"-"`),
				Type: reflect.TypeOf(testParseStructField{}),
			},
			want: nil,
		},
		{
			name: "Doesn't implement valuer interface",
			sf: reflect.StructField{
				Name: "Test",
				Tag:  reflect.StructTag(`signal:"test"`),
				Type: reflect.TypeOf(parseTypeTest{}),
			},
			want: nil,
		},
		{
			name: "p equals \"\" and p equals fieldTagLegacy",
			sf: reflect.StructField{
				Name: "Test",
				Tag:  reflect.StructTag(`signal:"legacy,  "`),
				Type: reflect.TypeOf(testParseStructField{}),
			},
			want: &fieldConfig{
				name:   "test",
				legacy: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := parseStructField(test.sf); !reflect.DeepEqual(got, test.want) {
				t.Errorf("parseStructField() = %v, want %v", got, test.want)
			}
		})
	}
}

type testIterSetFields struct { //nolint:govet
	UpdatedCount Field[int]
	Field        string
}

func (t testIterSetFields) Namespace() Namespace {
	return "test-"
}

func Test_iterSetFields(t *testing.T) {
	tests := []struct { //nolint:govet
		name    string
		s       Set
		cb      func(*fieldConfig, any) error
		wantErr bool
	}{
		{
			name: "valid",
			s: &IssuesSet{
				UpdatedCount: Field[int]{value: 1},
			},
			cb: func(fc *fieldConfig, v any) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "cb returns error",
			s: &IssuesSet{
				UpdatedCount: Field[int]{value: 1},
			},
			cb: func(fc *fieldConfig, v any) error {
				return errors.New("random error")
			},
			wantErr: true,
		},
		{
			name: "parseStructField returns nil",
			s: &testIterSetFields{
				UpdatedCount: Field[int]{value: 1},
				Field:        "random",
			},
			cb: func(fc *fieldConfig, v any) error {
				return nil
			},
			wantErr: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := iterSetFields(test.s, test.cb); (err != nil) != test.wantErr {
				t.Errorf("iterSetFields() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

type testSetFields struct {
	UpdatedCount Field[int] `signal:"legacy"`
}

func (t testSetFields) Namespace() Namespace {
	return "test-"
}

func TestSetFields(t *testing.T) {
	tests := []struct { //nolint:govet
		name      string
		s         Set
		namespace bool
		want      []string
	}{
		{
			name: "valid",
			s: &testIterSetFields{
				UpdatedCount: Field[int]{value: 1},
			},
			namespace: false,
			want:      []string{"updated_count"},
		},
		{
			name: "valid with namespace",
			s: &testIterSetFields{
				UpdatedCount: Field[int]{value: 1},
			},
			namespace: true,
			want:      []string{"test-.updated_count"},
		},
		{
			name: "valid with namespace and legacy",
			s: &testSetFields{
				UpdatedCount: Field[int]{value: 1},
			},
			namespace: true,
			want:      []string{"legacy.updated_count"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := SetFields(test.s, test.namespace); !reflect.DeepEqual(got, test.want) {
				t.Errorf("SetFields() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSetValues(t *testing.T) {
	tests := []struct {
		name string
		s    Set
		want []any
	}{
		{
			name: "valid",
			s: &testSetFields{
				UpdatedCount: Field[int]{value: 1, set: true},
			},
			want: []any{1},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := SetValues(test.s); !reflect.DeepEqual(got, test.want) {
				t.Errorf("SetValues() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSetAsMap(t *testing.T) {
	tests := []struct { //nolint:govet
		name      string
		s         Set
		namespace bool
		want      map[string]any
	}{
		{
			name: "valid",
			s: &testSetFields{
				UpdatedCount: Field[int]{value: 1, set: true},
			},
			namespace: false,
			want:      map[string]any{"updated_count": 1},
		},
		{
			name: "valid with namespace",
			s: &testSetFields{
				UpdatedCount: Field[int]{value: 1, set: true},
			},
			namespace: true,
			want:      map[string]any{"legacy.updated_count": 1},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := SetAsMap(test.s, test.namespace); !reflect.DeepEqual(got, test.want) {
				t.Errorf("SetAsMap() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSetAsMapWithNamespace(t *testing.T) {
	test := struct { //nolint:govet
		name string
		s    Set
		want map[string]map[string]any
	}{
		name: "valid",
		s: &testSetFields{
			UpdatedCount: Field[int]{value: 1, set: true},
		},
		want: map[string]map[string]any{"legacy": {"updated_count": 1}},
	}

	if got := SetAsMapWithNamespace(test.s); !reflect.DeepEqual(got, test.want) {
		t.Errorf("SetAsMapWithNamespace() = %v, want %v", got, test.want)
	}
}

func TestField_Set(t *testing.T) {
	type testCase[T SupportedType] struct { //nolint:govet
		name      string
		s         Field[T]
		v         T
		shouldSet bool
	}
	tests := []testCase[int]{
		{
			name:      "valid",
			s:         Field[int]{},
			v:         2,
			shouldSet: true,
		},
		{
			name:      "not set",
			s:         Field[int]{},
			shouldSet: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.shouldSet {
				test.s.Set(test.v)
			}
			if test.shouldSet && test.s.value != test.v {
				t.Errorf("Field.Set() = %v, want %v", test.s.value, test.v)
			}

			if test.shouldSet && !test.s.IsSet() {
				t.Errorf("Field.Set() was not set")
			} else if !test.shouldSet && test.s.IsSet() {
				t.Errorf("Field.Set() was set")
			}
		})
	}
}

func TestField_Get(t *testing.T) {
	type testCase[T SupportedType] struct { //nolint:govet
		name string
		s    Field[T]
		want T
	}
	tests := []testCase[int]{
		{
			name: "valid",
			s:    Field[int]{value: 1, set: true},
			want: 1,
		},
		{
			name: "not set",
			s:    Field[int]{},
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
	type testCase[T SupportedType] struct {
		name        string
		s           Field[T]
		shouldUnset bool
	}
	tests := []testCase[int]{
		{
			name:        "valid", // should we reset the value to T when we unset?
			s:           Field[int]{value: 1, set: true},
			shouldUnset: true,
		},
		{
			name: "not set",
			s:    Field[int]{},
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
	type testCase[T SupportedType] struct {
		name string
		v    T
		want Field[T]
	}
	test := testCase[int]{
		name: "valid",
		v:    1,
		want: Field[int]{value: 1, set: true},
	}
	if got := Val(test.v); !reflect.DeepEqual(got, test.want) {
		t.Errorf("Val() = %v, want %v", got, test.want)
	}
}
