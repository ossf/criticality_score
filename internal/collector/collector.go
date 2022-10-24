// Copyright 2022 Criticality Score Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package collector is used to collect signals for a given repository from a
// variety of sources.
package collector

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"go.uber.org/zap"

	"github.com/ossf/criticality_score/internal/collector/depsdev"
	"github.com/ossf/criticality_score/internal/collector/github"
	"github.com/ossf/criticality_score/internal/collector/githubmentions"
	"github.com/ossf/criticality_score/internal/collector/projectrepo"
	"github.com/ossf/criticality_score/internal/collector/signal"
	"github.com/ossf/criticality_score/internal/githubapi"
)

// ErrUncollectableRepo is the base error returned when there is a problem with
// the repo url passed in to be collected.
//
// For example, the URL may point to an invalid repository host, or the URL
// may point to a repo that is inaccessible or missing.
var ErrUncollectableRepo = errors.New("repo failed")

// ErrRepoNotFound wraps ErrUncollectableRepo and is used when a repo cannot be
// found for collection.
var ErrRepoNotFound = fmt.Errorf("%w: not found", ErrUncollectableRepo)

// ErrUnsupportedURL wraps ErrUncollectableRepo and is used when a repo url
// does not match any of the supported hosts.
var ErrUnsupportedURL = fmt.Errorf("%w: unsupported url", ErrUncollectableRepo)

type Collector struct {
	config   *Config
	logger   *zap.Logger
	resolver *projectrepo.Resolver
	registry *registry
}

func New(ctx context.Context, logger *zap.Logger, opts ...Option) (*Collector, error) {
	c := &Collector{
		config:   makeConfig(ctx, logger, opts...),
		logger:   logger,
		resolver: &projectrepo.Resolver{},
		registry: newRegistry(),
	}

	ghClient := githubapi.NewClient(c.config.gitHubHTTPClient)

	// Register all the Repo factories.
	c.resolver.Register(github.NewRepoFactory(ghClient, logger))

	// Register all the sources that are supported and enabled.
	if c.config.IsEnabled(SourceTypeGithubRepo) {
		c.registry.Register(&github.RepoSource{})
	}
	if c.config.IsEnabled(SourceTypeGithubIssues) {
		c.registry.Register(&github.IssuesSource{})
	}
	if c.config.IsEnabled(SourceTypeGitHubMentions) {
		c.registry.Register(githubmentions.NewSource(ghClient))
	}
	if !c.config.IsEnabled(SourceTypeDepsDev) {
		// deps.dev collection source has been disabled, so skip it.
		logger.Warn("deps.dev signal source is disabled.")
	} else {
		ddsource, err := depsdev.NewSource(ctx, logger, c.config.gcpProject, c.config.gcpDatasetName)
		if err != nil {
			return nil, fmt.Errorf("init deps.dev source: %w", err)
		}
		logger.Info("deps.dev signal source enabled")
		c.registry.Register(ddsource)
	}

	return c, nil
}

// EmptySet returns all the empty instances of signal Sets that are used for
// determining the namespace and signals supported by the Source.
func (c *Collector) EmptySets() []signal.Set {
	return c.registry.EmptySets()
}

// Collect gathers and returns all the signals for the given project repo url.
func (c *Collector) Collect(ctx context.Context, u *url.URL) ([]signal.Set, error) {
	l := c.config.logger.With(zap.String("url", u.String()))

	repo, err := c.resolver.Resolve(ctx, u)
	if err != nil {
		switch {
		case errors.Is(err, projectrepo.ErrNoFactoryFound):
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedURL, u)
		case errors.Is(err, projectrepo.ErrNoRepoFound):
			return nil, fmt.Errorf("%w: %s", ErrRepoNotFound, u)
		default:
			return nil, fmt.Errorf("resolving project: %w", err)
		}
	}
	l = l.With(zap.String("canonical_url", repo.URL().String()))

	l.Info("Collecting")
	ss, err := c.registry.Collect(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("collecting project: %w", err)
	}
	return ss, nil
}
