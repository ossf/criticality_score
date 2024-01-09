package github

import (
	"context"
	"errors"
	"net/url"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"go.uber.org/zap"

	"github.com/ossf/criticality_score/internal/collector/projectrepo"
	"github.com/ossf/criticality_score/internal/githubapi"
)

func Test_factory_New(t *testing.T) {
	type fields struct {
		client *githubapi.Client
		logger *zap.Logger
	}
	type args struct {
		ctx context.Context
		u   *url.URL
	}
	tests := []struct { //nolint:govet
		name     string
		fields   fields
		args     args
		repoData *BasicRepoData
		queryErr error
		want     projectrepo.Repo
		wantErr  bool
	}{
		{
			name: "r.Init() error is not equal to nil",
			fields: fields{
				client: &githubapi.Client{},
				logger: zap.NewNop(),
			},
			args: args{
				ctx: context.Background(),
				u: &url.URL{
					Host: "github.com",
					Path: "/ossf/criticality_score",
				},
			},
			queryErr: errors.New("query error"),
			wantErr:  true,
		},
		{
			name: "query error is ErrGraphQLNotFound",
			fields: fields{
				client: &githubapi.Client{},
				logger: zap.NewNop(),
			},
			args: args{
				ctx: context.Background(),
				u: &url.URL{
					Host: "github.com",
					Path: "/ossf/criticality_score",
				},
			},
			queryErr: githubapi.ErrGraphQLNotFound,
			wantErr:  true,
		},

		// TODO: add test for r.Init not returning error
		// Not able to do this right now because of legacy.FetchCreatedTime is not yet mocked.
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := NewRepoFactory(test.fields.client, test.fields.logger) // testing NewRepoFactory

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			q := NewMockQuery(ctrl)
			q.EXPECT().QueryBasicRepoData(gomock.Any(), gomock.Any(), gomock.Any()).Return(
				test.repoData, test.queryErr).AnyTimes()

			f = &factory{
				client: f.(*factory).client,
				logger: f.(*factory).logger,
				query:  q,
			}

			got, err := f.New(test.args.ctx, test.args.u)
			if (err != nil) != test.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("New() got = %v, want %v", got, test.want)
			}
		})
	}
}

func Test_factory_Match(t *testing.T) {
	type fields struct {
		client *githubapi.Client
		logger *zap.Logger
	}
	tests := []struct { //nolint:govet
		name   string
		fields fields
		u      *url.URL
		want   bool
	}{
		{
			name: "valid",
			fields: fields{
				client: &githubapi.Client{},
				logger: zap.NewNop(),
			},
			u: &url.URL{
				Host: "github.com",
			},
			want: true,
		},
		{
			name: "invalid",
			fields: fields{
				client: &githubapi.Client{},
				logger: zap.NewNop(),
			},
			u: &url.URL{
				Host: "invalid.com",
			},
			want: false,
		},
		{
			name: "is a host name",
			fields: fields{
				client: &githubapi.Client{},
				logger: zap.NewNop(),
			},
			u: &url.URL{
				Host: "[github.com]:123",
			},
			want: true,
		},
		{
			name: "uppercase",
			fields: fields{
				client: &githubapi.Client{},
				logger: zap.NewNop(),
			},
			u: &url.URL{
				Host: "GITHUB.COM",
			},
			want: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := NewRepoFactory(test.fields.client, test.fields.logger)

			if got := f.Match(test.u); got != test.want {
				t.Errorf("Match() = %v, want %v", got, test.want)
			}
		})
	}
}
