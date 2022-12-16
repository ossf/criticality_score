package github

import (
	"context"
	"net/url"

	"github.com/shurcooL/githubv4"
)

type Query interface {
	QueryBasicRepoData(ctx context.Context, client *githubv4.Client, u *url.URL) (*BasicRepoData, error)
}
