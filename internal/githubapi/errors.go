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

package githubapi

import (
	"errors"
	"fmt"

	"github.com/google/go-github/v47/github"
)

// ErrorResponseStatusCode will unwrap a github.ErrorResponse and return the
// status code inside.
//
// If the error is nil, or not an ErrorResponse it will return a status code of
// 0.
func ErrorResponseStatusCode(err error) int {
	if err == nil {
		return 0
	}
	var e *github.ErrorResponse
	ok := errors.As(err, &e)
	if !ok {
		return 0
	}
	return e.Response.StatusCode
}

var (
	// ErrGraphQLNotFound is an error used to test when GitHub GraphQL query
	// returns a single error with the type "NOT_FOUND".
	//
	// It should be used with errors.Is.
	ErrGraphQLNotFound = errors.New("GraphQL resource not found")

	// ErrGraphQLForbidden is an error used to test when GitHub GraphQL query
	// returns a single error with the type "FORBIDDEN".
	//
	// It should be used with errors.Is.
	ErrGraphQLForbidden = errors.New("GraphQL access forbidden")
)

const (
	// gitHubGraphQLNotFoundType matches the NOT_FOUND type field returned
	// in GitHub's GraphQL errors.
	//
	// GraphQL errors are required to have a Message, and optional Path and
	// Locations. Type is a non-standard field available on GitHub's API.
	gitHubGraphQLNotFoundType = "NOT_FOUND"

	// gitHubGraphQLForbiddenType matches the FORBIDDEN type field returned
	// in GitHub's GraphQL errors.
	//
	// This error type is used when the authorization token has been blocked
	// from accessing the repository. Usually due to an IP address block.
	gitHubGraphQLForbiddenType = "FORBIDDEN"
)

// GraphQLError stores the error result from a GitHub GraphQL query.
type GraphQLError struct {
	Message   string
	Type      string // GitHub specific GraphQL error field
	Locations []struct {
		Line   int
		Column int
	}
}

// GraphQLErrors wraps all the errors returned by a GraphQL response.
type GraphQLErrors struct {
	errors []GraphQLError
}

// Error implements error interface.
func (e *GraphQLErrors) Error() string {
	switch len(e.errors) {
	case 0:
		panic("no errors found")
	case 1:
		return fmt.Sprintf("%s (type: %s)", e.errors[0].Message, e.errors[0].Type)
	default:
		return fmt.Sprintf("%d GraphQL errors", len(e.errors))
	}
}

// HasType returns true if one of the errors matches the supplied type.
func (e *GraphQLErrors) HasType(t string) bool {
	for _, anError := range e.errors {
		if anError.Type == t {
			return true
		}
	}
	return false
}

// Errors returns a slice with each Error returned by the GraphQL API.
func (e *GraphQLErrors) Errors() []GraphQLError {
	return e.errors
}

// Is implements the errors.Is interface.
func (e *GraphQLErrors) Is(target error) bool {
	if target == ErrGraphQLNotFound {
		return len(e.errors) == 1 && e.HasType(gitHubGraphQLNotFoundType)
	}
	if target == ErrGraphQLForbidden {
		return len(e.errors) == 1 && e.HasType(gitHubGraphQLForbiddenType)
	}
	return false
}
