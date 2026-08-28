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

package legacy

import (
	"errors"
	"time"
)

const (
	SinceDuration time.Duration = time.Hour * 24 * 30
	IssueLookback time.Duration = time.Hour * 24 * 90

	// TODO: these limits should ultimately be imposed by the score generation, not here.
	MaxContributorLimit = 5000
	MaxTopContributors  = 15

	TooManyContributorsOrgCount = 10

	releasesPerPage = 100
)

var ErrorTooManyResults = errors.New("too many results")

// ErrorNoPageCount is returned when a REST response has a "next" page link but
// no "last" page link.
//
// Several signals are derived by requesting one result per page and reading the
// total from the "last" link. Endpoints that move to cursor based pagination
// stop sending that link, and the page count silently reads as zero. Returning
// this error keeps a migrated endpoint from being mistaken for an empty
// repository.
var ErrorNoPageCount = errors.New("response has no last page link")
