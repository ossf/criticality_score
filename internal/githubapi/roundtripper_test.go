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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/ossf/criticality_score/v2/internal/retry"
)

const (
	testAbuseRateLimitDocURL     = "https://docs.github.com/en/rest/overview/resources-in-the-rest-api#abuse-rate-limits"
	testSecondaryRateLimitDocURL = "https://docs.github.com/en/rest/overview/resources-in-the-rest-api#secondary-rate-limits"
)

func newTestStrategies(t *testing.T) *strategies {
	t.Helper()
	return &strategies{logger: zaptest.NewLogger(t)}
}

type readerFn func(p []byte) (n int, err error)

// Read implements the io.Reader interface.
func (r readerFn) Read(p []byte) (n int, err error) {
	return r(p)
}

func TestRetryAfter(t *testing.T) {
	r := &http.Response{Header: http.Header{http.CanonicalHeaderKey("Retry-After"): {"123"}}}
	if d := newTestStrategies(t).RetryAfter(r); d != 123*time.Second {
		t.Fatalf("RetryAfter() == %d, want %v", d, 123*time.Second)
	}
}

func TestRetryAfter_NoHeader(t *testing.T) {
	if d := newTestStrategies(t).RetryAfter(&http.Response{}); d != 0 {
		t.Fatalf("RetryAfter() == %d, want 0", d)
	}
}

func TestRetryAfter_InvalidTime(t *testing.T) {
	r := &http.Response{Header: http.Header{http.CanonicalHeaderKey("Retry-After"): {"junk"}}}
	if d := newTestStrategies(t).RetryAfter(r); d != 0 {
		t.Fatalf("RetryAfter() == %d, want 0", d)
	}
}

func TestRetryAfter_ZeroTime(t *testing.T) {
	r := &http.Response{Header: http.Header{http.CanonicalHeaderKey("Retry-After"): {"0"}}}
	if d := newTestStrategies(t).RetryAfter(r); d != 0 {
		t.Fatalf("RetryAfter() == %d, want 0", d)
	}
}

func TestServerError(t *testing.T) {
	u, _ := url.Parse("https://api.github.com/repos/example/example")
	r := &http.Response{
		Request:    &http.Request{URL: u},
		StatusCode: http.StatusInternalServerError,
	}
	s, err := newTestStrategies(t).ServerError(r)
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
	s, err := newTestStrategies(t).ServerError(r)
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
			s, _ := newTestStrategies(t).ServerError(r)
			if s != test.strategy {
				t.Fatalf("ServerError() == %v, want %v", s, test.strategy)
			}
		})
	}
}

func TestServerError400(t *testing.T) {
	u, _ := url.Parse("https://api.github.com/repos/example/example")
	r := &http.Response{
		Request:    &http.Request{URL: u},
		Header:     http.Header{http.CanonicalHeaderKey("Content-Type"): {"text/html"}},
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(`<html><body>This is <span id="error_500">an error</span></body></html>`)),
	}
	s, err := newTestStrategies(t).ServerError400(r)
	if err != nil {
		t.Fatalf("ServerError() errored %v, want no error", err)
	}
	if s != retry.RetryImmediate {
		t.Fatalf("ServerError() == %v, want %v", s, retry.RetryImmediate)
	}
}

func TestServerError400_NoMatchingString(t *testing.T) {
	u, _ := url.Parse("https://api.github.com/repos/example/example")
	r := &http.Response{
		Request:    &http.Request{URL: u},
		Header:     http.Header{http.CanonicalHeaderKey("Content-Type"): {"text/html"}},
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(`<html><body>Web Page</body></html`)),
	}
	s, err := newTestStrategies(t).ServerError400(r)
	if err != nil {
		t.Fatalf("ServerError() errored %v, want no error", err)
	}
	if s != retry.NoRetry {
		t.Fatalf("ServerError() == %v, want %v", s, retry.NoRetry)
	}
}

func TestServerError400_BodyError(t *testing.T) {
	want := errors.New("test error")
	u, _ := url.Parse("https://api.github.com/repos/example/example")
	r := &http.Response{
		Request:    &http.Request{URL: u},
		Header:     http.Header{http.CanonicalHeaderKey("Content-Type"): {"text/html"}},
		StatusCode: http.StatusBadRequest,
		Body: io.NopCloser(readerFn(func(b []byte) (int, error) {
			return 0, want
		})),
	}
	_, err := newTestStrategies(t).ServerError400(r)
	if err == nil {
		t.Fatalf("ServerError() returned no error, want %v", want)
	}
	if !errors.Is(err, want) {
		t.Fatalf("ServerError() errored %v, want %v", err, want)
	}
}

