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

package collector

import (
	"context"
	"testing"

	"go.uber.org/zap/zaptest"
)

var allSourceTypes = []SourceType{
	SourceTypeGithubRepo,
	SourceTypeGithubIssues,
	SourceTypeGitHubMentions,
	SourceTypeDepsDev,
}

func TestIsEnabled_AllEnabled(t *testing.T) {
	c := makeTestConfig(t, EnableAllSources())
	for _, sourceType := range allSourceTypes {
		t.Run(sourceType.String(), func(t *testing.T) {
			if !c.IsEnabled(sourceType) {
				t.Fatalf("IsEnabled(%s) = false, want true", sourceType)
			}
		})
	}
}

func TestIsEnabled_AllDisabled(t *testing.T) {
	c := makeTestConfig(t, DisableAllSources())
	for _, sourceType := range allSourceTypes {
		t.Run(sourceType.String(), func(t *testing.T) {
			if c.IsEnabled(sourceType) {
				t.Fatalf("IsEnabled(%s) = true, want false", sourceType)
			}
		})
	}
}

func TestIsEnabled_OneDisabled(t *testing.T) {
	c := makeTestConfig(t, EnableAllSources(), DisableSource(SourceTypeDepsDev))
	if c.IsEnabled(SourceTypeDepsDev) {
		t.Fatalf("IsEnabled(%s) = true, want false", SourceTypeDepsDev)
	}
}

func TestIsEnabled_OneEnabled(t *testing.T) {
	c := makeTestConfig(t, DisableAllSources(), EnableSource(SourceTypeDepsDev))
	if !c.IsEnabled(SourceTypeDepsDev) {
		t.Fatalf("IsEnabled(%s) = false, want true", SourceTypeDepsDev)
	}
}

func TestGCPProject(t *testing.T) {
	want := "my-project-id"
	c := makeTestConfig(t, GCPProject(want))
	if c.gcpProject != want {
		t.Fatalf("config.gcpProject = %q, want %q", c.gcpProject, want)
	}
}

func TestGCPDatasetName(t *testing.T) {
	want := "my-dataset-name"
	c := makeTestConfig(t, GCPDatasetName(want))
	if c.gcpDatasetName != want {
		t.Fatalf("config.gcpDatasetName = %q, want %q", c.gcpDatasetName, want)
	}
}

func makeTestConfig(t *testing.T, opts ...Option) *config {
	t.Helper()
	return makeConfig(context.Background(), zaptest.NewLogger(t), opts...)
}
