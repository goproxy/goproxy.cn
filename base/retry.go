package base

import (
	"context"
	"errors"
	"time"
)

// ErrRetryable means an operation is retryable.
var ErrRetryable = errors.New("retryable")

// RetryableError is an error indicating that an operation is retryable.
type RetryableError string

// Error implements the `error`.
func (re RetryableError) Error() string {
	return string(re)
}

// Is reports whether the target is `ErrRetryable`.
func (RetryableError) Is(target error) bool {
	return target == ErrRetryable
}

// Retry is like the `RetryN`, but at most 100 million retries.
func Retry(
	ctx context.Context,
	f func(ctx context.Context) error,
	retryable func(err error) bool,
	nap time.Duration,
) error {
	return RetryN(ctx, f, retryable, nap, 100_000_000)
}

// RetryN retries the f based on the retryable at most n times. And there is a
// nap before each retry.
func RetryN(
	ctx context.Context,
	f func(ctx context.Context) error,
	retryable func(err error) bool,
	nap time.Duration,
	n int,
) error {
	if retryable == nil {
		retryable = func(_ error) bool { return false }
	}
	n = max(n, 1)

	var err error
	for range n {
		if err = f(ctx); err == nil {
			return nil
		} else if !errors.Is(err, ErrRetryable) && !retryable(err) {
			return err
		}

		select {
		case <-ctx.Done():
			return err
		case <-time.After(nap):
		}
	}
	return err
}
