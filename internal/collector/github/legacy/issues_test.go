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
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ossf/criticality_score/v2/internal/githubapi"
)

// captureTransport serves a canned GraphQL response and records the request so
// the generated query can be asserted on.
//
// Asserting on the query matters more than it looks. The transport answers
// whatever it is asked, so without these checks the tests pass for any query
// the client happens to build, including one the real API would reject or,
// worse, one it answers with zero.
type captureTransport struct {
	body     string
	response string
	status   int
}

func (t *captureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Body != nil {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
		t.body = string(b)
	}
	status := t.status
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(t.response)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Request:    r,
	}, nil
}

func newTestClient(response string) (*githubapi.Client, *captureTransport) {
	tr := &captureTransport{response: response}
	return githubapi.NewClient(&http.Client{Transport: tr}), tr
}

const allCountsResponse = `{"data":{
	"updatedIssues":{"issueCount":1868},
	"updatedPullRequests":{"issueCount":2842},
	"closedIssues":{"issueCount":1024},
	"closedPullRequests":{"issueCount":1847}}}`

// request is the decoded GraphQL request body.
type request struct {
	Variables map[string]string `json:"variables"`
	Query     string            `json:"query"`
}

// testSince is the fixed window start used by these tests.
func testSince() time.Time {
	return time.Date(2024, time.March, 1, 12, 30, 45, 0, time.UTC)
}

func decodeRequest(t *testing.T, body string) request {
	t.Helper()
	var req request
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("failed to parse request body %q: %v", body, err)
	}
	return req
}

// TestFetchIssueCountsSumsIssuesAndPullRequests locks in each total being the
// sum of the issue and pull request counts. The REST endpoint this replaced
// counted them together, so dropping either would silently shrink the signal.
func TestFetchIssueCountsSumsIssuesAndPullRequests(t *testing.T) {
	c, _ := newTestClient(allCountsResponse)

	got, err := FetchIssueCounts(context.Background(), c, "kubernetes", "kubernetes", testSince())
	if err != nil {
		t.Fatalf("FetchIssueCounts() = %v, want nil error", err)
	}
	if want := 1868 + 2842; got.Updated != want {
		t.Errorf("Updated = %d, want %d", got.Updated, want)
	}
	if want := 1024 + 1847; got.Closed != want {
		t.Errorf("Closed = %d, want %d", got.Closed, want)
	}
}

// TestFetchIssueCountsUsesOneRequest pins the counts being gathered together.
// Splitting them across requests doubles the cost and lets the two totals be
// taken against different cutoffs.
func TestFetchIssueCountsUsesOneRequest(t *testing.T) {
	tr := &captureTransport{response: allCountsResponse}
	requests := 0
	c := githubapi.NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		return tr.RoundTrip(r)
	})})

	if _, err := FetchIssueCounts(context.Background(), c, "o", "n", testSince()); err != nil {
		t.Fatalf("FetchIssueCounts() = %v, want nil error", err)
	}
	if requests != 1 {
		t.Errorf("made %d requests, want 1", requests)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// TestFetchIssueCountsQueryShape asserts the structure of the generated query
// rather than only its variables, because an API contract error shows up in
// the query text and not in the values substituted into it.
func TestFetchIssueCountsQueryShape(t *testing.T) {
	c, tr := newTestClient(allCountsResponse)
	if _, err := FetchIssueCounts(context.Background(), c, "o", "n", testSince()); err != nil {
		t.Fatalf("FetchIssueCounts() = %v, want nil error", err)
	}
	req := decodeRequest(t, tr.body)

	aliases := []string{"updatedIssues", "updatedPullRequests", "closedIssues", "closedPullRequests"}
	for _, a := range aliases {
		if !strings.Contains(req.Query, a+":search(") {
			t.Errorf("query is missing the %s search alias:\n%s", a, req.Query)
		}
	}

	// Every search connection must carry a pagination boundary and be typed as
	// an issue search.
	searches := regexp.MustCompile(`search\([^)]*\)`).FindAllString(req.Query, -1)
	if len(searches) != len(aliases) {
		t.Fatalf("found %d search connections, want %d:\n%s", len(searches), len(aliases), req.Query)
	}
	for _, s := range searches {
		if !strings.Contains(s, "first:1") {
			t.Errorf("search connection %q has no pagination boundary", s)
		}
		if !strings.Contains(s, "type:ISSUE") {
			t.Errorf("search connection %q is not typed as an issue search", s)
		}
	}
}

// TestFetchIssueCountsQualifiers checks the state and kind qualifiers. Search
// answers a query with no is:issue or is:pull-request qualifier with a count of
// zero rather than an error, so losing one would zero the signal invisibly.
func TestFetchIssueCountsQualifiers(t *testing.T) {
	c, tr := newTestClient(allCountsResponse)
	if _, err := FetchIssueCounts(context.Background(), c, "kubernetes", "kubernetes", testSince()); err != nil {
		t.Fatalf("FetchIssueCounts() = %v, want nil error", err)
	}
	req := decodeRequest(t, tr.body)

	tests := []struct {
		variable    string
		wantAll     []string
		wantMissing []string
	}{
		{"updatedIssueQuery", []string{"is:issue"}, []string{"is:closed", "is:pull-request"}},
		{"updatedPullRequestQuery", []string{"is:pull-request"}, []string{"is:closed"}},
		{"closedIssueQuery", []string{"is:issue", "is:closed"}, []string{"is:pull-request"}},
		{"closedPullRequestQuery", []string{"is:pull-request", "is:closed"}, nil},
	}
	for _, test := range tests {
		q, ok := req.Variables[test.variable]
		if !ok {
			t.Errorf("request has no %s variable", test.variable)
			continue
		}
		for _, want := range test.wantAll {
			if !strings.Contains(q, want) {
				t.Errorf("%s = %q, want it to contain %q", test.variable, q, want)
			}
		}
		for _, unwanted := range test.wantMissing {
			if strings.Contains(q, unwanted) {
				t.Errorf("%s = %q, want it not to contain %q", test.variable, q, unwanted)
			}
		}
		if !strings.Contains(q, "repo:kubernetes/kubernetes") {
			t.Errorf("%s = %q, want it scoped to repo:kubernetes/kubernetes", test.variable, q)
		}
	}
}

// TestFetchIssueCountsCutoffIsPreciseAndShared checks that the window boundary
// is the exact timestamp the caller supplied and is identical across all four
// counts.
//
// The caller passes the cutoff rather than a lookback so this is deterministic
// and so the comment count it is divided by uses the same instant. Truncating
// the cutoff to a date would move the boundary by up to a day.
func TestFetchIssueCountsCutoffIsPreciseAndShared(t *testing.T) {
	c, tr := newTestClient(allCountsResponse)
	since := testSince()
	if _, err := FetchIssueCounts(context.Background(), c, "o", "n", since); err != nil {
		t.Fatalf("FetchIssueCounts() = %v, want nil error", err)
	}
	req := decodeRequest(t, tr.body)

	want := since.UTC().Format(time.RFC3339)
	cutoffRE := regexp.MustCompile(`updated:>=(\S+)`)
	for name, q := range req.Variables {
		m := cutoffRE.FindStringSubmatch(q)
		if m == nil {
			t.Errorf("%s = %q, want it to constrain the window with updated:>=", name, q)
			continue
		}
		if m[1] != want {
			t.Errorf("%s cutoff = %q, want exactly %q", name, m[1], want)
		}
	}
}

// TestFetchIssueCountsPropagatesErrors checks that an API failure surfaces
// instead of being reported as a repository with no issues.
func TestFetchIssueCountsPropagatesErrors(t *testing.T) {
	tests := []struct {
		name     string
		response string
		status   int
	}{
		{"graphql error", `{"errors":[{"message":"boom","type":"MISSING_PAGINATION_BOUNDARIES"}]}`, http.StatusOK},
		{"http error", `{"message":"unavailable"}`, http.StatusServiceUnavailable},
		{"malformed body", `not json`, http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tr := &captureTransport{response: test.response, status: test.status}
			c := githubapi.NewClient(&http.Client{Transport: tr})
			got, err := FetchIssueCounts(context.Background(), c, "o", "n", testSince())
			if err == nil {
				t.Fatalf("FetchIssueCounts() = %+v with nil error, want an error", got)
			}
			if got.Updated != 0 || got.Closed != 0 {
				t.Errorf("FetchIssueCounts() = %+v on failure, want zero values", got)
			}
		})
	}
}

