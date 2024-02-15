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

package githubsearch

import (
	"context"
	"errors"

	"github.com/hasura/go-graphql-client"

	"go.uber.org/zap"
)

// empty is a convenience wrapper for the empty struct.
type empty struct{}

var ErrorUnableToListAllResult = errors.New("unable to list all results")

type Searcher struct {
	ctx     context.Context
	client  *graphql.Client
	logger  *zap.Logger
	perPage int
}

type Option interface {
	set(*Searcher)
}

// option implements Option.
type option func(*Searcher)

func (o option) set(s *Searcher) { o(s) }

// PerPage will set how many results will per requested per page for each search query.
func PerPage(perPage int) Option {
	return option(func(s *Searcher) { s.perPage = perPage })
}

func NewSearcher(ctx context.Context, client *graphql.Client, logger *zap.Logger, options ...Option) *Searcher {
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
