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

// The package wam implements the Weighted Arithmetic Mean, which forms the
// basis of Rob Pike's criticality score algorithm as documented in
// Quantifying_criticality_algorithm.pdf.
package wam

import (
	"github.com/ossf/criticality_score/internal/scorer/algorithm"
)

type WeighetedArithmeticMean struct {
	inputs []*algorithm.Input
}

// New returns a new instance of the Weighted Arithmetic Mean algorithm, which
// is used by the Pike algorithm.
func New(inputs []*algorithm.Input) (algorithm.Algorithm, error) {
	return &WeighetedArithmeticMean{
		inputs: inputs,
	}, nil
}

func (p *WeighetedArithmeticMean) Score(record map[string]float64) float64 {
	var totalWeight float64
	var s float64
	for _, i := range p.inputs {
		v, ok := i.Value(record)
		if !ok {
			continue
		}
		totalWeight += i.Weight
		s += i.Weight * v
	}
	return s / totalWeight
}

func init() {
	algorithm.Register("weighted_arithmetic_mean", New)
}
