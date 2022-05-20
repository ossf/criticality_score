package github

import (
	"context"
	"net/url"

	"github.com/ossf/criticality_score/cmd/collect_signals/projectrepo"
	"github.com/ossf/criticality_score/internal/githubapi"
	log "github.com/sirupsen/logrus"
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
