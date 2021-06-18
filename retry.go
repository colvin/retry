// Package retry implements a function retry loop with configurable bounds.
package retry

import (
	"context"
	"time"
)

// Worker is a function that does some work.
type Worker func() error

// Limiter is a function that is called after the Worker has failed. It should
// return true if further attempts should be made, false if no further attempts
// should be made. It is passed the Worker's error to aid in its deliberation.
type Limiter func(error) bool

// Timer is a function that is called after the Limiter has indicated that
// further attempts will be made. The next attempt will not be made until the
// Timer has completed.
type Timer func()

// Retry implements a retry loop for the given Worker function. Attempts are
// made in succession until the Worker returns without error or the Limiter
// terminates the loop. The Timer is called between each attempt.
func Retry(worker Worker, limiter Limiter, timer Timer) error {
	err := worker()
	for err != nil && limiter(err) {
		timer()
		err = worker()
	}
	return err
}

// CancelableLimiter returns a Limiter that wraps another Limiter, adding the
// ability to be canceled by a context before the interior Limiter is
// evaluated.
func CancelableLimiter(ctx context.Context, limiter Limiter) Limiter {
	return func(err error) bool {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		return limiter(err)
	}
}

// Once returns a Limiter that immediately terminates the loop. The Worker is
// always executed once before a Limiter is evaluated.
func Once() Limiter {
	return func(_ error) bool {
		return false
	}
}

// Forever returns a Limiter that never terminates the loop.
func Forever() Limiter {
	return func(_ error) bool {
		return true
	}
}

// Counts returns a Limiter that terminates the loop after the given number
// of attempts have been made. Zero is treated the same as one.
func Counts(max int) Limiter {
	// First attempt counts
	c := 1
	return func(_ error) bool {
		for c < max {
			c++
			return true
		}
		return false
	}
}

// UntilCanceled returns a Limiter that never terminates until it is canceled
// by a context.
func UntilCanceled(ctx context.Context) Limiter {
	return CancelableLimiter(ctx, Forever())
}

// CancelableSleep returns a Timer that sleeps for the given duration but may
// be canceled using a context.
func CancelableSleep(ctx context.Context, dur time.Duration) Timer {
	return func() {
		timer := time.NewTimer(dur)
		select {
		case <-timer.C:
			return
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

// MultiplicativeBackoff returns a Timer that sleeps for a duration, where the
// duration doubles each iteration until a ceiling is reached.
func MultiplicativeBackoff(base time.Duration, ceil time.Duration) Timer {
	dur := base
	return func() {
		time.Sleep(dur)
		if dur != ceil {
			dur = dur * 2
			if dur > ceil {
				dur = ceil
			}
		}
	}
}

// CMB is an alias for the admittedly long-named
// CancelableMultiplicativeBackoff.
var CMB = CancelableMultiplicativeBackoff

// CancelableMultiplicativeBackoff is the same as MultiplicativeBackoff but can
// be canceled using a context.
func CancelableMultiplicativeBackoff(ctx context.Context, base time.Duration, ceil time.Duration) Timer {
	dur := base
	return func() {
		timer := time.NewTimer(dur)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
		}
		if dur != ceil {
			dur = dur * 2
			if dur > ceil {
				dur = ceil
			}
		}
	}
}
