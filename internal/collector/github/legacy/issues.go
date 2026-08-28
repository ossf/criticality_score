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

package legacy

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v47/github"
	"github.com/hasura/go-graphql-client"

	"github.com/ossf/criticality_score/v2/internal/githubapi"
)

// IssueCounts holds the issue totals for a repository over a lookback window.
//
// Both totals include pull requests as well as issues, matching the semantics
// of the REST endpoint these counts used to come from.
type IssueCounts struct {
	// Updated is the number of issues and pull requests touched in the window.
	Updated int

	// Closed is the subset of those that are currently closed.
	Closed int
}

// issueSearchQuery counts issues and pull requests in one request.
//
// GitHub's search index treats a pull request as a kind of issue, but a single
// search cannot return the two counts separately, so each total needs its own.
// All four are aliased into one request. That halves the round trips, and more
// importantly it gives every count the same cutoff: issuing them separately
// let the updated and closed totals fall either side of a boundary and
// disagree.
//
// `first: 1` is not needed to read issueCount, which is a scalar on the
// connection rather than a traversal of its nodes, but it costs nothing and
// states the pagination boundary explicitly.
type issueSearchQuery struct {
	UpdatedIssues struct {
		IssueCount int
	} `graphql:"updatedIssues:search(query:$updatedIssueQuery, type:ISSUE, first:1)"`

	UpdatedPullRequests struct {
		IssueCount int
	} `graphql:"updatedPullRequests:search(query:$updatedPullRequestQuery, type:ISSUE, first:1)"`

	ClosedIssues struct {
		IssueCount int
	} `graphql:"closedIssues:search(query:$closedIssueQuery, type:ISSUE, first:1)"`

	ClosedPullRequests struct {
		IssueCount int
	} `graphql:"closedPullRequests:search(query:$closedPullRequestQuery, type:ISSUE, first:1)"`
}

// FetchIssueCounts returns how many issues and pull requests for a repo were
// updated since the supplied time, and how many of those are closed.
//
// The counts come from GitHub's search index rather than from paginating the
// REST issues endpoint. That endpoint moved to cursor based pagination and no
// longer returns a "last" link, so the page count it used to be read from is
// always absent, and every repository with more than one page of issues
// reported zero.
//
// Search returns an exact total regardless of size, so unlike the REST
// approach there is no need to cap the result.
func FetchIssueCounts(ctx context.Context, c *githubapi.Client, owner, name string, since time.Time) (IssueCounts, error) {
	// One cutoff shared by all four counts, and by the comment count the
	// caller divides them into, so every part of comment frequency covers the
	// same window. It is a full timestamp because the search index accepts
	// RFC3339; truncating to a date moves the boundary by up to a day.
	base := fmt.Sprintf("repo:%s/%s updated:>=%s", owner, name, since.UTC().Format(time.RFC3339))

	// The is:issue and is:pull-request qualifiers are required. Without one of
	// them the search returns a count of zero rather than an error, which would
	// silently zero the signal.
	vars := map[string]any{
		"updatedIssueQuery":       graphql.String("is:issue " + base),
		"updatedPullRequestQuery": graphql.String("is:pull-request " + base),
		"closedIssueQuery":        graphql.String("is:issue is:closed " + base),
		"closedPullRequestQuery":  graphql.String("is:pull-request is:closed " + base),
	}

	s := &issueSearchQuery{}
	if err := c.GraphQL().Query(ctx, s, vars); err != nil {
		return IssueCounts{}, fmt.Errorf("issue search: %w", err)
	}
	return IssueCounts{
		Updated: s.UpdatedIssues.IssueCount + s.UpdatedPullRequests.IssueCount,
		Closed:  s.ClosedIssues.IssueCount + s.ClosedPullRequests.IssueCount,
	}, nil
}

// commentCountRetries is how many times a transient server error is retried
// before the comment total is treated as unavailable.
//
// The shared HTTP retry layer deliberately does not retry this endpoint, so
// the retry has to happen here.
const (
	commentCountRetries    = 2
	commentCountRetryDelay = 2 * time.Second
)

// sleepContext waits for d, or returns early if the context is cancelled.
func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("waiting to retry: %w", ctx.Err())
	case <-t.C:
		return nil
	}
}

// FetchIssueCommentCount returns the total number of comments for a given repo
// across all issues and pull requests, since the supplied time.
//
// The caller passes `since` rather than a lookback so that this count and the
// issue counts it is divided by share one window. Deriving the cutoff
// separately in each function let the numerator and denominator of comment
// frequency cover slightly different periods.
//
// If the total cannot be established, ErrorTooManyResults is returned wrapping
// the underlying cause.
func FetchIssueCommentCount(ctx context.Context, c *githubapi.Client, owner, name string, since time.Time) (int, error) {
	opts := &github.IssueListCommentsOptions{
		Since:       &since,
		ListOptions: github.ListOptions{PerPage: 1}, // 1 result per page means LastPage is total number of records.
	}

	var cs []*github.IssueComment
	var resp *github.Response
	var err error
	for attempt := 0; ; attempt++ {
		cs, resp, err = c.Rest().Issues.ListComments(ctx, owner, name, 0, opts)
		status := githubapi.ErrorResponseStatusCode(err)
		if status < 500 || status >= 600 {
			break
		}
		// GitHub answers 5xx both when a repository has more comments than it
		// will count and when something is transiently wrong. Treating every
		// 5xx as the former was wrong: kubernetes/kubernetes returned one on a
		// single run and the real count on the next. Retry first, and only
		// give up once the failure persists.
		if attempt >= commentCountRetries {
			return 0, fmt.Errorf("%w: comment count unavailable after %d attempts: %w",
				ErrorTooManyResults, attempt+1, err)
		}
		if err := sleepContext(ctx, commentCountRetryDelay); err != nil {
			return 0, err
		}
	}
	if err != nil {
		return 0, err
	}
	if resp.NextPage == 0 {
		return len(cs), nil
	}
	if resp.LastPage == 0 {
		// A "next" link without a "last" link means this endpoint has moved to
		// cursor based pagination and the page count is no longer available.
		// Report it rather than silently returning zero.
		return 0, ErrorNoPageCount
	}
	return resp.LastPage, nil
}
