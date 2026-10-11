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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

// countPageSize is the page size used when results have to be counted by
// paging through them. 100 is the largest page size GitHub allows.
const countPageSize = 100

// FetchIssueCount returns the total number of issues for a given repo in a
// given state, across the past lookback duration.
//
// This count includes both issues and pull requests.
func FetchIssueCount(ctx context.Context, c *githubapi.Client, owner, name string, state IssueState, lookback time.Duration) (int, error) {
	since := time.Now().UTC().Add(-lookback)
	opts := &github.IssueListByRepoOptions{
		Since:       since,
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
	if nextLink(resp) == "" {
		return len(is), nil
	}
	if resp.LastPage != 0 {
		return resp.LastPage, nil
	}

	// GitHub returned a "next" link without a "last" link (cursor-based
	// pagination), so the total is not available from a single response.
	query := url.Values{}
	query.Set("state", string(state))
	query.Set("since", since.Format(time.RFC3339))
	n, err := countByPaging(ctx, c.Rest(), fmt.Sprintf("repos/%s/%s/issues", owner, name), query, MaxIssuesLimit)
	if c := githubapi.ErrorResponseStatusCode(err); 500 <= c && c < 600 {
		return MaxIssuesLimit, nil
	}
	if err != nil {
		return 0, err
	}
	return n, nil
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
	if nextLink(resp) == "" {
		return len(cs), nil
	}
	if resp.LastPage != 0 {
		return resp.LastPage, nil
	}

	// No "last" link: count by paging, as in FetchIssueCount.
	query := url.Values{}
	query.Set("since", since.Format(time.RFC3339))
	n, err := countByPaging(ctx, c.Rest(), fmt.Sprintf("repos/%s/%s/issues/comments", owner, name), query, MaxIssuesLimit)
	if c := githubapi.ErrorResponseStatusCode(err); 500 <= c && c < 600 {
		return 0, ErrorTooManyResults
	}
	if err != nil {
		return 0, err
	}
	if n >= MaxIssuesLimit {
		return 0, ErrorTooManyResults
	}
	return n, nil
}

// countByPaging counts the results of a GitHub list endpoint by following the
// rel="next" links of its responses. It stops once limit results have been
// seen and returns limit in that case.
func countByPaging(ctx context.Context, client *github.Client, path string, query url.Values, limit int) (int, error) {
	query.Set("per_page", strconv.Itoa(countPageSize))
	next := path + "?" + query.Encode()
	total := 0
	for next != "" {
		req, err := client.NewRequest(http.MethodGet, next, nil)
		if err != nil {
			return 0, err
		}
		var page []json.RawMessage
		resp, err := client.Do(ctx, req, &page)
		if err != nil {
			return 0, err
		}
		total += len(page)
		if total >= limit {
			return limit, nil
		}
		if len(page) == 0 {
			break // Guard against a "next" link that never ends.
		}
		next = nextLink(resp)
	}
	return total, nil
}

// nextLink returns the URL of the rel="next" link in a response's Link
// header, or "" when there is none.
//
// github.Response.NextPage is not enough on its own: with cursor-based
// pagination the "next" link may carry only an "after" cursor and no page
// number.
func nextLink(resp *github.Response) string {
	if resp == nil || resp.Response == nil {
		return ""
	}
	for _, link := range strings.Split(resp.Header.Get("Link"), ",") {
		segments := strings.Split(strings.TrimSpace(link), ";")
		if len(segments) < 2 {
			continue
		}
		target := strings.TrimSpace(segments[0])
		if !strings.HasPrefix(target, "<") || !strings.HasSuffix(target, ">") {
			continue
		}
		for _, segment := range segments[1:] {
			if strings.TrimSpace(segment) == `rel="next"` {
				return target[1 : len(target)-1]
			}
		}
	}
	return ""
}
