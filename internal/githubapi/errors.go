// Copyright 2022 Google LLC
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

package githubapi

import "github.com/google/go-github/v44/github"

// ErrorResponseStatusCode will unwrap a github.ErrorResponse and return the
// status code inside.
//
// If the error is nil, or not an ErrorResponse it will return a status code of
// 0.
func ErrorResponseStatusCode(err error) int {
	if err == nil {
		return 0
	}
	e, ok := err.(*github.ErrorResponse)
	if !ok {
		return 0
	}
	return e.Response.StatusCode
}
