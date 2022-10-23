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

package signal

import "time"

//nolint:govet
type RepoSet struct {
	URL      Field[string]
	Language Field[string]
	License  Field[string]

	StarCount Field[int]
	CreatedAt Field[time.Time]
	UpdatedAt Field[time.Time]

	CreatedSince Field[int] `signal:"legacy"`
	UpdatedSince Field[int] `signal:"legacy"`

	ContributorCount Field[int] `signal:"legacy"`
	OrgCount         Field[int] `signal:"legacy"`

	CommitFrequency    Field[float64] `signal:"legacy"`
	RecentReleaseCount Field[int]     `signal:"legacy"`
}

func (r *RepoSet) Namespace() Namespace {
	return NamespaceRepo
}
