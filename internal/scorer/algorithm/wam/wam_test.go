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

package wam

import (
	"errors"
	"math"
	"testing"

	"github.com/ossf/criticality_score/internal/scorer/algorithm"
)

func TestWeighetedArithmeticMean_Score(t *testing.T) {
	//nolint:govet
	tests := []struct {
		name   string
		inputs []*algorithm.Input
		record map[string]float64
		want   float64
		err    error
	}{
		{
			name: "regular test",
			inputs: []*algorithm.Input{
				{
					Weight: 1, Distribution: algorithm.LookupDistribution("linear"),
					Source: algorithm.Value(algorithm.Field("1")),
				},
				{
					Weight: 2, Distribution: algorithm.LookupDistribution("linear"),
					Source: algorithm.Value(algorithm.Field("2")),
				},
				{
					Weight: 3, Distribution: algorithm.LookupDistribution("linear"),
					Source: algorithm.Value(algorithm.Field("3")),
				},
				{
					Weight: 4, Distribution: algorithm.LookupDistribution("linear"),
					Source: algorithm.Value(algorithm.Field("4")),
				},
				{
					Weight: 5, Distribution: algorithm.LookupDistribution("linear"),
					Source: algorithm.Value(algorithm.Field("5")),
				},
			},
			want:   3.6666666666666665,
			record: map[string]float64{"1": 1, "2": 2, "3": 3, "4": 4, "5": 5},
		},
		{
			name: "fields not matching the record",
			inputs: []*algorithm.Input{
				{
					Weight: 1, Distribution: algorithm.LookupDistribution("linear"),
					Source: algorithm.Value(algorithm.Field("1")),
				},
			},
			want:   math.NaN(),
			record: map[string]float64{"2": 2},
		},
		{
			name: "some fields matching the record",
			inputs: []*algorithm.Input{
				{
					Weight: 1, Distribution: algorithm.LookupDistribution("linear"),
					Source: algorithm.Value(algorithm.Field("1")),
				},
				{
					Weight: 2, Distribution: algorithm.LookupDistribution("linear"),
					Source: algorithm.Value(algorithm.Field("2")),
				},
			},
			want:   1,
			record: map[string]float64{"1": 1},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p, err := New(test.inputs)

			if err != nil && !errors.Is(err, test.err) {
				t.Errorf("New() error = %v, wantErr %v", err, test.err)
			}
			if err != nil {
				return
			}
			got := p.Score(test.record)

			if !(math.IsNaN(got) && math.IsNaN(test.want)) && got != test.want {
				t.Errorf("Score() = %v, want %v", got, test.want)
			}
		})
	}
}
