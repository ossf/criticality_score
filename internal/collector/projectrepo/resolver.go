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
	"errors"
	"fmt"
	"net/url"
)

// ErrNoFactoryFound is returned when there is no factory that can be used for
// a given URL.
var ErrNoFactoryFound = errors.New("factory not found for url")

// ErrNoRepoFound is returned when a factory cannot create a Repo for a given
// URL.
var ErrNoRepoFound = errors.New("repo not found")

// ErrRepoInaccessible is returned when a the Repo may exist, but is unable to
// access the repository for some reason.
var ErrRepoInaccessible = errors.New("repo inaccessible")

// Resolver is used to resolve a Repo url against a set of Factory instances
// registered with the resolver.
type Resolver struct {
	fs []Factory
}

func (r *Resolver) findFactory(u *url.URL) Factory {
	for _, f := range r.fs {
		if f.Match(u) {
			return f
		}
	}
	return nil
}

// Register adds the factory f to the set of factories that can be used for
// resolving a url to a Repo.
func (r *Resolver) Register(f Factory) {
	r.fs = append(r.fs, f)
}

// Resolve takes a url u and returns a corresponding instance of Repo if a
// matching factory has been registered.
//
// If a matching factory can not be found an ErrorNotFound will be returned.
//
// The factory may also return an error.
func (r *Resolver) Resolve(ctx context.Context, u *url.URL) (Repo, error) {
	f := r.findFactory(u)
	if f == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoFactoryFound, u)
	}
	return f.New(ctx, u)
}
