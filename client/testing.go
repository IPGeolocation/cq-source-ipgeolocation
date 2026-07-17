package client

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropic/cq-source-ipgeolocation/internal/ipgeolocation"
	"github.com/rs/zerolog"
)

// TestClient creates a Client wired to a local httptest.Server for unit testing.
// The caller provides a handler that simulates the IPGeolocation.io API.
// It returns the Client and a cleanup function that must be called when done.
func TestClient(t *testing.T, handler http.Handler) (*Client, func()) {
	t.Helper()

	server := httptest.NewServer(handler)

	spec := Spec{
		APIKey:        "test-api-key",
		Endpoint:      server.URL,
		Timeout:       "10s",
		RetryAttempts: 1,
		Concurrency:   2,
		RateLimit:     100,
		UserAgent:     "cq-test",
	}

	ipgeo, err := ipgeolocation.NewClient(
		spec.APIKey,
		ipgeolocation.WithEndpoint(spec.Endpoint),
		ipgeolocation.WithTimeout(spec.TimeoutDuration()),
		ipgeolocation.WithHTTPClient(server.Client()),
		ipgeolocation.WithUserAgent(spec.UserAgent),
		ipgeolocation.WithRateLimit(spec.RateLimit),
	)
	if err != nil {
		t.Fatalf("failed to create test ipgeolocation client: %v", err)
	}

	logger := zerolog.Nop()

	c := New(logger, spec, ipgeo, nil)
	return c, server.Close
}
