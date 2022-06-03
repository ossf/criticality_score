package retry

import (
	"errors"
	"net/http"
	"time"
)

const (
	DefaultMaxRetries      = 5
	DefaultInitialDuration = 2 * time.Minute
)

var (
	ErrorNoMoreAttempts = errors.New("request cannot by retried")
)

type RetryStrategy int

const (
	NoRetry RetryStrategy = iota
	RetryImmediate
	RetryWithInitialDelay
)

// String implements the Stringer interface.
func (s RetryStrategy) String() string {
	switch s {
	case NoRetry:
		return "NoRetry"
	case RetryImmediate:
		return "RetryImmediate"
	case RetryWithInitialDelay:
		return "RetryWithInitialDelay"
	default:
		panic("Invalid RetryStrategy")
	}
}

// A RetryStrategyFn is used to determine whether or not a retry is needed, and
// if so, whether an initial delay is needed or not.
type RetryStrategyFn func(*http.Response) (RetryStrategy, error)

// A RetryAfterFn is used to parse the delay from the Retry-After header.
//
// The returned time.Duration will be used as the delay.
//
// If the header is not set, or the value cannot be parsed, a Duration of 0
// must be returned.
type RetryAfterFn func(*http.Response) time.Duration

// A BackoffFn takes the previous delay and calculates a new, longer delay.
//
// This is used to extend the time between retries.
type BackoffFn func(time.Duration) time.Duration

// SleepFn must cause the current goroutine to sleep for the allocated Duration.
//
// The usual implementation of this is time.Sleep. This is provided for testing.
type sleepFn func(time.Duration)

// DefaultBackoff will double the duration d if it is greater than zero,
// otherwise it returns 1 minute.
func DefaultBackoff(d time.Duration) time.Duration {
	if d <= 0 {
		return time.Minute
	} else {
		return d * 2
	}
}

type Options struct {
	maxRetries         int
	initialDelay       time.Duration
	backoff            BackoffFn
	sleep              sleepFn
	retryAfter         RetryAfterFn
	retryStrategyFuncs []RetryStrategyFn
}
type Option interface {
	Apply(*Options)
}
type optionFn func(*Options)

func (o optionFn) Apply(opts *Options) {
	o(opts)
}

func MaxRetries(max int) Option {
	return optionFn(func(o *Options) {
		o.maxRetries = max
	})
}

func Backoff(bo BackoffFn) Option {
	return optionFn(func(o *Options) {
		o.backoff = bo
	})
}

func InitialDelay(d time.Duration) Option {
	return optionFn(func(o *Options) {
		o.initialDelay = d
	})
}

func RetryAfter(ra RetryAfterFn) Option {
	return optionFn(func(o *Options) {
		o.retryAfter = ra
	})
}

func Strategy(rs RetryStrategyFn) Option {
	return optionFn(func(o *Options) {
		o.retryStrategyFuncs = append(o.retryStrategyFuncs, rs)
	})
}

func MakeOptions(os ...Option) *Options {
	opts := &Options{
		maxRetries:   DefaultMaxRetries,
		initialDelay: DefaultInitialDuration,
		sleep:        time.Sleep,
		backoff:      DefaultBackoff,
	}
	for _, o := range os {
		o.Apply(opts)
	}
	return opts
}

type Request struct {
	client   func(*http.Request) (*http.Response, error)
	r        *http.Request
	o        *Options
	attempts int
	done     bool
	delay    time.Duration
}

func NewRequest(r *http.Request, client func(*http.Request) (*http.Response, error), o *Options) *Request {
	return &Request{
		client: client,
		r:      r,
		o:      o,
	}
}

func (r *Request) Done() bool {
	return r.done || r.attempts >= r.o.maxRetries
}

func (r *Request) onError(err error) (*http.Response, error) {
	r.done = true
	return nil, err
}

func (r *Request) onDone(resp *http.Response, err error) (*http.Response, error) {
	r.done = true
	return resp, err
}

func (r *Request) Do() (*http.Response, error) {
	if r.Done() {
		// Already done, so don't attempt any more.
		return nil, ErrorNoMoreAttempts
	}
	if r.attempts > 0 {
		// This is a retry!
		if r.delay > 0 {
			// Wait if we have a delay
			r.o.sleep(r.delay)
		}
		// Update the delay
		r.delay = r.o.backoff(r.delay)
	}
	// Bump the number of attempts
	r.attempts++

	// Issue the request
	resp, err := r.client(r.r)
	if err != nil {
		// Return the response and the error if the client returned an error
		// TODO: pass err to the strategy funcs.
		return r.onDone(resp, err)
	}

	// Immediate success if the request is 2xx or 3xx
	if http.StatusOK <= resp.StatusCode && resp.StatusCode < http.StatusBadRequest {
		return r.onDone(resp, err)
	}

	if r.o.retryAfter != nil {
		// Check if the Retry-After header is set
		d := r.o.retryAfter(resp)
		if d != 0 {
			// We have the Retry-After header so set the delay and return
			r.delay = d
			return resp, err
		}
	}

	// Check each of the retryStrategyFuncs to see if we should retry
	s := NoRetry
	var errS error
	for _, fn := range r.o.retryStrategyFuncs {
		s, errS = fn(resp)
		if errS != nil {
			return r.onError(errS)
		}
		if s != NoRetry {
			// We found a strategy that required a retry
			// TODO: we may want to find the retry needing the longest delay
			break
		}
	}
	if s == NoRetry {
		return r.onDone(resp, err)
	}
	// Only apply the strategy on the first iteration
	if r.attempts == 1 && s == RetryWithInitialDelay {
		r.delay = r.o.initialDelay
	}

	return resp, err
}

func NewRoundTripper(inner http.RoundTripper, o ...Option) http.RoundTripper {
	return &roundTripper{
		inner: inner,
		o:     MakeOptions(o...),
	}
}

type roundTripper struct {
	inner http.RoundTripper
	o     *Options
}

func (rt *roundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	rr := NewRequest(r, rt.inner.RoundTrip, rt.o)
	for !rr.Done() {
		resp, err = rr.Do()
	}
	return resp, err
}
