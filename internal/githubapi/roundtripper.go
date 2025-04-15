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
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v47/github"
	"go.uber.org/zap"

	"github.com/ossf/criticality_score/v2/internal/retry"
)

const (
	githubErrorIDSearch = "\"error_500\""
)

var (
	issuesRe        = regexp.MustCompile("^repos/[^/]+/[^/]+/issues$")
	issueCommentsRe = regexp.MustCompile("^repos/[^/]+/[^/]+/issues/comments$")
)

func NewRetryRoundTripper(rt http.RoundTripper, logger *zap.Logger) http.RoundTripper {
	s := &strategies{logger: logger}
	return retry.NewRoundTripper(rt,
		retry.InitialDelay(2*time.Minute),
		retry.RetryAfter(s.RetryAfter),
		retry.Strategy(s.SecondaryRateLimit),
		retry.Strategy(s.ServerError400),
		retry.Strategy(s.ServerError),
	)
}

type strategies struct {
	logger *zap.Logger
}

func respBodyContains(r *http.Response, search string) (bool, error) {
	data, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(data))
	if err != nil {
		return false, err
	}
	return bytes.Contains(data, []byte(search)), nil
}

// ServerError implements retry.RetryStrategyFn.
func (s *strategies) ServerError(r *http.Response) (retry.RetryStrategy, error) {
	if r.StatusCode < 500 || 600 <= r.StatusCode {
		return retry.NoRetry, nil
	}
	logger := s.logger.With(zap.Stringer("url", r.Request.URL))
	logger.With(zap.String("status", r.Status)).Warn("5xx: detected")
	path := strings.Trim(r.Request.URL.Path, "/")
	if issuesRe.MatchString(path) {
		logger.Warn("Ignoring /repos/X/Y/issues url.")
		// If the req url was /repos/[owner]/[name]/issues pass the
		// error through as it is likely a GitHub bug.
		return retry.NoRetry, nil
	}
	if issueCommentsRe.MatchString(path) {
		logger.Warn("Ignoring /repos/X/Y/issues/comments url.")
		// If the req url was /repos/[owner]/[name]/issues/comments pass the
		// error through as it is likely a GitHub bug.
		return retry.NoRetry, nil
	}
	return retry.RetryImmediate, nil
}

// ServerError400 implements retry.RetryStrategyFn.
func (s *strategies) ServerError400(r *http.Response) (retry.RetryStrategy, error) {
	if r.StatusCode != http.StatusBadRequest {
		return retry.NoRetry, nil
	}
	logger := s.logger.With(zap.Stringer("url", r.Request.URL))
	logger.Warn("400: bad request detected")
	if r.Header.Get("Content-Type") != "text/html" {
		return retry.NoRetry, nil
	}
	logger.Debug("It's a text/html doc")
	if isError, err := respBodyContains(r, githubErrorIDSearch); isError {
		logger.Debug("Found target string - assuming 500.")
		return retry.RetryImmediate, nil
	} else {
		return retry.NoRetry, err
	}
}

// SecondaryRateLimit implements retry.RetryStrategyFn.
func (s *strategies) SecondaryRateLimit(r *http.Response) (retry.RetryStrategy, error) {
	if r.StatusCode != http.StatusForbidden {
		return retry.NoRetry, nil
	}
	logger := s.logger.With(zap.Stringer("url", r.Request.URL))
	logger.Warn("403: forbidden detected")
	errorResponse := &github.ErrorResponse{Response: r}
	data, err := io.ReadAll(r.Body)
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(data))
	if err != nil || data == nil {
		logger.With(
			zap.Error(err),
			zap.Bool("data_nil", data == nil),
		).Warn("ReadAll failed")
		return retry.NoRetry, err
	}
	// Don't error check the unmarshall - if there is an error and the parsing
	// has failed then this function will return with no retry. A parsing error
	// here would mean the server did something wrong. Not attempting a retry
	// will cause the response to be processed again by go-github with an error
	// being generated there.
	json.Unmarshal(data, errorResponse)
	logger.With(
		zap.String("doc_url", errorResponse.DocumentationURL),
		zap.String("message", errorResponse.Message),
	).Warn("Error response data")
	if strings.HasSuffix(errorResponse.DocumentationURL, "#abuse-rate-limits") ||
		strings.HasSuffix(errorResponse.DocumentationURL, "#secondary-rate-limits") ||
		strings.HasSuffix(errorResponse.DocumentationURL, "#about-secondary-rate-limits") {
		logger.Warn("Secondary rate limit hit")
		return retry.RetryWithInitialDelay, nil
	}
	logger.Warn("Not an abuse rate limit error")
	return retry.NoRetry, nil
}

// RetryAfter implements retry.RetryAfterFn.
// TODO: move to retry once we're confident it is working.
func (s *strategies) RetryAfter(r *http.Response) time.Duration {
	if v := r.Header["Retry-After"]; len(v) > 0 {
		s.logger.Warn("Detected Retry-After header.")
		// According to GitHub support, the "Retry-After" header value will be
		// an integer which represents the number of seconds that one should
		// wait before resuming making requests.
		retryAfterSeconds, _ := strconv.ParseInt(v[0], 10, 64) // Error handling is noop.
		return time.Duration(retryAfterSeconds) * time.Second
	}
	return 0
}

// graphQLRoundTripper is used to make GraphQL errors produced by the GitHub
// GraphQL accessible.
//
// This allows the detection of NOT_FOUND responses, so we can accurately
// tell the difference between an error due to connectivity or server
// issues and a repository not existing.
type graphQLRoundTripper struct {
	inner http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface.
func (rt *graphQLRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := rt.inner.RoundTrip(r)
	if err != nil || resp.StatusCode != http.StatusOK {
		return resp, err
	}
	// Read the body in and close it.
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		// TODO: Do we need to do something with this error?
		return nil, err
	}

	var out struct {
		Data   *json.RawMessage
		Errors []GraphQLError
	}
	err = json.Unmarshal(body, &out)
	if err != nil {
		// TODO: Do we need to do something with this error?
		return nil, err
	}
	if len(out.Errors) > 0 {
		return nil, &GraphQLErrors{errors: out.Errors}
	}

	// Reset the body so that others can read it as well.
	resp.Body = io.NopCloser(bytes.NewBuffer(body))
	return resp, nil
}
