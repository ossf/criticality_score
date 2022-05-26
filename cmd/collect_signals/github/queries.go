package github

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
)

const (
	legacyReleaseLookbackDays = 365
	legacyReleaseLookback     = time.Duration(legacyReleaseLookbackDays * 24 * time.Hour)
	legacyCommitLookback      = time.Duration(365 * 24 * time.Hour)
)

type basicRepoData struct {
	Name            string
	Owner           struct{ Login string }
	LicenseInfo     struct{ Name string }
	StargazerCount  int
	URL             string
	MirrorURL       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	PrimaryLanguage struct {
		Name string
	}
	Watchers struct {
		TotalCount int
	}
	HasIssuesEnabled bool
	IsArchived       bool
	IsDisabled       bool
	IsEmpty          bool
	IsMirror         bool

	DefaultBranchRef struct {
		Target struct {
			Commit struct { // this is the last commit
				AuthoredDate  time.Time
				RecentCommits struct {
					TotalCount int
				} `graphql:"recentcommits:history(since:$legacyCommitLookback)"`
			} `graphql:"... on Commit"`
		}
	}

	Tags struct {
		TotalCount int
	} `graphql:"refs(refPrefix:\"refs/tags/\")"`
}

func queryBasicRepoData(ctx context.Context, client *githubv4.Client, u *url.URL) (*basicRepoData, error) {
	// Search based on owner and repo name becaues the `repository` query
	// better handles changes in ownership and repository name than the
	// `resource` query.
	// TODO - consider improving support for scp style urls and urls ending in .git
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	owner := parts[0]
	name := parts[1]
	s := &struct {
		Repository basicRepoData `graphql:"repository(owner: $repositoryOwner, name: $repositoryName)"`
	}{}
	now := time.Now().UTC()
	vars := map[string]any{
		"repositoryOwner":      githubv4.String(owner),
		"repositoryName":       githubv4.String(name),
		"legacyCommitLookback": githubv4.GitTimestamp{Time: now.Add(-legacyCommitLookback)},
	}
	if err := client.Query(ctx, s, vars); err != nil {
		return nil, err
	}
	return &s.Repository, nil
}
