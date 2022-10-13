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

// Package collector defines a registry for using signal sources together.
package collector

import (
	"context"
	"fmt"

	"github.com/ossf/criticality_score/cmd/collect_signals/projectrepo"
	"github.com/ossf/criticality_score/cmd/collect_signals/signal"
)

// empty is a convenience wrapper for the empty struct.
type empty struct{}

var globalRegistry = NewRegistry()

type Registry struct {
	ss []signal.Source
}

// NewRegistry creates a new instance of Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// containsSource returns true if c has already been registered.
func (r *Registry) containsSource(s signal.Source) bool {
	for _, regS := range r.ss {
		if regS == s {
			return true
		}
	}
	return false
}

// Register adds the Source s to the registry to be used when Collect is called.
//
// This method may panic if the Source's signal Set is not valid, or if the
// Source has already been added.
//
// The order which Sources are added is preserved.
func (r *Registry) Register(s signal.Source) {
	validateSource(s)
	if r.containsSource(s) {
		panic(fmt.Sprintf("source %s has already been registered", s.EmptySet().Namespace()))
	}
	if err := signal.ValidateSet(s.EmptySet()); err != nil {
		panic(err)
	}
	r.ss = append(r.ss, s)
}

func (r *Registry) sourcesForRepository(repo projectrepo.Repo) []signal.Source {
	// Check for duplicates using a map to preserve the insertion order
	// of the sources.
	exists := make(map[signal.Namespace]empty)
	var res []signal.Source
	for _, s := range r.ss {
		if !s.IsSupported(repo) {
			continue
		}
		if _, ok := exists[s.EmptySet().Namespace()]; ok {
			// This key'd source already exists for this repo.
			panic("")
		}
		// Record that we have seen this key
		exists[s.EmptySet().Namespace()] = empty{}
		res = append(res, s)
	}
	return res
}

// EmptySets returns all the empty signal Sets for all the registered
// Sources.
//
// This result can be used to determine all the signals that are defined.
//
// The order of each empty Set is the same as the order of registration. If two
// Sources return a Set with the same Namespace, only the first Set will be
// included.
func (r *Registry) EmptySets() []signal.Set {
	exists := make(map[signal.Namespace]empty)
	var ss []signal.Set
	for _, c := range r.ss {
		// skip existing namespaces
		if _, ok := exists[c.EmptySet().Namespace()]; ok {
			continue
		}
		ss = append(ss, c.EmptySet())
	}
	return ss
}

// Collect will collect all the signals for the given repo.
func (r *Registry) Collect(ctx context.Context, repo projectrepo.Repo) ([]signal.Set, error) {
	cs := r.sourcesForRepository(repo)
	var ss []signal.Set
	for _, c := range cs {
		s, err := c.Get(ctx, repo)
		if err != nil {
			return nil, err
		}
		ss = append(ss, s)
	}
	return ss, nil
}

// Register registers the source with the global registry for use during
// calls to Collect().
//
// See Registry.Register().
func Register(s signal.Source) {
	globalRegistry.Register(s)
}

// EmptySet returns all the empty signal Sets for all the Sources registered
// with the global registry.
//
// See Registry.EmptySets().
func EmptySets() []signal.Set {
	return globalRegistry.EmptySets()
}

// Collect collects all the signals for the given repo using the Sources
// registered with the global registry.
//
// See Registry.Collect().
func Collect(ctx context.Context, r projectrepo.Repo) ([]signal.Set, error) {
	return globalRegistry.Collect(ctx, r)
}

func validateSource(s signal.Source) {
	// TODO - ensure a source with the same Namespace as another use
	// the same signal.Set
}
