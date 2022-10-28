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

type Value interface {
	// Value takes in a set of fields does some work and returns either the
	// result and true to indicate success, or 0 and false to indicate
	// the result could not be generated.
	Value(fields map[string]float64) (float64, bool)
}

// Field implements the Value interface, but simply returns the raw value of
// the named field.
type Field string

func (f Field) String() string {
	return string(f)
}

// Value implements the Value interface.
func (f Field) Value(fields map[string]float64) (float64, bool) {
	v, ok := fields[string(f)]
	return v, ok
}

type Condition func(fields map[string]float64) bool

func NotCondition(c Condition) Condition {
	return func(fields map[string]float64) bool {
		return !c(fields)
	}
}

func ExistsCondition(f Field) Condition {
	return func(fields map[string]float64) bool {
		_, exists := fields[f.String()]
		return exists
	}
}

// ConditionalValue wraps an Inner value that will only be returned if the
// Condition returns true.
type ConditionalValue struct {
	Condition Condition
	Inner     Value
}

// Value implements the Value interface.
func (cv *ConditionalValue) Value(fields map[string]float64) (float64, bool) {
	v, ok := cv.Inner.Value(fields)
	if !ok {
		return 0, false
	}
	if cv.Condition(fields) {
		return v, true
	} else {
		return 0, false
	}
}
