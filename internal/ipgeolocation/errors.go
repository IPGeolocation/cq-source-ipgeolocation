package ipgeolocation

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// APIError represents an error response from the IPGeolocation.io API.
type APIError struct {
	StatusCode int
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("ipgeolocation api error (status %d): %s", e.StatusCode, e.Message)
}

// IsRetryable returns true for transient HTTP errors that are worth retrying.
func (e *APIError) IsRetryable() bool {
	switch e.StatusCode {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

func (c *Client) redact(text string) string {
	return strings.NewReplacer(url.QueryEscape(c.apiKey), "[REDACTED]", c.apiKey, "[REDACTED]").Replace(text)
}

type requestError struct {
	message   string
	timeout   bool
	temporary bool
	canceled  bool
	deadline  bool
}

func (e *requestError) Error() string   { return e.message }
func (e *requestError) Timeout() bool   { return e.timeout }
func (e *requestError) Temporary() bool { return e.temporary }
func (e *requestError) Is(target error) bool {
	return (target == context.Canceled && e.canceled) || (target == context.DeadlineExceeded && e.deadline)
}

func (c *Client) redactError(err error) error {
	if err == nil {
		return nil
	}
	safe := &requestError{
		message:  c.redact(err.Error()),
		canceled: errors.Is(err, context.Canceled),
		deadline: errors.Is(err, context.DeadlineExceeded),
	}
	var networkError net.Error
	if errors.As(err, &networkError) {
		safe.timeout = networkError.Timeout()
		safe.temporary = networkError.Temporary()
	}
	if original, ok := err.(*url.Error); ok {
		safe.message = c.redact(original.Err.Error())
		return &url.Error{Op: c.redact(original.Op), URL: c.redact(original.URL), Err: safe}
	}
	return safe
}
