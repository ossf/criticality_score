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
	"time"

	"go.uber.org/zap"

	"github.com/ossf/criticality_score/v2/internal/collector/github/legacy"
	"github.com/ossf/criticality_score/v2/internal/collector/projectrepo"
	"github.com/ossf/criticality_score/v2/internal/collector/signal"
)

type RepoSource struct{}

func (rc *RepoSource) EmptySet() signal.Set {
	return &signal.RepoSet{}
}

func (rc *RepoSource) Get(ctx context.Context, r projectrepo.Repo, _ string) (signal.Set, error) {
	ghr, ok := r.(*repo)
	if !ok {
		return nil, errors.New("project is not a github project")
	}
	now := time.Now()

	s := &signal.RepoSet{
		URL:          signal.Val(r.URL().String()),
		Language:     signal.Val(ghr.BasicData.PrimaryLanguage.Name),
		License:      signal.Val(ghr.BasicData.LicenseInfo.Name),
		StarCount:    signal.Val(ghr.BasicData.StargazerCount),
		CreatedAt:    signal.Val(ghr.createdAt()),
		CreatedSince: signal.Val(legacy.TimeDelta(now, ghr.createdAt(), legacy.SinceDuration)),
		UpdatedAt:    signal.Val(ghr.updatedAt()),
		UpdatedSince: signal.Val(legacy.TimeDelta(now, ghr.updatedAt(), legacy.SinceDuration)),
		// Note: the /stats/commit-activity REST endpoint used in the legacy Python codebase is stale.
		CommitFrequency: signal.Val(legacy.Round(
			float64(ghr.BasicData.DefaultBranchRef.Target.Commit.RecentCommits.TotalCount)/legacyCommitLookbackWeeks, 2)),
	}
	ghr.logger.Debug("Fetching contributors")
	if contributors, err := legacy.FetchTotalContributors(ctx, ghr.client, ghr.owner(), ghr.name()); err != nil {
		return nil, err
	} else {
		s.ContributorCount.Set(contributors)
	}
	ghr.logger.Debug("Fetching org count")
	if orgCount, err := legacy.FetchOrgCount(ctx, ghr.client, ghr.owner(), ghr.name()); err != nil {
		return nil, err
	} else {
		s.OrgCount.Set(orgCount)
	}
	ghr.logger.Debug("Fetching releases")
	releaseCount, err := legacy.FetchReleaseCount(ctx, ghr.client, ghr.owner(), ghr.name(), legacyReleaseLookback)
	if err != nil {
		return nil, err
	}
	if releaseCount != 0 {
		s.RecentReleaseCount.Set(releaseCount)
	} else {
		daysSinceCreated := int(now.Sub(ghr.createdAt()).Hours()) / 24
		if daysSinceCreated > 0 {
			t := (ghr.BasicData.Tags.TotalCount * legacyReleaseLookbackDays) / daysSinceCreated
			s.RecentReleaseCount.Set(t)
		} else {
			s.RecentReleaseCount.Set(0)
		}
	}
	return s, nil
}

func (rc *RepoSource) IsSupported(p projectrepo.Repo) bool {
	_, ok := p.(*repo)
	return ok
}

type IssuesSource struct{}

func (ic *IssuesSource) EmptySet() signal.Set {
	return &signal.IssuesSet{}
}

func (ic *IssuesSource) Get(ctx context.Context, r projectrepo.Repo, _ string) (signal.Set, error) {
	ghr, ok := r.(*repo)
	if !ok {
		return nil, errors.New("project is not a github project")
	}
	s := &signal.IssuesSet{}

	if !ghr.issuesEnabled() {
		// The project tracks bugs somewhere other than GitHub, so there is
		// nothing here to count. Returning zeros would assert that nobody
		// reports issues against the project, which is a very different claim
		// and one that penalises a project for where it keeps its bug
		// tracker. Leave every field unset so scoring skips them.
		ghr.logger.Debug("Skipping issue signals: issue tracker is disabled")
		return s, nil
	}

	// TODO: the calculation of the frequency should be moved into the legacy
	// package. Ideally this would be behind an struct/interface that allows
	// caching and also removes the need to pass client, owner and name to each
	// function call.
	// One cutoff for the issue counts and the comment count, so every part of
	// comment frequency covers the same window.
	since := time.Now().UTC().Add(-legacy.IssueLookback)

	ghr.logger.Debug("Fetching issue counts")
	counts, err := legacy.FetchIssueCounts(ctx, ghr.client, ghr.owner(), ghr.name(), since)
	if err != nil {
		return nil, err
	}
	s.UpdatedCount.Set(counts.Updated)
	s.ClosedCount.Set(counts.Closed)

	if counts.Updated == 0 {
		// Comment frequency is comments per updated issue. With no updated
		// issues the ratio is undefined, so leave it unset rather than
		// recording a zero that reads as "nobody comments here".
		return s, nil
	}

	ghr.logger.Debug("Fetching comment frequency")
	comments, err := legacy.FetchIssueCommentCount(ctx, ghr.client, ghr.owner(), ghr.name(), since)
	switch {
	case errors.Is(err, legacy.ErrorTooManyResults):
		// The comment total is unknown, so record no value and let scoring
		// skip the field. Substituting a constant here used to place the
		// busiest repositories, the only ones that can produce this error,
		// below the typical measured frequency, which inverted the signal for
		// exactly the projects it was meant to reward.
		//
		// This is logged at warning level rather than debug: a signal is
		// missing from the output, and a run that quietly drops one for many
		// repositories should be visible without turning on debug logging.
		ghr.logger.With(zap.Error(err)).Warn("Comment count unavailable")
	case errors.Is(err, legacy.ErrorNoPageCount):
		// The endpoint no longer reports a page count, so the comment total is
		// unknown. Leave the field unset so scoring skips it, rather than
		// recording a zero that would read as "nobody comments here".
		ghr.logger.Warn("Comment count unavailable: endpoint no longer returns a page count")
	case err != nil:
		return nil, err
	default:
		s.CommentFrequency.Set(legacy.Round(float64(comments)/float64(counts.Updated), 2))
	}
	return s, nil
}

func (ic *IssuesSource) IsSupported(r projectrepo.Repo) bool {
	_, ok := r.(*repo)
	return ok
}
