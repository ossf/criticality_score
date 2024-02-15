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

package pagination

import (
	"context"
	"io"

	"github.com/hasura/go-graphql-client"
)

// PagedQuery implementors go from being regular query struct passed to githubv4.Query()
// to a query that can be paginated.
type PagedQuery interface {
	Total() int
	Length() int
	Get(int) any
	Reset()
	HasNextPage() bool
	NextPageVars() map[string]any
}

type Cursor struct {
	ctx    context.Context
	client *graphql.Client
	query  PagedQuery
	vars   map[string]any
	cur    int
}

func Query(ctx context.Context, client *graphql.Client, query PagedQuery, vars map[string]any) (*Cursor, error) {
	c := &Cursor{
		ctx:    ctx,
		client: client,
		query:  query,
		vars:   vars,
	}
	if err := c.queryNextPage(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Cursor) queryNextPage() error {
	// Merge the next page vars with the current vars
	newVars := c.query.NextPageVars()
	for k, v := range newVars {
		c.vars[k] = v
	}
	// Reset the current position
	c.cur = 0
	// ZERO the query...
	c.query.Reset()
	// Execute the query
	return c.client.Query(c.ctx, c.query, c.vars)
}

func (c *Cursor) atEndOfPage() bool {
	return c.cur >= c.query.Length()
}

func (c *Cursor) isLastPage() bool {
	return !c.query.HasNextPage()
}

func (c *Cursor) Total() int {
	return c.query.Total()
}

func (c *Cursor) Next() (any, error) {
	if c.atEndOfPage() {
		// There are no more nodes in this page, so we need another page.
		if c.isLastPage() {
			// There are no more pages, so return an EOF
			return nil, io.EOF
		}
		// Grab the next page.
		if err := c.queryNextPage(); err != nil {
			return nil, err
		}
		if c.atEndOfPage() {
			// Despite grabing a new page we have no results,
			// so return an EOF.
			return nil, io.EOF
		}
	}
	val := c.query.Get(c.cur)
	c.cur++
	return val, nil
}
