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
	"net/url"
	"testing"

	"github.com/ossf/criticality_score/v2/internal/collector/projectrepo"
)

// Match() only tests the hostname, so a github.com URL that names no
// repository still reaches queryBasicRepoData. Splitting the path then left
// fewer than two parts and indexing the second one panicked, taking the whole
// collection run with it.
func TestQueryBasicRepoDataRejectsURLWithoutARepo(t *testing.T) {
	for _, raw := range []string{
		"https://github.com/ossf",
		"https://github.com/ossf/",
		"https://github.com/",
		"https://github.com",
	} {
		t.Run(raw, func(t *testing.T) {
			u, err := url.Parse(raw)
			if err != nil {
				t.Fatalf("parsing %q: %v", raw, err)
			}

			// A nil client is fine: the path is rejected before it is used.
			_, err = queryBasicRepoData(context.Background(), nil, u)

			if err == nil {
				t.Fatalf("queryBasicRepoData(%q) returned no error", raw)
			}
			if !errors.Is(err, projectrepo.ErrNoRepoFound) {
				t.Errorf("queryBasicRepoData(%q) error = %v, want it to wrap ErrNoRepoFound", raw, err)
			}
		})
	}
}
