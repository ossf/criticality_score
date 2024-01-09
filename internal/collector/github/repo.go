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
	"net/url"
	"time"

	"go.uber.org/zap"

	"github.com/ossf/criticality_score/internal/collector/github/legacy"
	"github.com/ossf/criticality_score/internal/githubapi"
)

// repo implements the projectrepo.Repo interface for a GitHub repository.
type repo struct {
	client  *githubapi.Client
	origURL *url.URL
	logger  *zap.Logger

	BasicData *BasicRepoData
	realURL   *url.URL
	created   time.Time
}

// URL implements the projectrepo.Repo interface.
func (r *repo) URL() *url.URL {
	return r.realURL
}

func (r *repo) init(ctx context.Context, q Query) error {
	r.logger.Debug("Fetching basic data from GitHub")
	data, err := q.QueryBasicRepoData(ctx, r.client.GraphQL(), r.origURL)
	if err != nil {
		return err
	}
	r.logger.Debug("Fetching created time")
	if created, err := legacy.FetchCreatedTime(ctx, r.client, data.Owner.Login, data.Name, data.CreatedAt); err != nil {
		return err
	} else {
		r.created = created
	}
	r.realURL, err = url.Parse(data.URL)
	if err != nil {
		return err
	}
	// Set BasicData last as it is used to indicate init() has been called.
	r.BasicData = data
	return nil
}

func (r *repo) owner() string {
	return r.BasicData.Owner.Login
}

func (r *repo) name() string {
	return r.BasicData.Name
}

func (r *repo) updatedAt() time.Time {
	return r.BasicData.DefaultBranchRef.Target.Commit.AuthoredDate
}

func (r *repo) createdAt() time.Time {
	return r.created
}
