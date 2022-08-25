// Copyright 2022 Google LLC
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

type Bounds struct {
	Lower           float64 `yaml:"lower"`
	Upper           float64 `yaml:"upper"`
	SmallerIsBetter bool    `yaml:"smaller_is_better"`
}

func (b Bounds) Apply(v float64) float64 {
	// |----L---v----U----| == v stays as is
	// |--v-L--------U----| == v moves to L
	// |----L--------U--v-| == v moves to U
	if v < b.Lower {
		v = b.Lower
	} else if v > b.Upper {
		v = b.Upper
	}
	// Afterwards we move L to 0, by calculating v = v - L
	v = v - b.Lower
	if b.SmallerIsBetter {
		// If "SmallerIsBetter" is true then invert the value with the
		// threshold. So, a 0 value becomes the threshold value and a
		// value at the threshold becomes 0.
		// TODO: consider how this affects the distribution
		v = b.Threshold() - v
	}
	return v
}

func (b Bounds) Threshold() float64 {
	return b.Upper - b.Lower

}

type Input struct {
	Bounds       *Bounds
	Weight       float64
	Distribution *Distribution
	Source       Value
	Tags         []string
}

func (i *Input) Value(fields map[string]float64) (float64, bool) {
	v, ok := i.Source.Value(fields)
	if !ok {
		return 0, false
	}
	var den float64 = 1
	if i.Bounds != nil {
		v = i.Bounds.Apply(v)
		den = i.Distribution.Normalize(i.Bounds.Threshold())
	}
	return i.Distribution.Normalize(v) / den, true
}
