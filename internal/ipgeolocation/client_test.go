package ipgeolocation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	c, err := NewClient("test-key",
		WithEndpoint(server.URL),
		WithHTTPClient(server.Client()),
		WithRetries(1),
		WithRateLimit(1000),
	)
	require.NoError(t, err)
	return c, server
}

func TestNewClientRequiresAPIKey(t *testing.T) {
	_, err := NewClient("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api key is required")
}

func TestGetIPGeolocationSuccess(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/ipgeo", r.URL.Path)
		assert.Equal(t, "test-key", r.URL.Query().Get("apiKey"))
		assert.Equal(t, "1.1.1.1", r.URL.Query().Get("ip"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip": "1.1.1.1",
			"location": map[string]interface{}{
				"country_name": "Australia",
				"city":         "Sydney",
				"latitude":     "-33.86820",
				"longitude":    "151.20270",
			},
		})
	})

	c, server := newTestClient(t, handler)
	defer server.Close()

	geo, err := c.GetIPGeolocation(context.Background(), "1.1.1.1")
	require.NoError(t, err)
	assert.Equal(t, "1.1.1.1", geo.IP)
	assert.Equal(t, "Australia", geo.Location.CountryName)
	assert.Equal(t, "Sydney", geo.Location.City)
}

func TestGetIPGeolocationRetries(t *testing.T) {
	var attempts int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n <= 1 {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": "server error"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ip": "1.1.1.1"})
	})

	c, server := newTestClient(t, handler)
	defer server.Close()

	geo, err := c.GetIPGeolocation(context.Background(), "1.1.1.1")
	require.NoError(t, err)
	assert.Equal(t, "1.1.1.1", geo.IP)
	assert.Equal(t, int32(2), atomic.LoadInt32(&attempts), "should have retried once")
}

func TestGetIPGeolocationNonRetryableError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Invalid API key"})
	})

	c, server := newTestClient(t, handler)
	defer server.Close()

	_, err := c.GetIPGeolocation(context.Background(), "1.1.1.1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid API key")
}

func TestGetTimezoneSuccess(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/timezone", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"timezone": "Asia/Tokyo",
			"time_zone": map[string]interface{}{
				"name":   "Asia/Tokyo",
				"offset": 9.0,
			},
		})
	})

	c, server := newTestClient(t, handler)
	defer server.Close()

	tz, err := c.GetTimezone(context.Background(), "8.8.8.8")
	require.NoError(t, err)
	assert.Equal(t, "Asia/Tokyo", tz.TimeZone.Name)
	assert.Equal(t, 9.0, tz.TimeZone.Offset)
}

func TestGetAstronomySuccess(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/astronomy", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"date":    "2026-01-15",
			"sunrise": "07:20",
			"sunset":  "16:52",
		})
	})

	c, server := newTestClient(t, handler)
	defer server.Close()

	astro, err := c.GetAstronomy(context.Background(), "")
	require.NoError(t, err)
	assert.Equal(t, "2026-01-15", astro.Date)
	assert.Equal(t, "07:20", astro.Sunrise)
}

func TestGetUserAgentSuccess(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/user-agent", r.URL.Path)
		assert.NotEmpty(t, r.URL.Query().Get("ua"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_agent_string": "test-ua",
			"name":              "TestBrowser",
			"type":              "Browser",
		})
	})

	c, server := newTestClient(t, handler)
	defer server.Close()

	ua, err := c.GetUserAgent(context.Background(), "test-ua")
	require.NoError(t, err)
	assert.Equal(t, "TestBrowser", ua.Name)
	assert.Equal(t, "Browser", ua.Type)
}

func TestContextCancellation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow server; the context should be cancelled before we respond.
		<-r.Context().Done()
	})

	c, server := newTestClient(t, handler)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := c.GetIPGeolocation(ctx, "1.1.1.1")
	require.Error(t, err)
}

func TestAPIErrorRetryable(t *testing.T) {
	tests := []struct {
		code      int
		retryable bool
	}{
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
	}

	for _, tt := range tests {
		e := &APIError{StatusCode: tt.code}
		assert.Equal(t, tt.retryable, e.IsRetryable(), "status %d", tt.code)
	}
}
