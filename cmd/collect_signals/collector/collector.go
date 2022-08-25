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

// Package collector defines the interface for signal collectors and a registry
// for using the collectors together.
package collector

import (
	"context"

	"github.com/ossf/criticality_score/cmd/collect_signals/projectrepo"
	"github.com/ossf/criticality_score/cmd/collect_signals/signal"
)

// A Collector is used to collect a set of signals for a given
// project repository.
type Collector interface {
	// EmptySet returns an empty instance of a signal Set that can be used for
	// determining the namespace and signals supported by the Collector.
	EmptySet() signal.Set

	// IsSupported returns true if the Collector supports the supplied Repo.
	IsSupported(projectrepo.Repo) bool

	// Collect gathers and returns a Set of signals for the given project repo.
	//
	// An error is returned if it is unable to successfully gather the signals,
	// or if the context is cancelled.
	Collect(context.Context, projectrepo.Repo) (signal.Set, error)
}
