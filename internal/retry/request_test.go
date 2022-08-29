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

package retry

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestInit_NotDone(t *testing.T) {
	req := NewRequest(&http.Request{}, func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}, MakeOptions())

	if req.Done() {
		t.Fatalf("Done() == true; want false")
	}
}

func Test200NoRetryStrategyFunc(t *testing.T) {
	req := NewRequest(&http.Request{}, func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}, MakeOptions())
	resp, err := req.Do()

	if !req.Done() {
		t.Fatalf("Done() == false; want true")
	}
	if err != nil {
		t.Fatalf("Do() returned err %v; want no err", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Do() return response with status == %d; want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestDoAfterDoneReturnsError(t *testing.T) {
	req := NewRequest(&http.Request{}, func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}, MakeOptions())

	// This Do() is successful
	req.Do()

	// This Do() returns an error.
	resp, err := req.Do()
	if !errors.Is(err, ErrorNoMoreAttempts) {
		t.Fatalf("Do() returned err %v; want %v", err, ErrorNoMoreAttempts)
	}
	if resp != nil {
		t.Fatalf("Do() returned a response; want nil")
	}
}

func Test2xx3xxAlwaysDone(t *testing.T) {
	for code := http.StatusOK; code < http.StatusBadRequest; code++ {
		t.Run(fmt.Sprintf("status %d", code), func(t *testing.T) {
			req := NewRequest(&http.Request{}, func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: code,
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			}, MakeOptions(Strategy(func(_ *http.Response) (RetryStrategy, error) {
				// Always force a retry if the strategy is called.
				return RetryImmediate, nil
			})))
			req.Do()
			if !req.Done() {
				t.Fatalf("Done() == false; want true")
			}
		})
	}
}

func TestRetryAfterZeroDuration(t *testing.T) {
	req := NewRequest(&http.Request{}, func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}, MakeOptions(RetryAfter(func(_ *http.Response) time.Duration {
		return 0
	})))
	req.Do()
	if !req.Done() {
		t.Fatalf("Done() == false; want true")
	}
}

func TestRetryAfterDuration(t *testing.T) {
	slept := time.Duration(0)
	opts := MakeOptions(RetryAfter(func(_ *http.Response) time.Duration {
		return time.Minute
	}))
	opts.sleep = func(d time.Duration) {
		slept += d
	}
	req := NewRequest(&http.Request{}, func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}, opts)
	req.Do()
	if req.Done() {
		t.Fatalf("Done() == true; want false")
	}

	req.Do()
	if req.Done() {
		t.Fatalf("Done() == true; want false")
	}
	if slept != time.Minute {
		t.Fatalf("Only slept for %d; want sleep for %d", slept, time.Minute)
	}
}

func TestZeroMaxRetriesOnlyTriesOnce(t *testing.T) {
	req := NewRequest(&http.Request{}, func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}, MakeOptions(MaxRetries(0), RetryAfter(func(_ *http.Response) time.Duration {
		// Force a retry
		return time.Minute
	})))
	req.Do()
	if !req.Done() {
		t.Fatalf("Done() == false; want true")
	}
}
