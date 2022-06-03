package retry

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestInit_NotDone(t *testing.T) {
	req := NewRequest(&http.Request{}, func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader("")),
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
			Body:       ioutil.NopCloser(strings.NewReader("")),
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
			Body:       ioutil.NopCloser(strings.NewReader("")),
		}, nil
	}, MakeOptions())

	// This Do() is successful
	req.Do()

	// This Do() returns an error.
	resp, err := req.Do()
	if err != ErrorNoMoreAttempts {
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
					Body:       ioutil.NopCloser(strings.NewReader("")),
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
			Body:       ioutil.NopCloser(strings.NewReader("")),
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
			Body:       ioutil.NopCloser(strings.NewReader("")),
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
			Body:       ioutil.NopCloser(strings.NewReader("")),
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
