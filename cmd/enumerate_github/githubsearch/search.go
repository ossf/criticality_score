package githubsearch

import (
	"context"
	"errors"

	"github.com/shurcooL/githubv4"
	log "github.com/sirupsen/logrus"
)

// empty is a convenience wrapper for the empty struct.
type empty struct{}

var ErrorUnableToListAllResult = errors.New("unable to list all results")

type Searcher struct {
	ctx     context.Context
	client  *githubv4.Client
	logger  *log.Entry
	perPage int
}

type Option interface{ set(*Searcher) }
type option func(*Searcher)      // option implements Option.
func (o option) set(s *Searcher) { o(s) }

// PerPage will set how many results will per requested per page for each search query.
func PerPage(perPage int) Option {
	return option(func(s *Searcher) { s.perPage = perPage })
}

func NewSearcher(ctx context.Context, client *githubv4.Client, logger *log.Entry, options ...Option) *Searcher {
	s := &Searcher{
		ctx:     ctx,
		client:  client,
		logger:  logger,
		perPage: 100,
	}
	for _, o := range options {
		o.set(s)
	}
	return s
}
