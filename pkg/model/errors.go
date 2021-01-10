package model

import (
	"errors"
	"fmt"
)

var (
	// ErrRetryable occurs when an error can be retried later
	ErrRetryable = errors.New("error is retryable")
)

// WrapRetryable wraps given error into retryable
func WrapRetryable(err error) error {
	return fmt.Errorf("%s: %w", err, ErrRetryable)
}
