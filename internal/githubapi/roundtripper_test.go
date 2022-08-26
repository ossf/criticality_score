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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/ossf/criticality_score/internal/retry"
	log "github.com/sirupsen/logrus"
)

const (
	testAbuseRateLimitDocUrl     = "https://docs.github.com/en/rest/overview/resources-in-the-rest-api#abuse-rate-limits"
	testSecondaryRateLimitDocUrl = "https://docs.github.com/en/rest/overview/resources-in-the-rest-api#secondary-rate-limits"
)

func newTestStrategies() *strategies {
	logger := log.New()
	logger.Out = ioutil.Discard
	return &strategies{logger: logger}
}

type readerFn func(p []byte) (n int, err error)

// Read implements the io.Reader interface.
func (r readerFn) Read(p []byte) (n int, err error) {
	return r(p)
}

func TestRetryAfter(t *testing.T) {
	r := &http.Response{Header: http.Header{http.CanonicalHeaderKey("Retry-After"): {"123"}}}
	if d := newTestStrategies().RetryAfter(r); d != 123*time.Second {
		t.Fatalf("RetryAfter() == %d, want %v", d, 123*time.Second)
	}
}

func TestRetryAfter_NoHeader(t *testing.T) {
	if d := newTestStrategies().RetryAfter(&http.Response{}); d != 0 {
		t.Fatalf("RetryAfter() == %d, want 0", d)
	}
}

func TestRetryAfter_InvalidTime(t *testing.T) {
	r := &http.Response{Header: http.Header{http.CanonicalHeaderKey("Retry-After"): {"junk"}}}
	if d := newTestStrategies().RetryAfter(r); d != 0 {
		t.Fatalf("RetryAfter() == %d, want 0", d)
	}
}

func TestRetryAfter_ZeroTime(t *testing.T) {
	r := &http.Response{Header: http.Header{http.CanonicalHeaderKey("Retry-After"): {"0"}}}
	if d := newTestStrategies().RetryAfter(r); d != 0 {
		t.Fatalf("RetryAfter() == %d, want 0", d)
	}
}

func TestServerError(t *testing.T) {
	u, _ := url.Parse("https://api.github.com/repos/example/example")
	r := &http.Response{
		Request:    &http.Request{URL: u},
		StatusCode: http.StatusInternalServerError,
	}
	s, err := newTestStrategies().ServerError(r)
	if err != nil {
		t.Fatalf("ServerError() errored %v, want no error", err)
	}
	if s != retry.RetryImmediate {
		t.Fatalf("ServerError() == %v, want %v", s, retry.RetryImmediate)
	}
}

func TestServerError_IssueComments(t *testing.T) {
	u, _ := url.Parse("https://api.github.com/repos/example/example/issues/comments/")
	r := &http.Response{
		Request:    &http.Request{URL: u},
		StatusCode: http.StatusInternalServerError,
	}
	s, err := newTestStrategies().ServerError(r)
	if err != nil {
		t.Fatalf("ServerError() errored %v, want no error", err)
	}
	if s != retry.NoRetry {
		t.Fatalf("ServerError() == %v, want %v", s, retry.NoRetry)
	}
}

func TestServerError_StatusCodes(t *testing.T) {
	tests := []struct {
		statusCode int
		strategy   retry.RetryStrategy
	}{
		{statusCode: http.StatusOK, strategy: retry.NoRetry},
		{statusCode: http.StatusCreated, strategy: retry.NoRetry},
		{statusCode: http.StatusPermanentRedirect, strategy: retry.NoRetry},
		{statusCode: http.StatusFound, strategy: retry.NoRetry},
		{statusCode: http.StatusBadRequest, strategy: retry.NoRetry},
		{statusCode: http.StatusConflict, strategy: retry.NoRetry},
		{statusCode: http.StatusFailedDependency, strategy: retry.NoRetry},
		{statusCode: http.StatusGone, strategy: retry.NoRetry},
		{statusCode: http.StatusInternalServerError, strategy: retry.RetryImmediate},
		{statusCode: http.StatusNotImplemented, strategy: retry.RetryImmediate},
		{statusCode: http.StatusBadGateway, strategy: retry.RetryImmediate},
		{statusCode: http.StatusServiceUnavailable, strategy: retry.RetryImmediate},
		{statusCode: http.StatusGatewayTimeout, strategy: retry.RetryImmediate},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("status %d", test.statusCode), func(t *testing.T) {
			u, _ := url.Parse("https://api.github.com/repos/example/example")
			r := &http.Response{
				Request:    &http.Request{URL: u},
				StatusCode: test.statusCode,
			}
			s, _ := newTestStrategies().ServerError(r)
			if s != test.strategy {
				t.Fatalf("ServerError() == %v, want %v", s, test.strategy)
			}
		})
	}
}

