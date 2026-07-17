package client

import (
	"fmt"
	"time"
)

const (
	defaultEndpoint      = "https://api.ipgeolocation.io"
	defaultTimeout       = 30 * time.Second
	defaultRetryAttempts = 3
	defaultConcurrency   = 10
	defaultRateLimit     = 15 // requests per second
	defaultUserAgent     = "cq-source-ipgeolocation/1.0"
)

// Spec holds the configuration for the IPGeolocation source plugin.
// It is parsed from the user's YAML configuration.
type Spec struct {
	// APIKey is the IPGeolocation.io API key used for authentication.
	APIKey string `json:"api_key"`

	// Endpoint is the base URL for the IPGeolocation.io API.
	// Defaults to "https://api.ipgeolocation.io".
	Endpoint string `json:"endpoint"`

	// Timeout is the HTTP client timeout as a duration string (e.g. "30s").
	// Defaults to "30s".
	Timeout string `json:"timeout"`

	// RetryAttempts is the number of times to retry a failed API request.
	// Defaults to 3.
	RetryAttempts int `json:"retry_attempts"`

	// Concurrency controls how many tables are synced in parallel.
	// Defaults to 10.
	Concurrency int `json:"concurrency"`

	// RateLimit is the maximum number of API requests per second.
	// Defaults to 15.
	RateLimit int `json:"rate_limit"`

	// UserAgent is the User-Agent header sent with every request.
	// Defaults to "cq-source-ipgeolocation/1.0".
	UserAgent string `json:"user_agent"`

	// IPs is a list of IP addresses to look up using the IP Geolocation endpoint.
	// If empty, the plugin looks up the caller's own public IP.
	IPs []string `json:"ips"`

	// UserAgents is a list of user-agent strings to parse via the User-Agent API.
	UserAgents []string `json:"user_agents"`

	// ASNs is a list of AS numbers (e.g. "24940", "15169") to look up via the dedicated ASN API.
	ASNs []string `json:"asns"`

	// IncludeSecurity adds IP security/threat data to geolocation results (paid plans).
	IncludeSecurity bool `json:"include_security"`

	// IncludeAbuse adds abuse contact data to geolocation results (paid plans).
	IncludeAbuse bool `json:"include_abuse"`
}

// SetDefaults populates zero-valued fields with sensible defaults.
func (s *Spec) SetDefaults() {
	if s.Endpoint == "" {
		s.Endpoint = defaultEndpoint
	}
	if s.Timeout == "" {
		s.Timeout = defaultTimeout.String()
	}
	if s.RetryAttempts == 0 {
		s.RetryAttempts = defaultRetryAttempts
	}
	if s.Concurrency == 0 {
		s.Concurrency = defaultConcurrency
	}
	if s.RateLimit == 0 {
		s.RateLimit = defaultRateLimit
	}
	if s.UserAgent == "" {
		s.UserAgent = defaultUserAgent
	}
}

// Validate checks required fields and returns a descriptive error if any are missing or invalid.
func (s *Spec) Validate() error {
	if s.APIKey == "" {
		return fmt.Errorf("api_key is required: get one from https://app.ipgeolocation.io/signup")
	}
	if _, err := time.ParseDuration(s.Timeout); err != nil {
		return fmt.Errorf("invalid timeout %q: %w", s.Timeout, err)
	}
	if s.RetryAttempts < 0 {
		return fmt.Errorf("retry_attempts must be >= 0, got %d", s.RetryAttempts)
	}
	if s.Concurrency < 1 {
		return fmt.Errorf("concurrency must be >= 1, got %d", s.Concurrency)
	}
	if s.RateLimit < 1 {
		return fmt.Errorf("rate_limit must be >= 1, got %d", s.RateLimit)
	}
	return nil
}

// TimeoutDuration parses and returns the configured timeout.
func (s *Spec) TimeoutDuration() time.Duration {
	d, err := time.ParseDuration(s.Timeout)
	if err != nil {
		return defaultTimeout
	}
	return d
}
