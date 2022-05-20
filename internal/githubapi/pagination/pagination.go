package pagination

import (
	"context"
	"io"

	"github.com/shurcooL/githubv4"
)

// PagedQuery implementors go from being regular query struct passed to githubv4.Query()
// to a query that can be paginated.
type PagedQuery interface {
	Total() int
	Length() int
	Get(int) interface{}
	HasNextPage() bool
	NextPageVars() map[string]interface{}
}

type Cursor struct {
	ctx    context.Context
	client *githubv4.Client
	query  PagedQuery
	vars   map[string]interface{}
	cur    int
}

func Query(ctx context.Context, client *githubv4.Client, query PagedQuery, vars map[string]interface{}) (*Cursor, error) {
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

func (c *Cursor) Next() (interface{}, error) {
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
