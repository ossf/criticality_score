package githubapi

import (
	"net/http"

	"github.com/google/go-github/v44/github"
	"github.com/shurcooL/githubv4"
)

type Client struct {
	restClient  *github.Client
	graphClient *githubv4.Client
}

func NewClient(client *http.Client) *Client {
	c := &Client{
		restClient:  github.NewClient(client),
		graphClient: githubv4.NewClient(client),
	}

	return c
}

func (c *Client) Rest() *github.Client {
	return c.restClient
}

func (c *Client) GraphQL() *githubv4.Client {
	return c.graphClient
}
