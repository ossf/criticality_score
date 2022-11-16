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

package collector

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-logr/zapr"
	"github.com/ossf/scorecard/v4/clients/githubrepo/roundtripper"
	sclog "github.com/ossf/scorecard/v4/log"
	"go.uber.org/zap"

	"github.com/ossf/criticality_score/internal/githubapi"
)

// DefaultGCPDatasetName is the default name to use for GCP BigQuery Datasets.
const DefaultGCPDatasetName = "criticality_score_data"

// SourceType is used to identify the various sources signals can be collected
// from.
type SourceType int

const (
	SourceTypeGithubRepo SourceType = iota
	SourceTypeGithubIssues
	SourceTypeGitHubMentions
	SourceTypeDepsDev
)

// String implements the fmt.Stringer interface.
func (t SourceType) String() string {
	switch t {
	case SourceTypeGithubRepo:
		return "SourceTypeGithubRepo"
	case SourceTypeGithubIssues:
		return "SourceTypeGithubIssues"
	case SourceTypeGitHubMentions:
		return "SourceTypeGitHubMentions"
	case SourceTypeDepsDev:
		return "SourceTypeDepsDev"
	default:
		return fmt.Sprintf("Unknown SourceType %d", int(t))
	}
}

type sourceStatus int

const (
	sourceStatusDisabled sourceStatus = iota
	sourceStatusEnabled
)

//nolint:govet
type config struct {
	logger *zap.Logger

	gitHubHTTPClient *http.Client

	gcpProject     string
	gcpDatasetName string

	sourceStatuses      map[SourceType]sourceStatus
	defaultSourceStatus sourceStatus
}

// Option is an interface used to change the config.
type Option interface{ set(*config) }

// option implements Option interface.
type option func(*config)

// set implements the Option interface.
func (o option) set(c *config) { o(c) }

func makeConfig(ctx context.Context, logger *zap.Logger, opts ...Option) *config {
	c := &config{
		logger:              logger,
		defaultSourceStatus: sourceStatusEnabled,
		sourceStatuses:      make(map[SourceType]sourceStatus),
		gitHubHTTPClient:    defaultGitHubHTTPClient(ctx, logger),
		gcpProject:          "",
		gcpDatasetName:      DefaultGCPDatasetName,
	}

	for _, opt := range opts {
		opt.set(c)
	}

	return c
}

// IsEnabled returns true if the given SourceType is enabled.
func (c *config) IsEnabled(s SourceType) bool {
	if status, ok := c.sourceStatuses[s]; ok {
		return status == sourceStatusEnabled
	} else {
		return c.defaultSourceStatus == sourceStatusEnabled
	}
}

func defaultGitHubHTTPClient(ctx context.Context, logger *zap.Logger) *http.Client {
	// roundtripper requires us to use the scorecard logger.
	innerLogger := zapr.NewLogger(logger)
	scLogger := &sclog.Logger{Logger: &innerLogger}

	// Prepare a client for communicating with GitHub's GraphQLv4 API and Restv3 API
	rt := githubapi.NewRetryRoundTripper(roundtripper.NewTransport(ctx, scLogger), logger)

	return &http.Client{
		Transport: rt,
	}
}

// EnableAllSources enables all SourceTypes for collection.
//
// All data sources will be used for collection unless explicitly disabled
// with DisableSource.
func EnableAllSources() Option {
	return option(func(c *config) {
		c.defaultSourceStatus = sourceStatusEnabled
	})
}

// DisableAllSources will disable all SourceTypes for collection.
//
// No data sources will be used for collection unless explicitly enabled
// with EnableSource.
func DisableAllSources() Option {
	return option(func(c *config) {
		c.defaultSourceStatus = sourceStatusDisabled
	})
}

// EnableSource will enable the supplied SourceType for collection.
func EnableSource(s SourceType) Option {
	return option(func(c *config) {
		c.sourceStatuses[s] = sourceStatusEnabled
	})
}

// DisableSource will enable the supplied SourceType for collection.
func DisableSource(s SourceType) Option {
	return option(func(c *config) {
		c.sourceStatuses[s] = sourceStatusDisabled
	})
}

// GCPProject is used to set the ID of the GCP project used for sources that
// depend on GCP.
//
// If not supplied, the currently configured project will be used.
func GCPProject(n string) Option {
	return option(func(c *config) {
		c.gcpProject = n
	})
}

// GCPDatasetName overrides DefaultGCPDatasetName with the supplied dataset name.
func GCPDatasetName(n string) Option {
	return option(func(c *config) {
		c.gcpDatasetName = n
	})
}
