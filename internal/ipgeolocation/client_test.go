package ipgeolocation

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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
		// Official API: POST /v3/user-agent with a JSON body {"uaString": "..."}.
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v3/user-agent", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-key", r.URL.Query().Get("apiKey"))
		var body map[string]string
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "test-ua", body["uaString"])
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

func TestGetASNIncludeParameter(t *testing.T) {
	var got url.Values
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/asn", r.URL.Path)
		got = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"asn": map[string]interface{}{
				"as_number":      "AS12",
				"whois_response": "ASNumber: 12",
				"peers":          []map[string]string{{"as_number": "AS3356", "description": "Level 3", "country": "US"}},
			},
		})
	})
	c, server := newTestClient(t, handler)
	defer server.Close()

	// No modules requested: no include parameter, by number and by IP.
	_, err := c.GetASNByNumber(context.Background(), "12")
	require.NoError(t, err)
	assert.False(t, got.Has("include"))
	assert.Equal(t, "12", got.Get("asn"))

	_, err = c.GetASNByIP(context.Background(), "8.8.8.8")
	require.NoError(t, err)
	assert.False(t, got.Has("include"))
	assert.Equal(t, "8.8.8.8", got.Get("ip"))

	// Modules requested: sent comma-separated, and the response is decoded.
	resp, err := c.GetASNByNumber(context.Background(), "12", ASNIncludePeers, ASNIncludeWhoisResponse)
	require.NoError(t, err)
	assert.Equal(t, "peers,whois_response", got.Get("include"))
	assert.Equal(t, "ASNumber: 12", resp.ASN.WhoisResponse)
	require.Len(t, resp.ASN.Peers, 1)
	assert.Equal(t, "AS3356", resp.ASN.Peers[0].ASNumber)

	_, err = c.GetASNByIP(context.Background(), "", ASNIncludeRoutes)
	require.NoError(t, err)
	assert.Equal(t, "routes", got.Get("include"))
	assert.False(t, got.Has("ip"), "empty ip means caller IP: no ip param")
}

func TestGetSecurityDecodesAllFields(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ip":"87.58.66.106","security":{
			"threat_score":5,"is_bot":true,"bot_confidence_score":95,"bot_operator_name":"ChatGPT",
			"bot_type":"ai_crawler","is_known_good_bot":true,"bot_last_seen":"2026-09-03",
			"is_cloud_provider":true,"cloud_provider_name":"Zscaler Switzerland GmbH",
			"is_corporate_gateway":true,"corporate_gateway_type":"secure_web_gateway",
			"corporate_gateway_provider_name":"Zscaler"}}`))
	})
	c, server := newTestClient(t, handler)
	defer server.Close()

	resp, err := c.GetSecurity(context.Background(), "87.58.66.106")
	require.NoError(t, err)
	s := resp.Security
	require.NotNil(t, s)
	assert.Equal(t, 95, s.BotConfidenceScore)
	assert.Equal(t, "ChatGPT", s.BotOperatorName)
	assert.Equal(t, "ai_crawler", s.BotType)
	assert.True(t, s.IsKnownGoodBot)
	assert.Equal(t, "2026-09-03", s.BotLastSeen)
	assert.True(t, s.IsCorporateGateway)
	assert.Equal(t, "secure_web_gateway", s.CorporateGatewayType)
	assert.Equal(t, "Zscaler", s.CorporateGatewayProviderName)
}

func TestGetIPGeolocationDecodesNetworkCDN(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ip":"104.16.0.1","network":{"route":"104.16.0.0/13","is_cdn":true,"cdn_provider_name":"Cloudflare"}}`))
	})
	c, server := newTestClient(t, handler)
	defer server.Close()

	geo, err := c.GetIPGeolocation(context.Background(), "104.16.0.1")
	require.NoError(t, err)
	assert.True(t, geo.Network.IsCDN)
	assert.Equal(t, "Cloudflare", geo.Network.CDNProviderName)
}

// A retried POST must re-send its body: the reader is consumed by the first attempt.
func TestPostBodyIsResentOnRetry(t *testing.T) {
	var bodies []string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(raw))
		w.Header().Set("Content-Type", "application/json")
		if len(bodies) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable) // retryable
			w.Write([]byte(`{"message":"try again"}`))
			return
		}
		w.Write([]byte(`{"user_agent_string":"retry-ua","name":"Chrome"}`))
	})
	c, server := newTestClient(t, handler) // WithRetries(1)
	defer server.Close()

	ua, err := c.GetUserAgent(context.Background(), "retry-ua")
	require.NoError(t, err)
	assert.Equal(t, "Chrome", ua.Name)

	require.Len(t, bodies, 2, "expected one failed attempt and one retry")
	assert.JSONEq(t, `{"uaString":"retry-ua"}`, bodies[0])
	assert.JSONEq(t, `{"uaString":"retry-ua"}`, bodies[1], "retry must carry the same body")
}
