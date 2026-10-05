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
	"strings"

	"github.com/google/go-github/v47/github"
	"github.com/hasura/go-graphql-client"

	"github.com/ossf/criticality_score/v2/internal/githubapi"
)

// FetchTotalContributors returns the total number of contributors for the given repository.
//
// Results will be capped to MaxContributorLimit.
func FetchTotalContributors(ctx context.Context, c *githubapi.Client, owner, name string) (int, error) {
	opts := &github.ListContributorsOptions{
		Anon:        "1",
		ListOptions: github.ListOptions{PerPage: 1}, // 1 result per page means LastPage is total number of records.
	}
	cs, resp, err := c.Rest().Repositories.ListContributors(ctx, owner, name, opts)
	if errorTooManyContributors(err) {
		return MaxContributorLimit, nil
	}
	if err != nil {
		return 0, err
	}
	if resp.NextPage == 0 {
		return len(cs), nil
	}
	total := resp.LastPage
	if total > MaxContributorLimit {
		return MaxContributorLimit, nil
	}
	return total, nil
}

// FetchOrgCount returns the number of unique orgs/companies for the top
// MaxTopContributors of a given repository.
//
// If there are too many contributors for the given repo, the number returned
// will be TooManyContributorsOrgCount.
func FetchOrgCount(ctx context.Context, c *githubapi.Client, owner, name string) (int, error) {
	orgFilter := strings.NewReplacer(
		"inc.", "",
		"llc", "",
		"@", "",
		" ", "",
	)

	opts := &github.ListContributorsOptions{
		ListOptions: github.ListOptions{
			PerPage: MaxTopContributors,
		},
	}
	// Get the list of contributors
	cs, _, err := c.Rest().Repositories.ListContributors(ctx, owner, name, opts)
	if errorTooManyContributors(err) {
		return TooManyContributorsOrgCount, nil
	}
	if err != nil {
		return 0, err
	}

	// Doing this over REST would take O(n) requests, using GraphQL takes O(1).
	var userIds []graphql.ID
	for _, contributor := range cs {
		login := contributor.GetLogin()
		if login == "" {
			continue
		}
		accountType := contributor.GetType()
		if accountType == "Bot" {
			continue
		}
		userIds = append(userIds, graphql.ID(contributor.GetNodeID()))
	}
	if len(userIds) == 0 {
		// We didn't add any users.
		return 0, err
	}
	var r struct {
		Nodes []struct {
			User struct{ Company string } `graphql:"... on User"`
		} `graphql:"nodes(ids: $ids)"`
	}
	err = c.GraphQL().Query(ctx, &r, map[string]any{
		"ids": userIds,
	})
	if err != nil {
		return 0, err
	}
	// Extract the Company from each returned field and add it to the org set.
	orgSet := make(map[string]empty)
	for _, u := range r.Nodes {
		org := u.User.Company
		if org == "" {
			continue
		}
		org = strings.TrimRight(orgFilter.Replace(strings.ToLower(org)), ",")
		orgSet[org] = empty{}
	}
	return len(orgSet), nil
}

// errorTooManyContributors returns true if err is a 403 due to too many
// contributors.
func errorTooManyContributors(err error) bool {
	if err == nil {
		return false
	}
	var e *github.ErrorResponse
	ok := errors.As(err, &e)
	if !ok {
		return false
	}
	return e.Response.StatusCode == 403 && strings.Contains(e.Message, "list is too large")
}
