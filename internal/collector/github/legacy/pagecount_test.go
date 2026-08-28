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
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ossf/criticality_score/v2/internal/githubapi"
)

// linkClient returns a client whose every response carries the given Link
// header and body.
func linkClient(link, body string) *githubapi.Client {
	return githubapi.NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		h := http.Header{"Content-Type": []string{"application/json"}}
		if link != "" {
			h.Set("Link", link)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     h,
			Request:    r,
		}, nil
	})})
}

const (
	nextOnly = `<https://api.github.com/x?page=2&after=Y3Vyc29y>; rel="next"`
	nextLast = `<https://api.github.com/x?page=2>; rel="next", <https://api.github.com/x?page=93>; rel="last"`
)

// TestFetchTotalContributorsNoPageCount covers the branch where a "next" link
// arrives without a "last" link.
//
// Contributor count is deliberately treated as fatal, unlike comment
// frequency, which is left unset. It carries the heaviest weight of any signal
// in the default configuration, and a repository silently credited with zero
// contributors is worse than one that fails loudly and can be retried.
func TestFetchTotalContributorsNoPageCount(t *testing.T) {
	tests := []struct {
		wantErr error
		name    string
		link    string
		body    string
		want    int
	}{
		{
			name:    "cursor pagination has no last link",
			link:    nextOnly,
			body:    `[{"login":"a"}]`,
			wantErr: ErrorNoPageCount,
		},
		{
			name: "page count is available",
			link: nextLast,
			body: `[{"login":"a"}]`,
			want: 93,
		},
		{
			name: "single page returns the number of items",
			body: `[{"login":"a"},{"login":"b"}]`,
			want: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := FetchTotalContributors(context.Background(), linkClient(test.link, test.body), "o", "n")
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("FetchTotalContributors() error = %v, want %v", err, test.wantErr)
				}
				if got != 0 {
					t.Errorf("FetchTotalContributors() = %d on failure, want 0", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("FetchTotalContributors() = %v, want nil error", err)
			}
			if got != test.want {
				t.Errorf("FetchTotalContributors() = %d, want %d", got, test.want)
			}
		})
	}
}

// TestFetchCreatedTimeNoPageCount covers the same branch for the first commit
// lookup, which reaches it on every repository whose history predates its
// GitHub repository, because supplying `until` suppresses the "last" link.
//
// Here the fallback is earliestSoFar rather than an error. There is no way to
// reach the oldest commit without a page count, but the caller already has a
// usable upper bound, and this function is documented to return it when no
// earlier time can be established.
func TestFetchCreatedTimeNoPageCount(t *testing.T) {
	earliest := time.Date(2014, time.August, 19, 0, 0, 0, 0, time.UTC)
	oldest := `[{"commit":{"committer":{"date":"1991-09-17T00:00:00Z"}}}]`

	tests := []struct {
		want time.Time
		name string
		link string
		body string
	}{
		{
			// Without the guard the function would return the commit from the
			// first page, which is the newest at or before earliestSoFar, and
			// date the project to when it appeared on GitHub.
			name: "cursor pagination falls back to the known bound",
			link: nextOnly,
			body: `[{"commit":{"committer":{"date":"2014-08-18T00:00:00Z"}}}]`,
			want: earliest,
		},
		{
			name: "page count is available",
			link: nextLast,
			body: oldest,
			want: time.Date(1991, time.September, 17, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "no earlier commits returns the known bound",
			body: `[]`,
			want: earliest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := FetchCreatedTime(context.Background(), linkClient(test.link, test.body), "o", "n", earliest)
			if err != nil {
				t.Fatalf("FetchCreatedTime() = %v, want nil error", err)
			}
			if !got.Equal(test.want) {
				t.Errorf("FetchCreatedTime() = %v, want %v", got, test.want)
			}
		})
	}
}
