package collector

import (
	"context"

	"github.com/ossf/criticality_score/cmd/collect_signals/projectrepo"
	"github.com/ossf/criticality_score/cmd/collect_signals/signal"
)

type Collector interface {
	EmptySet() signal.Set
	Collect(context.Context, projectrepo.Repo) (signal.Set, error)
}

type LimitedCollector interface {
	Collector
	IsSupported(projectrepo.Repo) bool
}

// collectorWrapper turns a Collector into an implementation
// of LimitedCollector.
type collectorWrapper struct {
	Collector
}

// IsSupported implements the LimitedCollector interface.
func (w *collectorWrapper) IsSupported(r projectrepo.Repo) bool {
	switch c := w.Collector.(type) {
	case LimitedCollector:
		return c.IsSupported(r)
	default:
		return true
	}
}

func MakeLimitedCollector(c Collector) LimitedCollector {
	return &collectorWrapper{Collector: c}
}
