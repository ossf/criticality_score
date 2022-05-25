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
