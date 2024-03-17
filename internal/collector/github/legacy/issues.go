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
	"time"

	"github.com/google/go-github/v47/github"

	"github.com/ossf/criticality_score/v2/internal/githubapi"
)

type IssueState string

const (
	IssueStateAll    = "all"
	IssueStateOpen   = "open"
	IssueStateClosed = "closed"
)

// FetchIssueCount returns the total number of issues for a given repo in a
// given state, across the past lookback duration.
//
// This count includes both issues and pull requests.
func FetchIssueCount(ctx context.Context, c *githubapi.Client, owner, name string, state IssueState, lookback time.Duration) (int, error) {
	opts := &github.IssueListByRepoOptions{
		Since:       time.Now().UTC().Add(-lookback),
		State:       string(state),
		ListOptions: github.ListOptions{PerPage: 1}, // 1 result per page means LastPage is total number of records.
	}
	is, resp, err := c.Rest().Issues.ListByRepo(ctx, owner, name, opts)
	// The API returns 5xx responses if there are too many issues.
	if c := githubapi.ErrorResponseStatusCode(err); 500 <= c && c < 600 {
		return MaxIssuesLimit, nil
	}
	if err != nil {
		return 0, err
	}
	if resp.NextPage == 0 {
		return len(is), nil
	}
	return resp.LastPage, nil
}

// FetchIssueCommentCount returns the total number of comments for a given repo
// across all issues and pull requests, for the past lookback duration.
//
// If the exact number if unable to be returned because there are too many
// results, a TooManyResultsError will be returned.
func FetchIssueCommentCount(ctx context.Context, c *githubapi.Client, owner, name string, lookback time.Duration) (int, error) {
	since := time.Now().UTC().Add(-lookback)
	opts := &github.IssueListCommentsOptions{
		Since:       &since,
		ListOptions: github.ListOptions{PerPage: 1}, // 1 result per page means LastPage is total number of records.
	}
	cs, resp, err := c.Rest().Issues.ListComments(ctx, owner, name, 0, opts)
	// The API returns 5xx responses if there are too many comments.
	if c := githubapi.ErrorResponseStatusCode(err); 500 <= c && c < 600 {
		return 0, ErrorTooManyResults
	}
	if err != nil {
		return 0, err
	}
	if resp.NextPage == 0 {
		return len(cs), nil
	}
	return resp.LastPage, nil
}
