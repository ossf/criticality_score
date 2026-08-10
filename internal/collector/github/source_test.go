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

package github

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"

	"github.com/ossf/criticality_score/v2/internal/collector/signal"
	"github.com/ossf/criticality_score/v2/internal/githubapi"
)

// recordingTransport answers every request from a canned body and records the
// paths that were requested, so a test can assert that a request was never
// made at all.
type recordingTransport struct {
	body  string
	paths []string
	mu    sync.Mutex
}

func (t *recordingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.paths = append(t.paths, r.URL.Path)
	t.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(t.body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Request:    r,
	}, nil
}

func (t *recordingTransport) requested() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.paths...)
}

// testRepo builds a repo with the basic data a source reads, without touching
// the network.
func testRepo(t *testing.T, tr http.RoundTripper, issuesEnabled bool) *repo {
	t.Helper()
	u, err := url.Parse("https://github.com/o/n")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	r := &repo{
		client:  githubapi.NewClient(&http.Client{Transport: tr}),
		origURL: u,
		realURL: u,
		logger:  zap.NewNop(),
	}
	r.BasicData = &basicRepoData{Name: "n", HasIssuesEnabled: issuesEnabled}
	r.BasicData.Owner.Login = "o"
	return r
}

func issuesSetOf(t *testing.T, s signal.Set) *signal.IssuesSet {
	t.Helper()
	is, ok := s.(*signal.IssuesSet)
	if !ok {
		t.Fatalf("Get() returned %T, want *signal.IssuesSet", s)
	}
	return is
}

// TestIssuesSourceSkipsDisabledTracker is the behaviour that keeps projects
// tracking bugs elsewhere from being recorded as having no bug reports. It
// also asserts no request is made, since there is nothing to ask for.
func TestIssuesSourceSkipsDisabledTracker(t *testing.T) {
	tr := &recordingTransport{body: `{}`}
	r := testRepo(t, tr, false)

	got, err := (&IssuesSource{}).Get(context.Background(), r, "")
	if err != nil {
		t.Fatalf("Get() = %v, want nil error", err)
	}
	is := issuesSetOf(t, got)

	for name, f := range map[string]bool{
		"updated_issues_count":    is.UpdatedCount.IsSet(),
		"closed_issues_count":     is.ClosedCount.IsSet(),
		"issue_comment_frequency": is.CommentFrequency.IsSet(),
	} {
		if f {
			t.Errorf("%s is set, want it left unknown when the tracker is disabled", name)
		}
	}
	if paths := tr.requested(); len(paths) != 0 {
		t.Errorf("made requests %v, want none when the tracker is disabled", paths)
	}
}

const countsBody = `{"data":{
	"updatedIssues":{"issueCount":10},
	"updatedPullRequests":{"issueCount":5},
	"closedIssues":{"issueCount":4},
	"closedPullRequests":{"issueCount":1}}}`

// TestIssuesSourceMapsCounts checks the totals land in the fields a scoring
// config reads, and that comment frequency is comments per updated item.
func TestIssuesSourceMapsCounts(t *testing.T) {
	tr := &roundTripSwitch{graphQL: countsBody, rest: `[{"id":1}]`, restLink: `<https://x?page=2>; rel="next", <https://x?page=30>; rel="last"`}
	r := testRepo(t, tr, true)

	got, err := (&IssuesSource{}).Get(context.Background(), r, "")
	if err != nil {
		t.Fatalf("Get() = %v, want nil error", err)
	}
	is := issuesSetOf(t, got)

	if want := 15; is.UpdatedCount.Get() != want {
		t.Errorf("updated = %d, want %d (issues plus pull requests)", is.UpdatedCount.Get(), want)
	}
	if want := 5; is.ClosedCount.Get() != want {
		t.Errorf("closed = %d, want %d", is.ClosedCount.Get(), want)
	}
	// 30 comments over 15 updated items.
	if want := 2.0; is.CommentFrequency.Get() != want {
		t.Errorf("comment frequency = %v, want %v", is.CommentFrequency.Get(), want)
	}
}

// TestIssuesSourceLeavesCommentFrequencyUnknown covers the two ways the
// comment total can be unavailable. Neither may record a zero, which would
// read as "nobody comments here", and neither may fail the repository, since
// the issue counts themselves are fine.
func TestIssuesSourceLeavesCommentFrequencyUnknown(t *testing.T) {
	tests := map[string]*roundTripSwitch{
		"no updated issues": {
			graphQL: `{"data":{"updatedIssues":{"issueCount":0},"updatedPullRequests":{"issueCount":0},
				"closedIssues":{"issueCount":0},"closedPullRequests":{"issueCount":0}}}`,
			rest: `[]`,
		},
		"comment endpoint has no page count": {
			graphQL:  countsBody,
			rest:     `[{"id":1}]`,
			restLink: `<https://x?page=2&after=cursor>; rel="next"`,
		},
	}

	for name, tr := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := (&IssuesSource{}).Get(context.Background(), testRepo(t, tr, true), "")
			if err != nil {
				t.Fatalf("Get() = %v, want nil error", err)
			}
			if is := issuesSetOf(t, got); is.CommentFrequency.IsSet() {
				t.Errorf("comment frequency = %v, want it left unknown", is.CommentFrequency.Get())
			}
		})
	}
}

// roundTripSwitch answers GraphQL and REST requests with different bodies.
type roundTripSwitch struct {
	graphQL  string
	rest     string
	restLink string
}

func (t *roundTripSwitch) RoundTrip(r *http.Request) (*http.Response, error) {
	body, h := t.rest, http.Header{"Content-Type": []string{"application/json"}}
	if strings.Contains(r.URL.Path, "graphql") {
		body = t.graphQL
	} else if t.restLink != "" {
		h.Set("Link", t.restLink)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     h,
		Request:    r,
	}, nil
}
