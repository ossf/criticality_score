// Copyright 2026 Criticality Score Authors
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

package legacy_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ossf/criticality_score/v2/internal/collector/github/legacy"
	"github.com/ossf/criticality_score/v2/internal/githubapi"
)

// newTestClient returns a githubapi.Client pointed at the given test server.
func newTestClient(t *testing.T, srv *httptest.Server) *githubapi.Client {
	t.Helper()
	c := githubapi.NewClient(srv.Client())
	u, err := url.Parse(srv.URL + "/")
	if err != nil {
		t.Fatalf("failed to parse test server url: %v", err)
	}
	c.Rest().BaseURL = u
	return c
}

// writeIssues writes n placeholder issues as the response body.
func writeIssues(w http.ResponseWriter, n int) {
	var sb strings.Builder
	sb.WriteString("[")
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `{"number":%d}`, i+1)
	}
	sb.WriteString("]")
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(sb.String()))
}

func TestFetchIssueCount(t *testing.T) {
	//nolint:govet // field alignment is irrelevant for a test table.
	tests := []struct {
		name string
		// handler serves the issues endpoint.
		handler func(w http.ResponseWriter, r *http.Request)
		want    int
	}{
		{
			// GitHub migrated this endpoint to cursor-based pagination: the
			// Link header carries only a rel="next" with an opaque cursor, so
			// LastPage is 0 and the count has to come from paging.
			name: "cursor pagination without rel=last",
			handler: func(w http.ResponseWriter, r *http.Request) {
				perPage := r.URL.Query().Get("per_page")
				page := r.URL.Query().Get("page")
				if perPage == "1" {
					// The initial probe request.
					w.Header().Set("Link",
						`<https://api.github.com/repositories/1/issues?per_page=1&after=Y3Vyc29yOnYyOpK5&page=2>; rel="next"`)
					writeIssues(w, 1)
					return
				}
				switch page {
				case "", "1":
					w.Header().Set("Link",
						`<https://api.github.com/repositories/1/issues?per_page=100&after=Y3Vyc29yOnYyOpK5&page=2>; rel="next"`)
					writeIssues(w, 100)
				default:
					// Final short page, no Link header.
					writeIssues(w, 50)
				}
			},
			want: 150,
		},
		{
			// Older behaviour: rel="last" is present, so the total can be read
			// straight off the Link header without paging.
			name: "rel=last is used when present",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("per_page") != "1" {
					t.Errorf("unexpected paging request: %s", r.URL.String())
				}
				w.Header().Set("Link",
					`<https://api.github.com/repositories/1/issues?per_page=1&page=2>; rel="next", `+
						`<https://api.github.com/repositories/1/issues?per_page=1&page=42>; rel="last"`)
				writeIssues(w, 1)
			},
			want: 42,
		},
		{
			name: "single result",
			handler: func(w http.ResponseWriter, r *http.Request) {
				writeIssues(w, 1)
			},
			want: 1,
		},
		{
			name: "no results",
			handler: func(w http.ResponseWriter, r *http.Request) {
				writeIssues(w, 0)
			},
			want: 0,
		},
		{
			// The API returns 5xx when there are too many issues to count.
			name: "too many issues",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			want: legacy.MaxIssuesLimit,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(test.handler))
			defer srv.Close()

			got, err := legacy.FetchIssueCount(context.Background(), newTestClient(t, srv),
				"owner", "name", legacy.IssueStateAll, legacy.IssueLookback)
			if err != nil {
				t.Fatalf("FetchIssueCount() returned err: %v", err)
			}
			if got != test.want {
				t.Errorf("FetchIssueCount() = %d, want %d", got, test.want)
			}
		})
	}
}

// TestFetchIssueCountPagingIsCapped ensures paging stops once the limit is hit,
// rather than walking an unbounded number of pages.
func TestFetchIssueCountPagingIsCapped(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Query().Get("per_page") == "1" {
			w.Header().Set("Link",
				`<https://api.github.com/repositories/1/issues?per_page=1&after=cursor&page=2>; rel="next"`)
			writeIssues(w, 1)
			return
		}
		page := r.URL.Query().Get("page")
		if page == "" {
			page = "1"
		}
		n, _ := strconv.Atoi(page)
		w.Header().Set("Link", fmt.Sprintf(
			`<https://api.github.com/repositories/1/issues?per_page=100&after=cursor&page=%d>; rel="next"`, n+1))
		writeIssues(w, 100)
	}))
	defer srv.Close()

	got, err := legacy.FetchIssueCount(context.Background(), newTestClient(t, srv),
		"owner", "name", legacy.IssueStateAll, legacy.IssueLookback)
	if err != nil {
		t.Fatalf("FetchIssueCount() returned err: %v", err)
	}
	if got != legacy.MaxIssuesLimit {
		t.Errorf("FetchIssueCount() = %d, want %d", got, legacy.MaxIssuesLimit)
	}
	// 1 probe request, plus MaxIssuesLimit/100 paging requests.
	if want := 1 + legacy.MaxIssuesLimit/100; requests != want {
		t.Errorf("made %d requests, want %d", requests, want)
	}
}

// TestFetchIssueCountLookback checks the since parameter is derived from the
// lookback duration.
func TestFetchIssueCountLookback(t *testing.T) {
	var since string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		since = r.URL.Query().Get("since")
		writeIssues(w, 0)
	}))
	defer srv.Close()

	lookback := 90 * 24 * time.Hour
	before := time.Now().UTC().Add(-lookback)
	if _, err := legacy.FetchIssueCount(context.Background(), newTestClient(t, srv),
		"owner", "name", legacy.IssueStateAll, lookback); err != nil {
		t.Fatalf("FetchIssueCount() returned err: %v", err)
	}
	after := time.Now().UTC().Add(-lookback)

	got, err := time.Parse(time.RFC3339, since)
	if err != nil {
		t.Fatalf("failed to parse since %q: %v", since, err)
	}
	if got.Before(before.Truncate(time.Second)) || got.After(after.Add(time.Second)) {
		t.Errorf("since = %v, want between %v and %v", got, before, after)
	}
}
