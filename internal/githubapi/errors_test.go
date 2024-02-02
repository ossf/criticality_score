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
	"net/http"
	"testing"

	"github.com/google/go-github/v47/github"
)

func TestErrorResponseStatusCode(t *testing.T) {
	tests := []struct { //nolint:govet
		name string
		err  error
		want int
	}{
		{
			name: "nil error",
			want: 0,
		},
		{
			name: "non-nil error that is not an ErrorResponse",
			err:  errors.New("some error"),
			want: 0,
		},
		{
			name: "error that is an ErrorResponse",
			err: &github.ErrorResponse{
				Response: &http.Response{
					StatusCode: 404,
				},
			},
			want: 404,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ErrorResponseStatusCode(test.err); got != test.want {
				t.Errorf("ErrorResponseStatusCode() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestGraphQLErrors_Error(t *testing.T) {
	tests := []struct { //nolint:govet
		name      string
		errors    []GraphQLError
		want      string
		wantPanic bool
	}{
		{
			name:      "zero errors",
			errors:    []GraphQLError{},
			wantPanic: true,
		},
		{
			name: "one error",
			errors: []GraphQLError{
				{Message: "one", Type: "A_TYPE"},
			},
			want: "one (type: A_TYPE)",
		},
		{
			name: "more than one error",
			errors: []GraphQLError{
				{Message: "one"},
				{Message: "two"},
				{Message: "three"},
			},
			want: "3 GraphQL errors",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r != nil && !test.wantPanic {
					t.Fatalf("Error() panic: %v, want no panic", r)
				}
			}()

			e := &GraphQLErrors{test.errors}

			if got := e.Error(); got != test.want {
				t.Errorf("Error() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestGraphQLErrors_HasType(t *testing.T) {
	tests := []struct { //nolint:govet
		name   string
		errors []GraphQLError
		t      string
		want   bool
	}{
		{
			name: "type is equal to t",
			errors: []GraphQLError{
				{Type: "NOT_FOUND"},
			},
			t:    "NOT_FOUND",
			want: true,
		},
		{
			name:   "type without NOT_FOUND",
			errors: []GraphQLError{},
			t:      "NOT_FOUND",
			want:   false,
		},
		{
			name: "type multiple type fields not set to t",
			errors: []GraphQLError{
				{Type: "random_1"},
				{Type: "random_2"},
				{Type: "random_3"},
			},
			t:    "NOT_FOUND",
			want: false,
		},
		{
			name: "type is not equal to t",
			errors: []GraphQLError{
				{Type: "NOT_FOUND"},
			},
			t:    "invalid type",
			want: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e := &GraphQLErrors{
				errors: test.errors,
			}
			if got := e.HasType(test.t); got != test.want {
				t.Errorf("HasType() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestGraphQLErrors_Is(t *testing.T) {
	tests := []struct { //nolint:govet
		name   string
		errors []GraphQLError
		target error
		want   bool
	}{
		{
			name: "target equals ErrGraphQLNotFound and returns true",
			errors: []GraphQLError{
				{Message: "one", Type: "NOT_FOUND"},
			},
			target: ErrGraphQLNotFound,
			want:   true,
		},
		{
			name: "target equals ErrGraphQLNotFound and returns false",
			errors: []GraphQLError{
				{Message: "one"},
			},
			target: ErrGraphQLNotFound,
			want:   false,
		},
		{
			name: "target equals ErrGraphQLForbidden and returns true",
			errors: []GraphQLError{
				{Message: "one", Type: "FORBIDDEN"},
			},
			target: ErrGraphQLForbidden,
			want:   true,
		},
		{
			name: "target equals ErrGraphQLForbidden and returns false",
			errors: []GraphQLError{
				{Message: "one"},
			},
			target: ErrGraphQLForbidden,
			want:   false,
		},
		{
			name: "regular testcase",
			errors: []GraphQLError{
				{Message: "one"},
			},
			target: errors.New("some error"),
			want:   false,
		},
		{
			name: "multiple errors with only one having the error type as 'NOT_FOUND'",
			errors: []GraphQLError{
				{Message: "one"},
				{Message: "two", Type: "NOT_FOUND"},
				{Message: "three"},
			},
			target: ErrGraphQLNotFound,
			want:   false,
		},
		{
			name: "multiple errors with only one having the error type as 'FORBIDDEN'",
			errors: []GraphQLError{
				{Message: "one"},
				{Message: "two", Type: "FORBIDDEN"},
				{Message: "three"},
			},
			target: ErrGraphQLForbidden,
			want:   false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e := &GraphQLErrors{
				errors: test.errors,
			}
			if got := e.Is(test.target); got != test.want {
				t.Errorf("Is() = %v, want %v", got, test.want)
			}
		})
	}
}
