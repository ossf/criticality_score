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

func (c *Cursor) Finished() bool {
	return c.atEndOfPage() && c.isLastPage()
}

func (c *Cursor) Total() int {
	return c.query.Total()
}

func (c *Cursor) Next() (interface{}, error) {
	if c.Finished() {
		// We've finished so return an EOF
		return nil, io.EOF
	} else if c.atEndOfPage() {
		// We're at the end of the page, but not finished, so grab the next page.
		if err := c.queryNextPage(); err != nil {
			return nil, err
		}
		// Make sure we didn't get an empty result.
		if c.atEndOfPage() {
			return nil, io.EOF
		}
	}
	val := c.query.Get(c.cur)
	c.cur++
	return val, nil
}