func TestServerError400_NotHTML(t *testing.T) {
	u, _ := url.Parse("https://api.github.com/repos/example/example")
	r := &http.Response{
		Request:    &http.Request{URL: u},
		Header:     http.Header{http.CanonicalHeaderKey("Content-Type"): {"text/plain"}},
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(`text doc`)),
	}
	s, err := newTestStrategies(t).ServerError400(r)
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
			u, _ := url.Parse("https://api.github.com/repos/example/example")
			r := &http.Response{
				Request:    &http.Request{URL: u},
				Header:     http.Header{http.CanonicalHeaderKey("Content-Type"): {"text/html"}},
				StatusCode: test.statusCode,
				Body:       io.NopCloser(strings.NewReader(`<html><body>This is <span id="error_500">an error</span></body></html>`)),
			}
			s, err := newTestStrategies(t).ServerError400(r)
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
	u, _ := url.Parse("https://api.github.com/repos/example/example")
	r := &http.Response{
		Request:    &http.Request{URL: u},
		StatusCode: http.StatusForbidden,
		Body: io.NopCloser(strings.NewReader(
			fmt.Sprintf(`{"message": "test", "documentation_url": "%s"}`, testSecondaryRateLimitDocURL))),
	}
	s, err := newTestStrategies(t).SecondaryRateLimit(r)
	if err != nil {
		t.Fatalf("ServerError() errored %v, want no error", err)
	}
	if s != retry.RetryWithInitialDelay {
		t.Fatalf("ServerError() == %v, want %v", s, retry.RetryWithInitialDelay)
	}
}

func TestSecondaryRateLimit_AbuseUrl(t *testing.T) {
	u, _ := url.Parse("https://api.github.com/repos/example/example")
	r := &http.Response{
		Request:    &http.Request{URL: u},
		StatusCode: http.StatusForbidden,
		Body: io.NopCloser(strings.NewReader(
			fmt.Sprintf(`{"message": "test", "documentation_url": "%s"}`, testAbuseRateLimitDocURL))),
	}
	s, err := newTestStrategies(t).SecondaryRateLimit(r)
	if err != nil {
		t.Fatalf("ServerError() errored %v, want no error", err)
	}
	if s != retry.RetryWithInitialDelay {
		t.Fatalf("ServerError() == %v, want %v", s, retry.RetryWithInitialDelay)
	}
}

func TestSecondaryRateLimit_OtherUrl(t *testing.T) {
	u, _ := url.Parse("https://api.github.com/repos/example/example")
	r := &http.Response{
		Request:    &http.Request{URL: u},
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(strings.NewReader(`{"message": "test", "documentation_url": "https://example.org/"}`)),
	}
	s, err := newTestStrategies(t).SecondaryRateLimit(r)
	if err != nil {
		t.Fatalf("ServerError() errored %v, want no error", err)
	}
	if s != retry.NoRetry {
		t.Fatalf("ServerError() == %v, want %v", s, retry.NoRetry)
	}
}

func TestSecondaryRateLimit_BodyError(t *testing.T) {
	want := errors.New("test error")
	u, _ := url.Parse("https://api.github.com/repos/example/example")
	r := &http.Response{
		Request:    &http.Request{URL: u},
		StatusCode: http.StatusForbidden,
		Body: io.NopCloser(readerFn(func(b []byte) (int, error) {
			return 0, want
		})),
	}
	_, err := newTestStrategies(t).SecondaryRateLimit(r)
	if err == nil {
		t.Fatalf("ServerError() returned no error, want %v", want)
	}
	if !errors.Is(err, want) {
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
			u, _ := url.Parse("https://api.github.com/repos/example/example")
			r := &http.Response{
				Request:    &http.Request{URL: u},
				StatusCode: test.statusCode,
				Body: io.NopCloser(strings.NewReader(
					fmt.Sprintf(`{"message": "test", "documentation_url": "%s"}`, testSecondaryRateLimitDocURL))),
			}
			s, err := newTestStrategies(t).SecondaryRateLimit(r)
			if err != nil {
				t.Fatalf("ServerError() errored %v, want no error", err)
			}
			if s != test.strategy {
				t.Fatalf("ServerError() == %v, want %v", s, test.strategy)
			}
		})
	}
}

type testRoundTripperFunc func(*http.Request) (*http.Response, error)

func (t testRoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return t(r)
}

func TestGraphQLRoundTripper_InnerError(t *testing.T) {
	want := errors.New("inner error")
	rt := &graphQLRoundTripper{inner: testRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return nil, want
	})}
	r, err := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", strings.NewReader(""))
	if err != nil {
		t.Fatalf("NewRequest() failed with %v", err)
	}
	if _, err := rt.RoundTrip(r); !errors.Is(err, want) {
		t.Fatalf("RoundTrip() returned %v, want %v", err, want)
	}
}

func TestGraphQLRoundTripper_Non200Response(t *testing.T) {
	want := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader("")),
	}
	rt := &graphQLRoundTripper{inner: testRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return want, nil
	})}
	r, err := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", strings.NewReader(""))
	if err != nil {
		t.Fatalf("NewRequest() failed with %v", err)
	}
	if resp, err := rt.RoundTrip(r); err != nil {
		t.Fatalf("RoundTrip() returned %v, want no error", err)
	} else if resp != want {
		t.Fatalf("RoundTrip() = %v, want %v", resp, want)
	}
}

