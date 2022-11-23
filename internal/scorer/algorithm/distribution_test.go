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
	"math"
	"testing"
)

func TestLookupDistribution(t *testing.T) {
	type result struct {
		name  string
		value float64
	}

	//nolint:govet
	tests := []struct {
		name              string
		distributionName  string
		distributionValue float64
		want              result
		wantNil           bool
	}{
		{
			name:             "invalid name",
			distributionName: "invalid",

			wantNil: true,
		},
		{
			name: "linear test",

			distributionName:  "linear",
			distributionValue: 300,
			want: result{
				name:  "linear",
				value: 300,
			},
		},
		{
			name: "linear test with zero",

			distributionName:  "linear",
			distributionValue: 0,
			want: result{
				name:  "linear",
				value: 0,
			},
		},
		{
			name: "negative linear test",

			distributionName:  "linear",
			distributionValue: -10,

			want: result{
				name:  "linear",
				value: -10,
			},
		},
		{
			name: "linear test with max int",

			distributionName:  "linear",
			distributionValue: math.MaxInt64,

			want: result{
				name:  "linear",
				value: math.MaxInt64,
			},
		},
		{
			name: "zipfian test",

			distributionName:  "zipfian",
			distributionValue: 300,

			want: result{
				name:  "zipfian",
				value: 5.707110264748875,
			},
		},
		{
			name: "zipfian test with zero",

			distributionName:  "zipfian",
			distributionValue: 0,

			want: result{
				name:  "zipfian",
				value: 0,
			},
		},
		{
			name: "negative zipfian test",

			distributionName:  "zipfian",
			distributionValue: -10,

			want: result{
				name:  "zipfian",
				value: math.NaN(),
			},
		},
		{
			name: "zipfian test with max int",

			distributionName:  "zipfian",
			distributionValue: math.MaxInt64,

			want: result{
				name:  "zipfian",
				value: 43.668272,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LookupDistribution(tt.distributionName)

			if got == nil && tt.wantNil {
				return
			}

			normalizedValue := got.Normalize(tt.distributionValue)

			// round to 6 decimal places for comparison
			normalizedValue = math.Round(normalizedValue*1000000) / 1000000
			wantValue := math.Round(tt.want.value*1000000) / 1000000

			if got.String() != tt.want.name {
				t.Errorf("The names are not equal LookupDistribution name = %s, want name %s",
					got.String(), tt.want.name)
			}

			if math.IsNaN(normalizedValue) && math.IsNaN(tt.want.value) {
				// both are NaN, and we can't compare two NaNs together
				return
			}

			if normalizedValue != wantValue {
				t.Errorf("The values are not equal LookupDistribution value %f, want value %f",
					normalizedValue, wantValue)
			}
		})
	}
}
