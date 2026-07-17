package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecSetDefaults(t *testing.T) {
	s := Spec{}
	s.SetDefaults()

	assert.Equal(t, defaultEndpoint, s.Endpoint)
	assert.Equal(t, defaultTimeout.String(), s.Timeout)
	assert.Equal(t, defaultRetryAttempts, s.RetryAttempts)
	assert.Equal(t, defaultConcurrency, s.Concurrency)
	assert.Equal(t, defaultRateLimit, s.RateLimit)
	assert.Equal(t, defaultUserAgent, s.UserAgent)
}

func TestSpecSetDefaultsPreservesValues(t *testing.T) {
	s := Spec{
		Endpoint:      "https://custom.api.example.com",
		Timeout:       "60s",
		RetryAttempts: 5,
		Concurrency:   20,
		RateLimit:     50,
		UserAgent:     "my-custom-agent",
	}
	s.SetDefaults()

	assert.Equal(t, "https://custom.api.example.com", s.Endpoint)
	assert.Equal(t, "60s", s.Timeout)
	assert.Equal(t, 5, s.RetryAttempts)
	assert.Equal(t, 20, s.Concurrency)
	assert.Equal(t, 50, s.RateLimit)
	assert.Equal(t, "my-custom-agent", s.UserAgent)
}

func TestSpecValidateSuccess(t *testing.T) {
	s := Spec{APIKey: "test-key"}
	s.SetDefaults()
	err := s.Validate()
	require.NoError(t, err)
}

func TestSpecValidateMissingAPIKey(t *testing.T) {
	s := Spec{}
	s.SetDefaults()
	err := s.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_key is required")
}

func TestSpecValidateInvalidTimeout(t *testing.T) {
	s := Spec{APIKey: "key", Timeout: "not-a-duration"}
	err := s.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timeout")
}

func TestSpecValidateNegativeRetries(t *testing.T) {
	s := Spec{APIKey: "key", RetryAttempts: -1}
	s.SetDefaults()
	s.RetryAttempts = -1
	err := s.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retry_attempts must be >= 0")
}

func TestSpecValidateZeroConcurrency(t *testing.T) {
	s := Spec{APIKey: "key", Concurrency: -1}
	s.SetDefaults()
	s.Concurrency = -1
	err := s.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "concurrency must be >= 1")
}

func TestSpecValidateZeroRateLimit(t *testing.T) {
	s := Spec{APIKey: "key", RateLimit: -1}
	s.SetDefaults()
	s.RateLimit = -1
	err := s.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rate_limit must be >= 1")
}

func TestSpecTimeoutDuration(t *testing.T) {
	s := Spec{Timeout: "15s"}
	d := s.TimeoutDuration()
	assert.Equal(t, 15*time.Second, d)
}

func TestSpecTimeoutDurationInvalidFallback(t *testing.T) {
	s := Spec{Timeout: "garbage"}
	d := s.TimeoutDuration()
	assert.Equal(t, defaultTimeout, d)
}
