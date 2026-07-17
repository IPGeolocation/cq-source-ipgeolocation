package services

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/IPGeolocation/cq-source-ipgeolocation/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchIPSecurity(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/security", r.URL.Path)
		assert.Equal(t, "2.56.188.34", r.URL.Query().Get("ip"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip": "2.56.188.34",
			"security": map[string]interface{}{
				"threat_score":           80,
				"is_tor":                 false,
				"is_proxy":               true,
				"proxy_provider_names":   []string{"Zyte Proxy"},
				"proxy_confidence_score": 80,
				"proxy_last_seen":        "2025-12-12",
				"is_residential_proxy":   true,
				"is_vpn":                 true,
				"vpn_provider_names":     []string{"Nord VPN"},
				"vpn_confidence_score":   80,
				"vpn_last_seen":          "2026-01-19",
				"is_relay":               false,
				"relay_provider_name":    "",
				"is_anonymous":           true,
				"is_known_attacker":      true,
				"is_bot":                 false,
				"is_spam":                false,
				"is_cloud_provider":      true,
				"cloud_provider_name":    "Packethub S.A.",
			},
		})
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()
	c.Spec.IPs = []string{"2.56.188.34"}

	results := make(chan any, 10)
	err := fetchIPSecurity(context.Background(), c, nil, results)
	require.NoError(t, err)
	close(results)

	var collected []*IPSecurityFlat
	for item := range results {
		flat, ok := item.(*IPSecurityFlat)
		require.True(t, ok)
		collected = append(collected, flat)
	}

	require.Len(t, collected, 1)
	sec := collected[0]
	assert.Equal(t, "2.56.188.34", sec.IP)
	assert.Equal(t, 80, sec.ThreatScore)
	assert.True(t, sec.IsProxy)
	assert.True(t, sec.IsVPN)
	assert.True(t, sec.IsAnonymous)
	assert.True(t, sec.IsKnownAttacker)
	assert.False(t, sec.IsTor)
	assert.False(t, sec.IsBot)
	assert.True(t, sec.IsCloudProvider)
	assert.Equal(t, "Packethub S.A.", sec.CloudProviderName)
	assert.Equal(t, "Zyte Proxy", sec.ProxyProviderNames)
	assert.Equal(t, "Nord VPN", sec.VPNProviderNames)
}

func TestFetchIPSecurityCallerIP(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("ip"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip": "203.0.113.50",
			"security": map[string]interface{}{
				"threat_score": 0,
				"is_tor":       false,
				"is_proxy":     false,
				"is_vpn":       false,
				"is_anonymous": false,
			},
		})
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()
	c.Spec.IPs = nil

	results := make(chan any, 10)
	err := fetchIPSecurity(context.Background(), c, nil, results)
	require.NoError(t, err)
	close(results)

	var collected []*IPSecurityFlat
	for item := range results {
		collected = append(collected, item.(*IPSecurityFlat))
	}
	require.Len(t, collected, 1)
	assert.Equal(t, "203.0.113.50", collected[0].IP, "should use IP from API response")
}

func TestFetchIPSecurityMultipleIPs(t *testing.T) {
	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		ip := r.URL.Query().Get("ip")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip":       ip,
			"security": map[string]interface{}{"threat_score": callCount * 10},
		})
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()
	c.Spec.IPs = []string{"1.1.1.1", "8.8.8.8", "9.9.9.9"}

	results := make(chan any, 10)
	err := fetchIPSecurity(context.Background(), c, nil, results)
	require.NoError(t, err)
	close(results)

	count := 0
	for range results {
		count++
	}
	assert.Equal(t, 3, count)
	assert.Equal(t, 3, callCount)
}
