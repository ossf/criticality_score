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
	"testing"
	"time"
)

func TestTimeDelta(t *testing.T) {
	now := time.Date(2024, time.June, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		a    time.Time
		b    time.Time
		name string
		unit time.Duration
		want int
	}{
		{
			name: "b is one unit before a",
			a:    now,
			b:    now.Add(-SinceDuration),
			unit: SinceDuration,
			want: 1,
		},
		{
			name: "b is many units before a",
			a:    now,
			b:    now.Add(-10 * SinceDuration),
			unit: SinceDuration,
			want: 10,
		},
		{
			name: "partial units are truncated",
			a:    now,
			b:    now.Add(-SinceDuration - time.Hour),
			unit: SinceDuration,
			want: 1,
		},
		{
			name: "identical times",
			a:    now,
			b:    now,
			unit: SinceDuration,
			want: 0,
		},
		{
			// Git timestamps are supplied by the committer and are not
			// validated, so a commit can be dated in the future. Measuring the
			// absolute difference would report such a repository as very old,
			// or as having gone unmaintained for years, whichever helps it.
			name: "b in the future is clamped to zero",
			a:    now,
			b:    now.Add(100 * SinceDuration),
			unit: SinceDuration,
			want: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := TimeDelta(test.a, test.b, test.unit); got != test.want {
				t.Errorf("TimeDelta() = %d, want %d", got, test.want)
			}
		})
	}
}
