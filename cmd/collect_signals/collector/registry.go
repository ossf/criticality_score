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
	cs []Collector
}

// NewRegistry creates a new instance of Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// containsCollector returns true if c has already been registered.
func (r *Registry) containsCollector(c Collector) bool {
	for _, regC := range r.cs {
		if regC == c {
			return true
		}
	}
	return false
}

// Register adds the Collector c to the registry to be used when Collect is
// called.
//
// This method may panic if the Collector's signal Set is not valid, or if the
// Collector has already been added.
//
// The order which Collectors are added is preserved.
func (r *Registry) Register(c Collector) {
	validateCollector(c)
	if r.containsCollector(c) {
		panic(fmt.Sprintf("collector %s has already been registered", c.EmptySet().Namespace()))
	}
	if err := signal.ValidateSet(c.EmptySet()); err != nil {
		panic(err)
	}
	r.cs = append(r.cs, c)
}

func (r *Registry) collectorsForRepository(repo projectrepo.Repo) []Collector {
	// Check for duplicates using a map to preserve the insertion order
	// of the collectors.
	exists := make(map[signal.Namespace]empty)
	var res []Collector
	for _, c := range r.cs {
		if !c.IsSupported(repo) {
			continue
		}
		if _, ok := exists[c.EmptySet().Namespace()]; ok {
			// This key'd collector already exists for this repo.
			panic("")
		}
		// Record that we have seen this key
		exists[c.EmptySet().Namespace()] = empty{}
		res = append(res, c)
	}
	return res
}

// EmptySets returns all the empty signal Sets for all the registered
// Collectors.
//
// This result can be used to determine all the signals that are defined.
//
// The order of each empty Set is the same as the order of registration. If two
// Collectors return a Set with the same Namespace, only the first Set will be
// included.
func (r *Registry) EmptySets() []signal.Set {
	exists := make(map[signal.Namespace]empty)
	var ss []signal.Set
	for _, c := range r.cs {
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
	cs := r.collectorsForRepository(repo)
	var ss []signal.Set
	for _, c := range cs {
		s, err := c.Collect(ctx, repo)
		if err != nil {
			return nil, err
		}
		ss = append(ss, s)
	}
	return ss, nil
}

// Register registers the collector with the global registry for use during
// calls to Collect().
//
// See Registry.Register().
func Register(c Collector) {
	globalRegistry.Register(c)
}

// EmptySet returns all the empty signal Sets for all the Collectors registered
// with the global registry.
//
// See Registry.EmptySets().
func EmptySets() []signal.Set {
	return globalRegistry.EmptySets()
}

// Collect collects all the signals for the given repo using the Collectors
// registered with the global registry.
//
// See Registry.Collect().
func Collect(ctx context.Context, r projectrepo.Repo) ([]signal.Set, error) {
	return globalRegistry.Collect(ctx, r)
}

func validateCollector(c Collector) {
	// TODO - ensure a collector with the same Namespace as another use
	// the same signal.Set
}
