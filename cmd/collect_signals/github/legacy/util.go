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

package legacy

import (
	"math"
	"time"
)

// empty is a convenience wrapper for the empty struct.
type empty struct{}

func TimeDelta(a, b time.Time, u time.Duration) int {
	var d time.Duration
	if a.Before(b) {
		d = b.Sub(a)
	} else {
		d = a.Sub(b)
	}
	return int(d / u)
}

// Round will return v approximately rounded to a precision of p decimal places.
func Round(v float64, p int) float64 {
	m := math.Pow10(p)
	return math.Round(v*m) / m
}