func TestGraphQLRoundTripper_BodyReadError(t *testing.T) {
	want := errors.New("read failed")
	rt := &graphQLRoundTripper{inner: testRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(iotest.ErrReader(want)),
		}, nil
	})}
	r, err := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", strings.NewReader(""))
	if err != nil {
		t.Fatalf("NewRequest() failed with %v", err)
	}
	if _, err := rt.RoundTrip(r); !errors.Is(err, want) {
		t.Fatalf("RoundTrip() returned %v, want %v", err, want)
	}
}

func TestGraphQLRoundTripper_JSONUnmarshalError(t *testing.T) {
	var want *json.SyntaxError
	rt := &graphQLRoundTripper{inner: testRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("{")),
		}, nil
	})}
	r, err := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", strings.NewReader(""))
	if err != nil {
		t.Fatalf("NewRequest() failed with %v", err)
	}
	if _, err := rt.RoundTrip(r); !errors.As(err, &want) {
		t.Fatalf("RoundTrip() returned %v, want %v", err, want)
	}
}

func TestGraphQLRoundTripper_NoErrors(t *testing.T) {
	rt := &graphQLRoundTripper{inner: testRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":{}}`)),
		}, nil
	})}
	r, err := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", strings.NewReader(""))
	if err != nil {
		t.Fatalf("NewRequest() failed with %v", err)
	}
	if _, err := rt.RoundTrip(r); err != nil {
		t.Fatalf("RoundTrip() returned %v, want no errors", err)
	}
}

func TestGraphQLRoundTripper_EmptyErrors(t *testing.T) {
	rt := &graphQLRoundTripper{inner: testRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":null, "errors":[]}`)),
		}, nil
	})}
	r, err := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", strings.NewReader(""))
	if err != nil {
		t.Fatalf("NewRequest() failed with %v", err)
	}
	if _, err := rt.RoundTrip(r); err != nil {
		t.Fatalf("RoundTrip() returned %v, want no errors", err)
	}
}

func TestGraphQLRoundTripper_OneError(t *testing.T) {
	var want *GraphQLErrors
	rt := &graphQLRoundTripper{inner: testRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":null, "errors":[{"message":"test", "type":"test"}]}`)),
		}, nil
	})}
	r, err := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", strings.NewReader(""))
	if err != nil {
		t.Fatalf("NewRequest() failed with %v", err)
	}
	if _, err := rt.RoundTrip(r); !errors.As(err, &want) {
		t.Fatalf("RoundTrip() returned %v, want %v", err, want)
	}
	if n := len(want.Errors()); n != 1 {
		t.Fatalf("len(Errors) = %d; want 1", n)
	}
}

func TestGraphQLRoundTripper_MultipleErrors(t *testing.T) {
	var want *GraphQLErrors
	rt := &graphQLRoundTripper{inner: testRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":null, "errors":[{"message":"test", "type":"test"}, {"message": "test2", "type":"test"}]}`)),
		}, nil
	})}
	r, err := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", strings.NewReader(""))
	if err != nil {
		t.Fatalf("NewRequest() failed with %v", err)
	}
	if _, err := rt.RoundTrip(r); !errors.As(err, &want) {
		t.Fatalf("RoundTrip() returned %v, want %v", err, want)
	}
	if n := len(want.Errors()); n != 2 {
		t.Fatalf("len(Errors) = %d; want 2", n)
	}
}

func TestGraphQLRoundTripper_OneErrorNotFound(t *testing.T) {
	want := ErrGraphQLNotFound
	rt := &graphQLRoundTripper{inner: testRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":null, "errors":[{"message":"test", "type":"NOT_FOUND"}]}`)),
		}, nil
	})}
	r, err := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", strings.NewReader(""))
	if err != nil {
		t.Fatalf("NewRequest() failed with %v", err)
	}
	if _, err := rt.RoundTrip(r); !errors.Is(err, want) {
		t.Fatalf("RoundTrip() returned %v, want %v", err, want)
	}
}

func TestGraphQLRoundTripper_MultipleErrors_OneNotFound(t *testing.T) {
	var want *GraphQLErrors
	rt := &graphQLRoundTripper{inner: testRoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":null, "errors":[{"message":"test", "type":"NOT_FOUND"}, {"message": "test2", "type":"test"}]}`)),
		}, nil
	})}
	r, err := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", strings.NewReader(""))
	if err != nil {
		t.Fatalf("NewRequest() failed with %v", err)
	}
	_, err = rt.RoundTrip(r)
	if !errors.As(err, &want) {
		t.Fatalf("RoundTrip() returned %v, want %v", err, want)
	}
	if errors.Is(err, ErrGraphQLNotFound) {
		t.Fatalf("RoundTrip() returned %v is %v", err, ErrGraphQLNotFound)
	}
	if !want.HasType(gitHubGraphQLNotFoundType) {
		t.Fatalf("HasType(%v) returned false, want true", gitHubGraphQLNotFoundType)
	}
}
