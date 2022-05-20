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
