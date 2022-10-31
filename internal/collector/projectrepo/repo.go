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

package projectrepo

import (
	"context"
	"net/url"
)

// Repo is the core interface representing a project's source repository.
type Repo interface {
	URL() *url.URL
}

// Factory is used to obtain new instances of Repo.
type Factory interface {
	// New returns a new instance of Repo for the supplied URL.
	//
	// If the project can not be found, the error NoRepoFound will be returned.
	//
	// If the project is not valid for use, or there is any other issue creating
	// the Repo, Repo will be nil and an error will be returned.
	New(context.Context, *url.URL) (Repo, error)

	// Match returns true if this factory can create a new instance of Repo
	// repository for the given repository URL.
	Match(*url.URL) bool
}