func TestServerError400(t *testing.T) {
	r := &http.Response{
		Header:     http.Header{http.CanonicalHeaderKey("Content-Type"): {"text/html"}},
		StatusCode: http.StatusBadRequest,
		Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(`<html><body>This is <span id="error_500">an error</span></body></html>`))),
	}
	s, err := newTestStrategies().ServerError400(r)
	if err != nil {
		t.Fatalf("ServerError() errored %v, want no error", err)
	}
	if s != retry.RetryImmediate {
		t.Fatalf("ServerError() == %v, want %v", s, retry.RetryImmediate)
	}
}

func TestServerError400_NoMatchingString(t *testing.T) {
	r := &http.Response{
		Header:     http.Header{http.CanonicalHeaderKey("Content-Type"): {"text/html"}},
		StatusCode: http.StatusBadRequest,
		Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(`<html><body>Web Page</body></html`))),
	}
	s, err := newTestStrategies().ServerError400(r)
	if err != nil {
		t.Fatalf("ServerError() errored %v, want no error", err)
	}
	if s != retry.NoRetry {
		t.Fatalf("ServerError() == %v, want %v", s, retry.NoRetry)
	}
}

func TestServerError400_BodyError(t *testing.T) {
	want := errors.New("test error")
	r := &http.Response{
		Header:     http.Header{http.CanonicalHeaderKey("Content-Type"): {"text/html"}},
		StatusCode: http.StatusBadRequest,
		Body: ioutil.NopCloser(readerFn(func(b []byte) (int, error) {
			return 0, want
		})),
	}
	_, err := newTestStrategies().ServerError400(r)
	if err == nil {
		t.Fatalf("ServerError() returned no error, want %v", want)
	}
	if err != want {
		t.Fatalf("ServerError() errored %v, want %v", err, want)
	}
}

func TestServerError400_NotHTML(t *testing.T) {
	r := &http.Response{
		Header:     http.Header{http.CanonicalHeaderKey("Content-Type"): {"text/plain"}},
		StatusCode: http.StatusBadRequest,
		Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(`text doc`))),
	}
	s, err := newTestStrategies().ServerError400(r)
	if err != nil {
		t.Fatalf("ServerError() errored %v, want no error", err)
	}
	if s != retry.NoRetry {
		t.Fatalf("ServerError() == %v, want %v", s, retry.NoRetry)
	}
}

func TestServerError400_StatusCodes(t *testing.T) {
	tests := []struct {
		statusCode int
		strategy   retry.RetryStrategy
	}{
		{statusCode: http.StatusOK, strategy: retry.NoRetry},
		{statusCode: http.StatusCreated, strategy: retry.NoRetry},
		{statusCode: http.StatusPermanentRedirect, strategy: retry.NoRetry},
		{statusCode: http.StatusFound, strategy: retry.NoRetry},
		{statusCode: http.StatusBadRequest, strategy: retry.RetryImmediate},
		{statusCode: http.StatusConflict, strategy: retry.NoRetry},
		{statusCode: http.StatusFailedDependency, strategy: retry.NoRetry},
		{statusCode: http.StatusGone, strategy: retry.NoRetry},
		{statusCode: http.StatusInternalServerError, strategy: retry.NoRetry},
		{statusCode: http.StatusNotImplemented, strategy: retry.NoRetry},
		{statusCode: http.StatusBadGateway, strategy: retry.NoRetry},
		{statusCode: http.StatusServiceUnavailable, strategy: retry.NoRetry},
		{statusCode: http.StatusGatewayTimeout, strategy: retry.NoRetry},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("status %d", test.statusCode), func(t *testing.T) {
			r := &http.Response{
				Header:     http.Header{http.CanonicalHeaderKey("Content-Type"): {"text/html"}},
				StatusCode: test.statusCode,
				Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(`<html><body>This is <span id="error_500">an error</span></body></html>`))),
			}
			s, err := newTestStrategies().ServerError400(r)
			if err != nil {
				t.Fatalf("ServerError() errored %v, want no error", err)
			}
			if s != test.strategy {
				t.Fatalf("ServerError() == %v, want %v", s, test.strategy)
			}
		})
	}
}

