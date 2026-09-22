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

// securityFixture returns a response using the values from the official
// IP Security API docs (145.223.7.7 example) with every field populated.
func securityFixture() map[string]interface{} {
	return map[string]interface{}{
		"threat_score":                    90,
		"is_tor":                          false,
		"is_proxy":                        true,
		"proxy_provider_names":            []string{"NetNut", "ProxyScrape", "Oxy Labs", "DataImpulse"},
		"proxy_confidence_score":          99,
		"proxy_last_seen":                 "2026-09-01",
		"is_residential_proxy":            true,
		"is_vpn":                          true,
		"vpn_provider_names":              []string{"SurfShark VPN", "Ishaan VPN"},
		"vpn_confidence_score":            98,
		"vpn_last_seen":                   "2026-07-31",
		"is_relay":                        true,
		"relay_provider_name":             "Example Relay",
		"is_anonymous":                    true,
		"is_known_attacker":               true,
		"is_bot":                          true,
		"bot_confidence_score":            95,
		"bot_operator_name":               "ChatGPT",
		"bot_type":                        "ai_crawler",
		"is_known_good_bot":               true,
		"bot_last_seen":                   "2026-09-03",
		"is_spam":                         true,
		"is_cloud_provider":               true,
		"cloud_provider_name":             "Brander Group Inc.",
		"is_corporate_gateway":            true,
		"corporate_gateway_type":          "secure_web_gateway",
		"corporate_gateway_provider_name": "Zscaler",
	}
}

func TestFetchIPSecurityAllFields(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ip": "145.223.7.7", "security": securityFixture()})
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()
	c.Spec.IPs = []string{"145.223.7.7"}

	results := make(chan any, 10)
	require.NoError(t, fetchIPSecurity(context.Background(), c, nil, results))
	close(results)

	var got []*IPSecurityFlat
	for item := range results {
		got = append(got, item.(*IPSecurityFlat))
	}
	require.Len(t, got, 1)

	// Resolve through the real table so we assert on actual column values.
	tbl := buildTable(t, IPSecurityTable())
	row := resolveRow(t, tbl, got[0])

	want := map[string]any{
		"ip":                              "145.223.7.7",
		"threat_score":                    int64(90),
		"is_tor":                          false,
		"is_proxy":                        true,
		"proxy_provider_names":            "NetNut,ProxyScrape,Oxy Labs,DataImpulse",
		"proxy_confidence_score":          int64(99),
		"proxy_last_seen":                 "2026-09-01",
		"is_residential_proxy":            true,
		"is_vpn":                          true,
		"vpn_provider_names":              "SurfShark VPN,Ishaan VPN",
		"vpn_confidence_score":            int64(98),
		"vpn_last_seen":                   "2026-07-31",
		"is_relay":                        true,
		"relay_provider_name":             "Example Relay",
		"is_anonymous":                    true,
		"is_known_attacker":               true,
		"is_bot":                          true,
		"bot_confidence_score":            int64(95),
		"bot_operator_name":               "ChatGPT",
		"bot_type":                        "ai_crawler",
		"is_known_good_bot":               true,
		"bot_last_seen":                   "2026-09-03",
		"is_spam":                         true,
		"is_cloud_provider":               true,
		"cloud_provider_name":             "Brander Group Inc.",
		"is_corporate_gateway":            true,
		"corporate_gateway_type":          "secure_web_gateway",
		"corporate_gateway_provider_name": "Zscaler",
	}
	for col, v := range want {
		require.Contains(t, row, col)
		assert.True(t, row[col].IsValid(), "column %s should not be NULL", col)
		assert.Equal(t, v, row[col].Get(), "column %s", col)
	}
	// 27 security fields + ip; nothing extra sneaking in.
	assert.Len(t, want, 28)
}

// Bad-bot example from the official docs: attack bot types also flag the IP
// as a known attacker, and bot detail fields are populated.
func TestFetchIPSecurityBadBot(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip": "223.197.196.92",
			"security": map[string]interface{}{
				"threat_score":         40,
				"is_known_attacker":    true,
				"is_bot":               true,
				"bot_confidence_score": 95,
				"bot_type":             "brute_force",
				"is_known_good_bot":    false,
				"bot_last_seen":        "2026-06-21",
			},
		})
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()
	c.Spec.IPs = []string{"223.197.196.92"}

	results := make(chan any, 1)
	require.NoError(t, fetchIPSecurity(context.Background(), c, nil, results))
	close(results)

	sec := (<-results).(*IPSecurityFlat)
	assert.True(t, sec.IsBot)
	assert.Equal(t, 95, sec.BotConfidenceScore)
	assert.Equal(t, "brute_force", sec.BotType)
	assert.False(t, sec.IsKnownGoodBot)
	assert.Equal(t, "", sec.BotOperatorName)
	assert.Equal(t, "2026-06-21", sec.BotLastSeen)
	assert.True(t, sec.IsKnownAttacker)
}
