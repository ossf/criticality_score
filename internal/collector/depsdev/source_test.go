package depsdev

import (
	"context"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"go.uber.org/zap"

	"github.com/ossf/criticality_score/internal/collector/signal"
	mocks "github.com/ossf/criticality_score/internal/mock"
)

func Test_parseRepoURL(t *testing.T) {
	tests := []struct {
		name            string
		u               *url.URL
		wantProjectName string
		wantProjectType string
	}{
		{
			name: "github.com",
			u: &url.URL{
				Host: "github.com",
			},
			wantProjectName: "",
			wantProjectType: "GITHUB",
		},
		{
			name: "random",
			u: &url.URL{
				Host: "random",
			},
			wantProjectName: "",
			wantProjectType: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProjectName, gotProjectType := parseRepoURL(tt.u)
			if gotProjectName != tt.wantProjectName {
				t.Errorf("parseRepoURL() gotProjectName = %v, want %v", gotProjectName, tt.wantProjectName)
			}
			if gotProjectType != tt.wantProjectType {
				t.Errorf("parseRepoURL() gotProjectType = %v, want %v", gotProjectType, tt.wantProjectType)
			}
		})
	}
}

func Test_depsDevSource_Get(t *testing.T) {
	type args struct {
		ctx       context.Context
		rHostName string
		jobID     string
	}
	tests := []struct { //nolint:govet
		name       string
		logger     *zap.Logger
		dependents *dependents
		args       args
		want       signal.Set
		wantErr    bool
	}{
		{
			name:       "invalid url",
			logger:     zap.NewNop(),
			dependents: &dependents{},
			args: args{
				ctx:       context.Background(),
				rHostName: "random",
			},
			want: &depsDevSet{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			r := mocks.NewMockRepo(ctrl)

			r.EXPECT().URL().Return(&url.URL{Host: tt.args.rHostName}).AnyTimes()

			c := &depsDevSource{
				logger:     tt.logger,
				dependents: tt.dependents,
			}
			got, err := c.Get(tt.args.ctx, r, tt.args.jobID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewSource(t *testing.T) {
	type args struct {
		ctx         context.Context
		logger      *zap.Logger
		projectID   string
		datasetName string
		datasetTTL  time.Duration
	}
	test := struct { //nolint:govet
		name    string
		args    args
		want    signal.Source
		wantErr bool
	}{
		name: "new client error",
		args: args{
			ctx:         context.Background(),
			logger:      zap.NewNop(),
			projectID:   "",
			datasetName: "dataset",
			datasetTTL:  time.Hour,
		},
		wantErr: true,
	}
	got, err := NewSource(test.args.ctx, test.args.logger, test.args.projectID, test.args.datasetName, test.args.datasetTTL)
	if (err != nil) != test.wantErr {
		t.Errorf("NewSource() error = %v, wantErr %v", err, test.wantErr)
		return
	}
	if !reflect.DeepEqual(got, test.want) {
		t.Errorf("NewSource() got = %v, want %v", got, test.want)
	}
}

func Test_depsDevSource_IsSupported(t *testing.T) {
	type fields struct {
		logger     *zap.Logger
		dependents *dependents
	}
	tests := []struct {
		name      string
		fields    fields
		rHostName string
		want      bool
	}{
		{
			name: "github",
			fields: fields{
				logger: zap.NewNop(),
			},
			rHostName: "github.com",
			want:      true,
		},
		{
			name: "random",
			fields: fields{
				logger: zap.NewNop(),
			},
			rHostName: "random",
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &depsDevSource{
				logger:     tt.fields.logger,
				dependents: tt.fields.dependents,
			}
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			r := mocks.NewMockRepo(ctrl)

			r.EXPECT().URL().Return(&url.URL{Host: tt.rHostName}).AnyTimes()

			if got := c.IsSupported(r); got != tt.want {
				t.Errorf("IsSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}
