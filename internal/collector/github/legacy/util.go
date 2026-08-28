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

// TimeDelta returns how many whole units u have elapsed from b to a.
//
// If b is after a the result is 0. Timestamps that come from Git are not
// trustworthy: they are written by the committer's machine, are not validated
// by GitHub, and are trivially forged. Taking the absolute difference here
// would turn a timestamp far in the future into a large positive age, so a
// repository could claim to be a decade old, or to have been updated a decade
// ago, by pushing one commit with a bad clock.
func TimeDelta(a, b time.Time, u time.Duration) int {
	if a.Before(b) {
		return 0
	}
	return int(a.Sub(b) / u)
}

// Round will return v approximately rounded to a precision of p decimal places.
func Round(v float64, p int) float64 {
	m := math.Pow10(p)
	return math.Round(v*m) / m
}