func TestSecondaryRateLimit(t *testing.T) {
	r := &http.Response{
		StatusCode: http.StatusForbidden,
		Body: ioutil.NopCloser(bytes.NewBuffer(
			[]byte(fmt.Sprintf(`{"message": "test", "documentation_url": "%s"}`, testSecondaryRateLimitDocUrl)))),
	}
	s, err := newTestStrategies().SecondaryRateLimit(r)
	if err != nil {
		t.Fatalf("ServerError() errored %v, want no error", err)
	}
	if s != retry.RetryWithInitialDelay {
		t.Fatalf("ServerError() == %v, want %v", s, retry.RetryWithInitialDelay)
	}
}

func TestSecondaryRateLimit_AbuseUrl(t *testing.T) {
	r := &http.Response{
		StatusCode: http.StatusForbidden,
		Body: ioutil.NopCloser(bytes.NewBuffer(
			[]byte(fmt.Sprintf(`{"message": "test", "documentation_url": "%s"}`, testAbuseRateLimitDocUrl)))),
	}
	s, err := newTestStrategies().SecondaryRateLimit(r)
	if err != nil {
		t.Fatalf("ServerError() errored %v, want no error", err)
	}
	if s != retry.RetryWithInitialDelay {
		t.Fatalf("ServerError() == %v, want %v", s, retry.RetryWithInitialDelay)
	}
}

func TestSecondaryRateLimit_OtherUrl(t *testing.T) {
	r := &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(`{"message": "test", "documentation_url": "https://example.org/"}`))),
	}
	s, err := newTestStrategies().SecondaryRateLimit(r)
	if err != nil {
		t.Fatalf("ServerError() errored %v, want no error", err)
	}
	if s != retry.NoRetry {
		t.Fatalf("ServerError() == %v, want %v", s, retry.NoRetry)
	}
}

func TestSecondaryRateLimit_BodyError(t *testing.T) {
	want := errors.New("test error")
	r := &http.Response{
		StatusCode: http.StatusForbidden,
		Body: ioutil.NopCloser(readerFn(func(b []byte) (int, error) {
			return 0, want
		})),
	}
	_, err := newTestStrategies().SecondaryRateLimit(r)
	if err == nil {
		t.Fatalf("ServerError() returned no error, want %v", want)
	}
	if err != want {
		t.Fatalf("ServerError() errored %v, want %v", err, want)
	}
}

func TestSecondaryRateLimit_StatusCodes(t *testing.T) {
	tests := []struct {
		statusCode int
		strategy   retry.RetryStrategy
	}{
		{statusCode: http.StatusOK, strategy: retry.NoRetry},
		{statusCode: http.StatusCreated, strategy: retry.NoRetry},
		{statusCode: http.StatusPermanentRedirect, strategy: retry.NoRetry},
		{statusCode: http.StatusFound, strategy: retry.NoRetry},
		{statusCode: http.StatusBadRequest, strategy: retry.NoRetry},
		{statusCode: http.StatusForbidden, strategy: retry.RetryWithInitialDelay},
		{statusCode: http.StatusConflict, strategy: retry.NoRetry},
		{statusCode: http.StatusFailedDependency, strategy: retry.NoRetry},
		{statusCode: http.StatusGone, strategy: retry.NoRetry},
		{statusCode: http.StatusInternalServerError, strategy: retry.NoRetry},
		{statusCode: http.StatusNotImplemented, strategy: retry.NoRetry},
		{statusCode: http.StatusBadGateway, strategy: retry.NoRetry},
		{statusCode: http.StatusServiceUnavailable, strategy: retry.NoRetry},
		{statusCode: http.StatusGatewayTimeout, strategy: retry.NoRetry},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("status %d", test.statusCode), func(t *testing.T) {
			r := &http.Response{
				StatusCode: test.statusCode,
				Body: ioutil.NopCloser(bytes.NewBuffer(
					[]byte(fmt.Sprintf(`{"message": "test", "documentation_url": "%s"}`, testSecondaryRateLimitDocUrl)))),
			}
			s, err := newTestStrategies().SecondaryRateLimit(r)
			if err != nil {
				t.Fatalf("ServerError() errored %v, want no error", err)
			}
			if s != test.strategy {
				t.Fatalf("ServerError() == %v, want %v", s, test.strategy)
			}
		})
	}
}
