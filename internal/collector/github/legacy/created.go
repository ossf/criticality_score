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
	"fmt"
	"time"

	"github.com/google/go-github/v47/github"

	"github.com/ossf/criticality_score/v2/internal/githubapi"
)

// FetchCreatedTime returns the earliest known creation time for a given
// repository based on the commit history, before or equal to earliestSoFar.
//
// Only times before earliestSoFar will be considered. If there is no time before
// earliestSoFar found, the value of earliestSoFar will be returned.
func FetchCreatedTime(ctx context.Context, c *githubapi.Client, owner, name string, earliestSoFar time.Time) (time.Time, error) {
	opts := &github.CommitsListOptions{
		Until:       earliestSoFar,
		ListOptions: github.ListOptions{PerPage: 1}, // 1 result per page means LastPage is total number of records.
	}
	cs, resp, err := c.Rest().Repositories.ListCommits(ctx, owner, name, opts)
	if githubapi.ErrorResponseStatusCode(err) == 409 {
		// 409 Conflict can happen if the Git Repository is empty.
		return earliestSoFar, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	// Handle 0 or 1 result.
	if resp.NextPage == 0 || resp.LastPage == 1 {
		if len(cs) == 0 {
			return earliestSoFar, nil
		} else {
			return cs[0].GetCommit().GetCommitter().GetDate(), nil
		}
	}
	if resp.LastPage == 0 {
		// GitHub does not always send a "last" link, and observed cases
		// include requests that constrain the range with `until`. Without a
		// page count there is no cheap way to jump to the oldest commit;
		// walking the cursor to the end of a large history would cost
		// thousands of requests per repository.
		//
		// Without this branch the loop below does not run at all, because
		// opts.Page and resp.LastPage are both zero, and the function falls
		// through to return the single commit from the request above. That is
		// the newest commit at or before earliestSoFar, not the oldest, so the
		// repository is dated to roughly when it appeared on GitHub rather
		// than when the project began.
		//
		// Returning earliestSoFar is this function's documented answer when no
		// earlier time can be established, and says so explicitly.
		return earliestSoFar, nil
	}
	// It is possible that new commits are pushed between the previous
	// request and the next. If we detect that we are not on LastPage
	// try again a few more times.
	attempts := 5
	for opts.Page != resp.LastPage && attempts > 0 {
		opts.Page = resp.LastPage
		cs, resp, err = c.Rest().Repositories.ListCommits(ctx, owner, name, opts)
		if err != nil {
			return time.Time{}, err
		}
		attempts--
	}
	if len(cs) != 0 {
		return cs[len(cs)-1].GetCommit().GetCommitter().GetDate(), nil
	} else {
		return time.Time{}, fmt.Errorf("commits disappeared for GitHub repo '%s/%s'", owner, name)
	}
}
