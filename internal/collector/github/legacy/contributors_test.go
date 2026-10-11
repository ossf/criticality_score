// Copyright 2026 Criticality Score Authors
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

package legacy

import (
	"testing"

	"github.com/google/go-github/v47/github"
)

func TestIsBot(t *testing.T) {
	tests := []struct {
		login       string
		accountType string
		want        bool
	}{
		{login: "octocat", accountType: "User", want: false},
		{login: "octocat", want: false},
		{login: "dependabot[bot]", accountType: "Bot", want: true},
		{login: "renovate[bot]", want: true},
		// GitHub's Copilot coding agent: a bot without the "[bot]" suffix.
		{login: "Copilot", accountType: "Bot", want: true},
	}
	for _, tt := range tests {
		c := &github.Contributor{Login: github.String(tt.login)}
		if tt.accountType != "" {
			c.Type = github.String(tt.accountType)
		}
		if got := isBot(c); got != tt.want {
			t.Errorf("isBot(login=%q, type=%q) = %v, want %v", tt.login, tt.accountType, got, tt.want)
		}
	}
}
