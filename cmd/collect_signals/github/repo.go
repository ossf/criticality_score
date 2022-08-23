package github

import (
	"context"
	"net/url"
	"time"

	"github.com/ossf/criticality_score/cmd/collect_signals/github/legacy"
	"github.com/ossf/criticality_score/internal/githubapi"
	log "github.com/sirupsen/logrus"
)

// repo implements the projectrepo.Repo interface for a GitHub repository.
type repo struct {
	client  *githubapi.Client
	origURL *url.URL
	logger  *log.Entry

	BasicData *basicRepoData
	realURL   *url.URL
	created   time.Time
}

// URL implements the projectrepo.Repo interface.
func (r *repo) URL() *url.URL {
	return r.realURL
}

func (r *repo) init(ctx context.Context) error {
	if r.BasicData != nil {
		// Already finished. Don't init() more than once.
		return nil
	}
	r.logger.Debug("Fetching basic data from GitHub")
	data, err := queryBasicRepoData(ctx, r.client.GraphQL(), r.origURL)
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
