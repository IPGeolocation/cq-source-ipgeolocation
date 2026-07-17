package ipgeolocation

import "fmt"

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
