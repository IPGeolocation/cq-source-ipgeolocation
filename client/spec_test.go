package client

import (
	"encoding/json"
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

func TestSpecASNIncludeDefaultsToNothing(t *testing.T) {
	s := Spec{APIKey: "k"}
	s.SetDefaults()
	require.NoError(t, s.Validate())

	mods, err := s.ASNIncludeModules()
	require.NoError(t, err)
	assert.Empty(t, mods, "heavy ASN modules must be opt-in")
}

func TestSpecASNIncludeNormalisation(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"single", []string{"peers"}, []string{"peers"}},
		{"canonical order", []string{"routes", "downstreams", "peers"}, []string{"peers", "downstreams", "routes"}},
		{"case and whitespace", []string{" Peers ", "UPSTREAMS"}, []string{"peers", "upstreams"}},
		{"duplicates", []string{"peers", "peers", "Peers"}, []string{"peers"}},
		{"blank entries ignored", []string{"", "  ", "routes"}, []string{"routes"}},
		{"wildcard", []string{"*"}, []string{"peers", "upstreams", "downstreams", "routes", "whois_response"}},
		{"wildcard plus explicit", []string{"routes", "*"}, []string{"peers", "upstreams", "downstreams", "routes", "whois_response"}},
		{"whois", []string{"whois_response"}, []string{"whois_response"}},
		{"empty list", []string{}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Spec{ASNInclude: tt.in}
			got, err := s.ASNIncludeModules()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSpecASNIncludeInvalidValue(t *testing.T) {
	s := Spec{APIKey: "k", ASNInclude: []string{"peers", "neighbours"}}
	s.SetDefaults()

	err := s.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "asn_include")
	assert.Contains(t, err.Error(), "neighbours")
	// The message tells the user what is allowed.
	for _, valid := range []string{"peers", "upstreams", "downstreams", "routes", "whois_response"} {
		assert.Contains(t, err.Error(), valid)
	}
}

func TestSpecASNIncludeParsesFromJSON(t *testing.T) {
	var s Spec
	require.NoError(t, json.Unmarshal([]byte(`{"api_key":"k","asn_include":["peers","routes"]}`), &s))
	s.SetDefaults()
	require.NoError(t, s.Validate())
	mods, err := s.ASNIncludeModules()
	require.NoError(t, err)
	assert.Equal(t, []string{"peers", "routes"}, mods)
}

func TestSpecIPGeolocationIncludes(t *testing.T) {
	tests := []struct {
		name string
		spec Spec
		want []string
	}{
		{"nothing enabled", Spec{}, nil},
		{"security only", Spec{IncludeSecurity: true}, []string{"security"}},
		{"geo accuracy only", Spec{IncludeGeoAccuracy: true}, []string{"geo_accuracy"}},
		{"dma code only", Spec{IncludeDMACode: true}, []string{"dma_code"}},
		{"hostname default mode", Spec{IncludeHostname: true}, []string{"hostnameFallbackLive"}},
		{"hostname database", Spec{IncludeHostname: true, HostnameLookup: "database"}, []string{"hostname"}},
		{"hostname live", Spec{IncludeHostname: true, HostnameLookup: "live"}, []string{"liveHostname"}},
		{"hostname fallback_live, case-insensitive", Spec{IncludeHostname: true, HostnameLookup: " Fallback_Live "}, []string{"hostnameFallbackLive"}},
		{"lookup mode ignored when hostname disabled", Spec{HostnameLookup: "live"}, nil},
		{
			"everything, stable order",
			Spec{IncludeHostname: true, IncludeDMACode: true, IncludeGeoAccuracy: true, IncludeAbuse: true, IncludeSecurity: true, HostnameLookup: "live"},
			[]string{"security", "abuse", "geo_accuracy", "dma_code", "liveHostname"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.spec.IPGeolocationIncludes()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSpecHostnameLookupInvalid(t *testing.T) {
	// Rejected even when include_hostname is false, so typos surface early.
	s := Spec{APIKey: "k", HostnameLookup: "dns"}
	s.SetDefaults()
	err := s.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hostname_lookup")
	for _, valid := range []string{"database", "live", "fallback_live"} {
		assert.Contains(t, err.Error(), valid)
	}

	_, err = s.IPGeolocationIncludes()
	require.Error(t, err)
}

func TestSpecGeoAccuracyHostnameParseFromJSON(t *testing.T) {
	var s Spec
	require.NoError(t, json.Unmarshal([]byte(`{"api_key":"k","include_geo_accuracy":true,"include_hostname":true,"hostname_lookup":"database"}`), &s))
	s.SetDefaults()
	require.NoError(t, s.Validate())
	got, err := s.IPGeolocationIncludes()
	require.NoError(t, err)
	assert.Equal(t, []string{"geo_accuracy", "hostname"}, got)
}

func TestSpecGeoAccuracyHostnameOffByDefault(t *testing.T) {
	s := Spec{APIKey: "k"}
	s.SetDefaults()
	require.NoError(t, s.Validate())
	got, err := s.IPGeolocationIncludes()
	require.NoError(t, err)
	assert.Empty(t, got)
}
