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

package githubapi

import (
	"net/http"

	"github.com/google/go-github/v47/github"
	"github.com/hasura/go-graphql-client"
)

// Client provides simple access to GitHub's REST and GraphQL APIs.
type Client struct {
	restClient  *github.Client
	graphClient *graphql.Client
}

// NewClient creates a new instances of Client.
func NewClient(client *http.Client) *Client {
	// Wrap the Transport for the GraphQL client to produce more useful errors.
	graphClient := *client // deref to copy the struct
	graphClient.Transport = &graphQLRoundTripper{inner: client.Transport}

	return &Client{
		restClient:  github.NewClient(client),
		graphClient: graphql.NewClient(DefaultGraphQLEndpoint, &graphClient),
	}
}

// Rest returns a client for communicating with GitHub's REST API.
func (c *Client) Rest() *github.Client {
	return c.restClient
}

// GraphQL returns a client for communicating with GitHub's GraphQL API.
func (c *Client) GraphQL() *graphql.Client {
	return c.graphClient
}
