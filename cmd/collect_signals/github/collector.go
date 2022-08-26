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

	"github.com/ossf/criticality_score/cmd/collect_signals/github/legacy"
	"github.com/ossf/criticality_score/cmd/collect_signals/projectrepo"
	"github.com/ossf/criticality_score/cmd/collect_signals/signal"
)

type RepoCollector struct {
}

func (rc *RepoCollector) EmptySet() signal.Set {
	return &signal.RepoSet{}
}

func (rc *RepoCollector) Collect(ctx context.Context, r projectrepo.Repo) (signal.Set, error) {
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
		CommitFrequency: signal.Val(legacy.Round(float64(ghr.BasicData.DefaultBranchRef.Target.Commit.RecentCommits.TotalCount)/52, 2)),
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
	if releaseCount, err := legacy.FetchReleaseCount(ctx, ghr.client, ghr.owner(), ghr.name(), legacyReleaseLookback); err != nil {
		return nil, err
	} else {
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
	}
	return s, nil
}

func (rc *RepoCollector) IsSupported(p projectrepo.Repo) bool {
	_, ok := p.(*repo)
	return ok
}

type IssuesCollector struct {
}

func (ic *IssuesCollector) EmptySet() signal.Set {
	return &signal.IssuesSet{}
}

func (ic *IssuesCollector) Collect(ctx context.Context, r projectrepo.Repo) (signal.Set, error) {
	ghr, ok := r.(*repo)
	if !ok {
		return nil, errors.New("project is not a github project")
	}
	s := &signal.IssuesSet{}

	ghr.logger.Debug("Fetching closed issues")
	closed, err := legacy.FetchIssueCount(ctx, ghr.client, ghr.owner(), ghr.name(), legacy.IssueStateClosed, legacy.IssueLookback)
	if err != nil {
		return nil, err
	}
	s.ClosedCount.Set(closed)

	// TODO: the calculation of the frequency should be moved into the legacy
	// package. Ideally this would be behind an struct/interface that allows
	// caching and also removes the need to pass client, owner and name to each
	// function call.
	ghr.logger.Debug("Fetching updated issues")
	up, err := legacy.FetchIssueCount(ctx, ghr.client, ghr.owner(), ghr.name(), legacy.IssueStateAll, legacy.IssueLookback)
	if err != nil {
		return nil, err
	}
	s.UpdatedCount.Set(up)

	if up == 0 {
		s.CommentFrequency.Set(0)
		return s, nil
	}

	ghr.logger.Debug("Fetching comment frequency")
	comments, err := legacy.FetchIssueCommentCount(ctx, ghr.client, ghr.owner(), ghr.name(), legacy.IssueLookback)
	if errors.Is(err, legacy.TooManyResultsError) {
		ghr.logger.Debug("Comment count failed with too many result")
		s.CommentFrequency.Set(legacy.TooManyCommentsFrequency)
	} else if err != nil {
		return nil, err
	} else {
		s.CommentFrequency.Set(legacy.Round(float64(comments)/float64(up), 2))
	}
	return s, nil
}

func (ic *IssuesCollector) IsSupported(r projectrepo.Repo) bool {
	_, ok := r.(*repo)
	return ok
}
