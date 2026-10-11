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

package legacy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-github/v47/github"

	"github.com/ossf/criticality_score/v2/internal/githubapi"
)

const (
	testOwner = "owner"
	testRepo  = "repo"
)

// fakeList serves a GitHub-style list endpoint with total items.
//
// With cursor set it paginates like GitHub's issues endpoint does today: the
// Link header has only a rel="next" link with an "after" cursor (and a page
// number), and no rel="last" link. Otherwise it uses classic page-number
// links including rel="last".
type fakeList struct {
	queries  []url.Values
	total    int
	status   int // When non-zero, every request fails with this status.
	requests int
	cursor   bool
}

func (f *fakeList) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.requests++
	q := r.URL.Query()
	f.queries = append(f.queries, q)
	if f.status != 0 {
		w.WriteHeader(f.status)
		_, _ = w.Write([]byte(`{"message":"server error"}`))
		return
	}

	perPage, _ := strconv.Atoi(q.Get("per_page"))
	if perPage <= 0 {
		perPage = 30
	}
	page, _ := strconv.Atoi(q.Get("page"))
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * perPage
	if f.cursor && q.Get("after") != "" {
		offset, _ = strconv.Atoi(q.Get("after"))
	}
	end := min(offset+perPage, f.total)

	items := []map[string]int{}
	for i := offset; i < end; i++ {
		items = append(items, map[string]int{"id": i + 1})
	}

	link := func(rel string, set map[string]string) string {
		u := url.URL{Scheme: "http", Host: r.Host, Path: r.URL.Path}
		nq := r.URL.Query()
		for k, v := range set {
			nq.Set(k, v)
		}
		u.RawQuery = nq.Encode()
		return fmt.Sprintf(`<%s>; rel="%s"`, u.String(), rel)
	}
	if end < f.total {
		if f.cursor {
			w.Header().Set("Link", link("next", map[string]string{"after": strconv.Itoa(end), "page": strconv.Itoa(page + 1)}))
		} else {
			last := (f.total + perPage - 1) / perPage
			w.Header().Set("Link", link("next", map[string]string{"page": strconv.Itoa(page + 1)})+", "+
				link("last", map[string]string{"page": strconv.Itoa(last)}))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

func newTestClient(t *testing.T, issues, comments http.Handler) *githubapi.Client {
	t.Helper()
	mux := http.NewServeMux()
	if issues != nil {
		mux.Handle("/repos/"+testOwner+"/"+testRepo+"/issues", issues)
	}
	if comments != nil {
		mux.Handle("/repos/"+testOwner+"/"+testRepo+"/issues/comments", comments)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := githubapi.NewClient(srv.Client())
	base, err := url.Parse(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	c.Rest().BaseURL = base
	return c
}

func TestFetchIssueCount(t *testing.T) {
	tests := []struct {
		name         string
		list         fakeList
		want         int
		wantRequests int
	}{
		{name: "no issues", list: fakeList{total: 0}, want: 0, wantRequests: 1},
		{name: "single issue", list: fakeList{total: 1}, want: 1, wantRequests: 1},
		{name: "last link present", list: fakeList{total: 9}, want: 9, wantRequests: 1},
		{name: "cursor pagination", list: fakeList{total: 9, cursor: true}, want: 9, wantRequests: 2},
		{name: "cursor pagination, several pages", list: fakeList{total: 250, cursor: true}, want: 250, wantRequests: 4},
		{name: "cursor pagination, capped", list: fakeList{total: MaxIssuesLimit + 1000, cursor: true}, want: MaxIssuesLimit, wantRequests: 1 + MaxIssuesLimit/countPageSize},
		{name: "server error", list: fakeList{status: http.StatusBadGateway}, want: MaxIssuesLimit, wantRequests: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list := tt.list
			c := newTestClient(t, &list, nil)
			got, err := FetchIssueCount(context.Background(), c, testOwner, testRepo, IssueStateAll, IssueLookback)
			if err != nil {
				t.Fatalf("FetchIssueCount() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("FetchIssueCount() = %d, want %d", got, tt.want)
			}
			if list.requests != tt.wantRequests {
				t.Errorf("requests = %d, want %d", list.requests, tt.wantRequests)
			}
		})
	}
}

func TestFetchIssueCountPagingKeepsFilters(t *testing.T) {
	list := fakeList{total: 3, cursor: true}
	c := newTestClient(t, &list, nil)
	if _, err := FetchIssueCount(context.Background(), c, testOwner, testRepo, IssueStateClosed, IssueLookback); err != nil {
		t.Fatal(err)
	}
	for i, q := range list.queries {
		if got := q.Get("state"); got != IssueStateClosed {
			t.Errorf("request %d: state = %q, want %q", i, got, IssueStateClosed)
		}
		since, err := time.Parse(time.RFC3339, q.Get("since"))
		if err != nil {
			t.Fatalf("request %d: since = %q: %v", i, q.Get("since"), err)
		}
		if d := time.Since(since); d < IssueLookback-time.Minute || d > IssueLookback+time.Minute {
			t.Errorf("request %d: since is %v ago, want about %v", i, d, IssueLookback)
		}
	}
}

func TestFetchIssueCommentCount(t *testing.T) {
	tests := []struct {
		wantErr error
		name    string
		list    fakeList
		want    int
	}{
		{name: "no comments", list: fakeList{total: 0}, want: 0},
		{name: "last link present", list: fakeList{total: 37}, want: 37},
		{name: "cursor pagination", list: fakeList{total: 37, cursor: true}, want: 37},
		{name: "cursor pagination, too many", list: fakeList{total: MaxIssuesLimit + 1, cursor: true}, wantErr: ErrorTooManyResults},
		{name: "server error", list: fakeList{status: http.StatusBadGateway}, wantErr: ErrorTooManyResults},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list := tt.list
			c := newTestClient(t, nil, &list)
			got, err := FetchIssueCommentCount(context.Background(), c, testOwner, testRepo, IssueLookback)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("FetchIssueCommentCount() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("FetchIssueCommentCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNextLink(t *testing.T) {
	// A real header from GitHub's issues endpoint (October 2026): cursor
	// pagination, with no rel="last" link.
	const realHeader = `<https://api.github.com/repositories/1414640827/issues?state=all&per_page=1&since=2026-07-01T00%3A00%3A00Z&after=Y3Vyc29yOnYyOpLPAAABoSxDOzDPAAAAAVnx7Tg%3D&page=2>; rel="next"`
	tests := []struct {
		header string
		want   string
	}{
		{header: realHeader, want: "https://api.github.com/repositories/1414640827/issues?state=all&per_page=1&since=2026-07-01T00%3A00%3A00Z&after=Y3Vyc29yOnYyOpLPAAABoSxDOzDPAAAAAVnx7Tg%3D&page=2"},
		{header: `<https://x/a?page=2>; rel="next", <https://x/a?page=5>; rel="last"`, want: "https://x/a?page=2"},
		{header: `<https://x/a?page=1>; rel="prev", <https://x/a?page=1>; rel="first"`, want: ""},
		{header: "", want: ""},
		{header: "garbage", want: ""},
	}
	for _, tt := range tests {
		resp := &github.Response{Response: &http.Response{Header: http.Header{}}}
		if tt.header != "" {
			resp.Header.Set("Link", tt.header)
		}
		if got := nextLink(resp); got != tt.want {
			t.Errorf("nextLink(%q) = %q, want %q", tt.header, got, tt.want)
		}
	}
	if got := nextLink(nil); got != "" {
		t.Errorf("nextLink(nil) = %q, want empty", got)
	}
}
