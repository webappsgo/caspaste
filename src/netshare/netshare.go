
// This file is part of CasPaste.

// CasPaste is free software released under the MIT License.
// See LICENSE.md file for details.

package netshare

import (
	"errors"
)

const (
	// Max length for paste author name, email and URL
	MaxLengthAuthorAll = 100
)

var (
	// HTTP 400
	ErrBadRequest = errors.New("bad request")
	// HTTP 401
	ErrUnauthorized = errors.New("unauthorized")
	// HTTP 404
	ErrNotFound = errors.New("not found")
	// HTTP 405
	ErrMethodNotAllowed = errors.New("method not allowed")
	// HTTP 413
	ErrPayloadTooLarge = errors.New("payload too large")
	// HTTP 429
	ErrTooManyRequests = errors.New("too many requests")
	// HTTP 500
	ErrInternal = errors.New("internal server error")
)

type RateLimitError struct {
	s          string
	RetryAfter int64
}

func (e *RateLimitError) Error() string {
	return e.s
}

func ErrTooManyRequestsNew(retryAfter int64) *RateLimitError {
	return &RateLimitError{
		s:          "Too Many Requests",
		RetryAfter: retryAfter,
	}
}