// TestFetchIssueCountsReportsZeroForEmptyRepo makes sure a genuinely empty
// result is still distinguishable from a failure.
func TestFetchIssueCountsReportsZeroForEmptyRepo(t *testing.T) {
	c, _ := newTestClient(`{"data":{
		"updatedIssues":{"issueCount":0},
		"updatedPullRequests":{"issueCount":0},
		"closedIssues":{"issueCount":0},
		"closedPullRequests":{"issueCount":0}}}`)

	got, err := FetchIssueCounts(context.Background(), c, "o", "n", testSince())
	if err != nil {
		t.Fatalf("FetchIssueCounts() = %v, want nil error", err)
	}
	if got.Updated != 0 || got.Closed != 0 {
		t.Errorf("FetchIssueCounts() = %+v, want zero values", got)
	}
}

// TestFetchIssueCommentCountNoPageCount covers the case this change exists to
// prevent: a "next" link with no "last" link must not read as an empty result.
func TestFetchIssueCommentCountNoPageCount(t *testing.T) {
	tests := []struct {
		wantErr error
		name    string
		link    string
		want    int
	}{
		{
			name:    "cursor pagination has no last link",
			link:    `<https://api.github.com/repositories/1/issues/comments?page=2&after=Y3Vyc29y>; rel="next"`,
			wantErr: ErrorNoPageCount,
		},
		{
			name: "page count is available",
			link: `<https://api.github.com/repositories/1/issues/comments?page=2>; rel="next", ` +
				`<https://api.github.com/repositories/1/issues/comments?page=417>; rel="last"`,
			want: 417,
		},
		{
			name: "single page returns the number of items",
			link: "",
			want: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := githubapi.NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				h := http.Header{"Content-Type": []string{"application/json"}}
				if test.link != "" {
					h.Set("Link", test.link)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"id":1}]`)),
					Header:     h,
					Request:    r,
				}, nil
			})})

			got, err := FetchIssueCommentCount(context.Background(), c, "o", "n", testSince())
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("FetchIssueCommentCount() error = %v, want %v", err, test.wantErr)
				}
				if got != 0 {
					t.Errorf("FetchIssueCommentCount() = %d on failure, want 0", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("FetchIssueCommentCount() = %v, want nil error", err)
			}
			if got != test.want {
				t.Errorf("FetchIssueCommentCount() = %d, want %d", got, test.want)
			}
		})
	}
}
