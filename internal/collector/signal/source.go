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

import (
	"context"

	"github.com/ossf/criticality_score/v2/internal/collector/projectrepo"
)

// A Source is used to get a set of signals for a given project repository.
type Source interface {
	// EmptySet returns an empty instance of a signal Set that can be used for
	// determining the namespace and signals supported by the Source.
	EmptySet() Set

	// IsSupported returns true if the Source supports the supplied Repo.
	IsSupported(projectrepo.Repo) bool

	// Get gathers and returns a Set of signals for the given project repo r.
	//
	// An optional string jobID can be specified and may be used by the Source
	// to manage caches related to a collection run.
	//
	// An error is returned if it is unable to successfully gather the signals,
	// or if the context is cancelled.
	Get(ctx context.Context, r projectrepo.Repo, jobID string) (Set, error)
}
