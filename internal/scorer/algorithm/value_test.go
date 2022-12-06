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

func TestCondition(t *testing.T) {
	tests := []struct { //nolint:govet
		name       string
		f          Field
		fields     map[string]float64
		existsWant bool // for ExistsCondition()
		notWant    bool // for NotCondition()
	}{
		{
			name:       "exists",
			f:          Field("a"),
			fields:     map[string]float64{"a": 1},
			existsWant: true,
			notWant:    false,
		},
		{
			name:       "not exists",
			f:          Field("a"),
			fields:     map[string]float64{"b": 1},
			existsWant: false,
			notWant:    true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ExistsCondition(test.f)

			if got(test.fields) != test.existsWant {
				t.Errorf("ExistsCondition() = %v, wantVal %v", got(test.fields), test.existsWant)
			}

			got = NotCondition(got)

			if got(test.fields) != test.notWant {
				t.Errorf("NotCondition() = %v, wantVal %v", got(test.fields), test.notWant)
			}
		})
	}
}

func TestConditionalValue_Value(t *testing.T) {
	type want struct {
		value  float64
		exists bool
	}
	tests := []struct { //nolint:govet
		name      string
		Condition Condition
		value     Field
		fields    map[string]float64
		w         want
	}{
		{
			name:      "exists",
			Condition: ExistsCondition(Field("a")),
			value:     Field("a"),
			fields:    map[string]float64{"a": 1},
			w:         want{1, true},
		},
		{
			name:      "not exists",
			Condition: ExistsCondition(Field("a")),
			value:     Field("a"),
			fields:    map[string]float64{"b": 1},
			w:         want{0, false},
		},
		{
			name:      "cv.Inner.Value not have fields",
			Condition: ExistsCondition(Field("a")),
			value:     Field("b"),
			fields:    map[string]float64{"b": 1},
			w:         want{0, false},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cv := &ConditionalValue{
				Condition: test.Condition,
				Inner:     test.value,
			}
			gotVal, gotBool := cv.Value(test.fields)
			if gotVal != test.w.value {
				t.Errorf("Value() gotVal = %v, want %v", gotVal, test.w.value)
			}
			if gotBool != test.w.exists {
				t.Errorf("Value() gotBool = %v, want %v", gotBool, test.w.exists)
			}
		})
	}
}
