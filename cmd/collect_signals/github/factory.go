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

	log "github.com/sirupsen/logrus"

	"github.com/ossf/criticality_score/cmd/collect_signals/projectrepo"
	"github.com/ossf/criticality_score/internal/githubapi"
)

type factory struct {
	client *githubapi.Client
	logger *log.Logger
}

func NewRepoFactory(client *githubapi.Client, logger *log.Logger) projectrepo.Factory {
	return &factory{
		client: client,
		logger: logger,
	}
}

func (f *factory) New(ctx context.Context, u *url.URL) (projectrepo.Repo, error) {
	p := &repo{
		client:  f.client,
		origURL: u,
		logger:  f.logger.WithField("url", u),
	}
	if err := p.init(ctx); err != nil {
		return nil, err
	}
	return p, nil
}

func (f *factory) Match(u *url.URL) bool {
	return u.Hostname() == "github.com"
}
