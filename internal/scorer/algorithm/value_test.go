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

package algorithm

import (
	"testing"
)

func TestExistsCondition(t *testing.T) {
	tests := []struct { //nolint:govet
		name   string
		f      Field
		fields map[string]float64
		want   bool
	}{
		{
			name:   "exists",
			f:      Field("a"),
			fields: map[string]float64{"a": 1},
			want:   true,
		},
		{
			name:   "not exists",
			f:      Field("a"),
			fields: map[string]float64{"b": 1},
			want:   false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ExistsCondition(test.f)

			if got(test.fields) != test.want {
				t.Errorf("ExistsCondition() = %v, wantVal %v", got(test.fields), test.want)
			}
		})
	}
}

func TestNotCondition(t *testing.T) {
	tests := []struct { //nolint:govet
		name   string
		f      Field
		fields map[string]float64
		want   bool
	}{
		{
			name:   "exists but wantVal false",
			f:      Field("a"),
			fields: map[string]float64{"a": 1},
			want:   false,
		},
		{
			name:   "not exists but wantVal true",
			f:      Field("a"),
			fields: map[string]float64{"b": 1},
			want:   true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exists := ExistsCondition(test.f)
			got := NotCondition(exists)

			if got(test.fields) != test.want {
				t.Errorf("NotCondition() = %v, wantVal %v", got(test.fields), test.want)
			}
		})
	}
}

func TestField_Value(t *testing.T) {
	tests := []struct { //nolint:govet
		name     string
		f        Field
		fields   map[string]float64
		wantVal  float64
		wantBool bool
	}{
		{
			name:     "exists",
			f:        Field("a"),
			fields:   map[string]float64{"a": 1},
			wantVal:  1,
			wantBool: true,
		},
		{
			name:     "not exists",
			f:        Field("a"),
			fields:   map[string]float64{"b": 1},
			wantVal:  0,
			wantBool: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotVal, gotBool := test.f.Value(test.fields)
			if gotVal != test.wantVal {
				t.Errorf("Value() gotVal = %v, wantVal %v", gotVal, test.wantVal)
			}
			if gotBool != test.wantBool {
				t.Errorf("Value() gotBool = %v, wantVal %v", gotBool, test.wantBool)
			}
		})
	}
}

func TestConditionalValue_Value(t *testing.T) {
	tests := []struct { //nolint:govet
		name      string
		Condition Condition
		f         Field
		fields    map[string]float64
		want      float64
		want1     bool
	}{
		{
			name:      "exists",
			Condition: ExistsCondition(Field("a")),
			f:         Field("a"),
			fields:    map[string]float64{"a": 1},
			want:      1,
			want1:     true,
		},
		{
			name:      "not exists",
			Condition: ExistsCondition(Field("a")),
			f:         Field("a"),
			fields:    map[string]float64{"b": 1},
			want:      0,
			want1:     false,
		},
		{
			name:      "cv.Inner.Value not have fields",
			Condition: ExistsCondition(Field("a")),
			f:         Field("b"),
			fields:    map[string]float64{"b": 1},
			want:      0,
			want1:     false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cv := &ConditionalValue{
				Condition: test.Condition,
				Inner:     test.f,
			}
			gotVal, gotBool := cv.Value(test.fields)
			if gotVal != test.want {
				t.Errorf("Value() gotVal = %v, want %v", gotVal, test.want)
			}
			if gotBool != test.want1 {
				t.Errorf("Value() gotBool = %v, want %v", gotBool, test.want1)
			}
		})
	}
}
