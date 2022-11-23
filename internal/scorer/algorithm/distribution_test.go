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
	type args struct {
		name  string
		value float64
	}
	tests := []struct {
		name string
		args args
		want *Distribution
	}{
		{
			name: "invalid name",

			args: args{
				name: "test",
			},
			want: nil,
		},
		{
			name: "linear test",

			args: args{
				name:  "linear",
				value: 300,
			},
			want: &Distribution{
				name: "linear",

				normalizeFn: func(v float64) float64 {
					return v
				},
			},
		},
		{
			name: "zipfian test",

			args: args{
				name:  "zipfian",
				value: 300,
			},
			want: &Distribution{
				name: "zipfian",

				normalizeFn: func(v float64) float64 {
					return math.Log(1 + v)
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LookupDistribution(tt.args.name)

			if got == nil && tt.want == nil {
				return
			}

			if got.String() != tt.want.String() || got.Normalize(tt.args.value) != tt.want.Normalize(tt.args.value) {
				t.Errorf("LookupDistribution() = %v, want %v", got, tt.want)
			}
		})
	}
}
