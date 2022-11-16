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
	"errors"
	"fmt"
	"net/url"

	"go.uber.org/zap"

	"github.com/ossf/criticality_score/internal/collector/projectrepo"
	"github.com/ossf/criticality_score/internal/githubapi"
)

type factory struct {
	client *githubapi.Client
	logger *zap.Logger
}

func NewRepoFactory(client *githubapi.Client, logger *zap.Logger) projectrepo.Factory {
	return &factory{
		client: client,
		logger: logger,
	}
}

func (f *factory) New(ctx context.Context, u *url.URL) (projectrepo.Repo, error) {
	r := &repo{
		client:  f.client,
		origURL: u,
		logger:  f.logger.With(zap.String("url", u.String())),
	}
	if err := r.init(ctx); err != nil {
		if errors.Is(err, githubapi.ErrGraphQLNotFound) {
			return nil, fmt.Errorf("%w: %s", projectrepo.ErrNoRepoFound, u)
		} else {
			return nil, err
		}
	}
	return r, nil
}

func (f *factory) Match(u *url.URL) bool {
	return u.Hostname() == "github.com"
}
