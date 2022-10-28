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
	"errors"
	"fmt"
	"io"

	"github.com/shurcooL/githubv4"
	"go.uber.org/zap"

	"github.com/ossf/criticality_score/internal/githubapi/pagination"
)

// repo is part of the GitHub GraphQL query and includes the fields
// that will be populated in a response.
type repo struct {
	URL            string
	StargazerCount int
}

// repoQuery is a GraphQL query for iterating over repositories in GitHub.
type repoQuery struct {
	Search struct {
		Nodes []struct {
			Repository repo `graphql:"...on Repository"`
		}
		PageInfo struct {
			EndCursor   string
			HasNextPage bool
		}
		RepositoryCount int
	} `graphql:"search(type: REPOSITORY, query: $query, first: $perPage, after: $endCursor)"`
}

// Total implements the pagination.PagedQuery interface.
func (q *repoQuery) Total() int {
	return q.Search.RepositoryCount
}

// Length implements the pagination.PagedQuery interface.
func (q *repoQuery) Length() int {
	return len(q.Search.Nodes)
}

// Get implements the pagination.PagedQuery interface.
func (q *repoQuery) Get(i int) any {
	return q.Search.Nodes[i].Repository
}

// HasNextPage implements the pagination.PagedQuery interface.
func (q *repoQuery) HasNextPage() bool {
	return q.Search.PageInfo.HasNextPage
}

// NextPageVars implements the pagination.PagedQuery interface.
func (q *repoQuery) NextPageVars() map[string]any {
	if q.Search.PageInfo.EndCursor == "" {
		return map[string]any{
			"endCursor": (*githubv4.String)(nil),
		}
	} else {
		return map[string]any{
			"endCursor": githubv4.String(q.Search.PageInfo.EndCursor),
		}
	}
}

func buildQuery(q string, minStars, maxStars int) string {
	q = q + " sort:stars "
	if maxStars > 0 {
		return q + fmt.Sprintf("stars:%d..%d", minStars, maxStars)
	} else {
		return q + fmt.Sprintf("stars:>=%d", minStars)
	}
}

func (re *Searcher) runRepoQuery(q string) (*pagination.Cursor, error) {
	re.logger.With(
		zap.String("query", q),
	).Debug("Searching GitHub")
	vars := map[string]any{
		"query":   githubv4.String(q),
		"perPage": githubv4.Int(re.perPage),
	}
	return pagination.Query(re.ctx, re.client, &repoQuery{}, vars)
}

// ReposByStars will call emitter once for each repository returned when searching for baseQuery
// with at least minStars, order from the most stars, to the least.
//
// The emitter function is called with the repository's Url.
//
// The algorithm works to overcome the approx 1000 repository limit returned by a single search
// across 10 pages by:
// - Ordering GitHub's repositories from most stars to least.
// - Iterating through all the repositories returned by each query.
// - Getting the number of stars for the last repository returned.
// - Using this star value, plus an overlap, to be the maximum star limit.
//
// The algorithm fails if the last star value plus overlap has the same or larger value as the
// previous iteration.
func (re *Searcher) ReposByStars(baseQuery string, minStars, overlap int, emitter func(string)) error {
	repos := make(map[string]empty)
	maxStars := -1
	stars := 0

	for {
		q := buildQuery(baseQuery, minStars, maxStars)
		c, err := re.runRepoQuery(q)
		if err != nil {
			return err
		}

		total := c.Total()
		seen := 0
		stars = 0
		for {
			obj, err := c.Next()
			if obj == nil && errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				return err
			}
			repo := obj.(repo)
			seen++
			stars = repo.StargazerCount
			if _, ok := repos[repo.URL]; !ok {
				repos[repo.URL] = empty{}
				emitter(repo.URL)
			}
		}
		remaining := total - seen
		re.logger.With(
			zap.Int("total_available", total),
			zap.Int("total_returned", seen),
			zap.Int("total_remaining", remaining),
			zap.Int("unique_repos", len(repos)),
			zap.Int("last_stars", stars),
			zap.String("query", q),
		).Debug("Finished iterating through results")
		newMaxStars := stars + overlap
		switch {
		case remaining <= 0:
			// nothing remains, we are done.
			return nil
		case maxStars == -1, newMaxStars < maxStars:
			maxStars = newMaxStars
		default:
			// the gap between "stars" and "maxStars" is less than "overlap", so we can't
			// safely step any lower without skipping stars.
			re.logger.With(
				zap.Error(ErrorUnableToListAllResult),
				zap.Int("min_stars", minStars),
				zap.Int("stars", stars),
				zap.Int("max_stars", maxStars),
				zap.Int("overlap", overlap),
			).Error("Too many repositories for current range")
			return ErrorUnableToListAllResult
		}
	}
}
