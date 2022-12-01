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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseStructField(tt.sf); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseStructField() = %v, want %v", got, tt.want)
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := iterSetFields(tt.s, tt.cb); (err != nil) != tt.wantErr {
				t.Errorf("iterSetFields() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
