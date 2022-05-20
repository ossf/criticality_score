package legacy

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/ossf/criticality_score/internal/githubapi"
	"github.com/ossf/criticality_score/internal/githubapi/pagination"
	"github.com/shurcooL/githubv4"
)

type repoReleasesQuery struct {
	Repository struct {
		Releases struct {
			TotalCount int
			Nodes      []struct {
				Release struct {
					CreatedAt time.Time
				} `graphql:"... on Release"`
			}
			PageInfo struct {
				EndCursor   string
				HasNextPage bool
			}
		} `graphql:"releases(orderBy:{direction:DESC, field:CREATED_AT}, first: $perPage, after: $endCursor)"`
	} `graphql:"repository(owner: $repositoryOwner, name: $repositoryName)"`
}

// Total implements the pagination.PagedQuery interface
func (r *repoReleasesQuery) Total() int {
	return r.Repository.Releases.TotalCount
}

// Length implements the pagination.PagedQuery interface
func (r *repoReleasesQuery) Length() int {
	return len(r.Repository.Releases.Nodes)
}

// Get implements the pagination.PagedQuery interface
func (r *repoReleasesQuery) Get(i int) interface{} {
	return r.Repository.Releases.Nodes[i].Release.CreatedAt
}

// HasNextPage implements the pagination.PagedQuery interface
func (r *repoReleasesQuery) HasNextPage() bool {
	return r.Repository.Releases.PageInfo.HasNextPage
}

// NextPageVars implements the pagination.PagedQuery interface
func (r *repoReleasesQuery) NextPageVars() map[string]interface{} {
	if r.Repository.Releases.PageInfo.EndCursor == "" {
		return map[string]interface{}{
			"endCursor": (*githubv4.String)(nil),
		}
	} else {
		return map[string]interface{}{
			"endCursor": githubv4.String(r.Repository.Releases.PageInfo.EndCursor),
		}
	}
}

func FetchReleaseCount(ctx context.Context, c *githubapi.Client, owner, name string, lookback time.Duration) (int, error) {
	s := &repoReleasesQuery{}
	vars := map[string]interface{}{
		"perPage":         githubv4.Int(releasesPerPage),
		"endCursor":       githubv4.String(owner),
		"repositoryOwner": githubv4.String(owner),
		"repositoryName":  githubv4.String(name),
	}
	cursor, err := pagination.Query(ctx, c.GraphQL(), s, vars)
	if err != nil {
		return 0, err
	}
	cutoff := time.Now().UTC().Add(-lookback)
	total := 0
	for {
		obj, err := cursor.Next()
		if obj == nil && errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return 0, err
		}
		releaseCreated := obj.(time.Time)
		if releaseCreated.Before(cutoff) {
			break
		} else {
			total++
		}
	}
	return total, nil
}
