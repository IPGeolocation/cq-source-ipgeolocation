package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IPGeolocation/cq-source-ipgeolocation/client"
	"github.com/cloudquery/plugin-sdk/v4/scalar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchIPGeolocation(t *testing.T) {
	mockResponse := map[string]interface{}{
		"ip": "8.8.8.8",
		"location": map[string]interface{}{
			"continent_code":        "NA",
			"continent_name":        "North America",
			"country_code2":         "US",
			"country_code3":         "USA",
			"country_name":          "United States",
			"country_name_official": "United States of America",
			"country_capital":       "Washington, D.C.",
			"state_prov":            "California",
			"state_code":            "US-CA",
			"district":              "Santa Clara County",
			"city":                  "Mountain View",
			"zipcode":               "94043",
			"latitude":              "37.42240",
			"longitude":             "-122.08421",
			"is_eu":                 false,
			"country_flag":          "https://ipgeolocation.io/static/flags/us_64.png",
			"geoname_id":            "5375480",
		},
		"currency": map[string]interface{}{
			"code":   "USD",
			"name":   "US Dollar",
			"symbol": "$",
		},
		"asn": map[string]interface{}{
			"as_number":    "AS15169",
			"organization": "Google LLC",
			"country":      "US",
		},
		"time_zone": map[string]interface{}{
			"name":                    "America/Los_Angeles",
			"offset":                  -8.0,
			"offset_with_dst":         -7.0,
			"current_time":            "2026-01-01 12:00:00.000-0800",
			"current_time_unix":       1735756800.0,
			"current_tz_abbreviation": "PST",
			"is_dst":                  false,
			"dst_savings":             0.0,
			"dst_exists":              true,
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/ipgeo", r.URL.Path)
		assert.Equal(t, "test-api-key", r.URL.Query().Get("apiKey"))
		assert.Equal(t, "8.8.8.8", r.URL.Query().Get("ip"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()

	c.Spec.IPs = []string{"8.8.8.8"}

	results := make(chan any, 10)
	err := fetchIPGeolocation(context.Background(), c, nil, results)
	require.NoError(t, err)
	close(results)

	var collected []*IPGeolocationFlat
	for item := range results {
		flat, ok := item.(*IPGeolocationFlat)
		require.True(t, ok, "expected *IPGeolocationFlat, got %T", item)
		collected = append(collected, flat)
	}

	require.Len(t, collected, 1)
	geo := collected[0]
	assert.Equal(t, "8.8.8.8", geo.IP)
	assert.Equal(t, "United States", geo.CountryName)
	assert.Equal(t, "Mountain View", geo.City)
	assert.Equal(t, "California", geo.StateProv)
	assert.Equal(t, "AS15169", geo.ASNumber)
	assert.Equal(t, "Google LLC", geo.ASNOrganization)
	assert.Equal(t, "USD", geo.CurrencyCode)
	assert.Equal(t, "America/Los_Angeles", geo.TimezoneName)
	assert.False(t, geo.IsEU)
}

func TestFetchIPGeolocationCallerIP(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("ip"), "should not send ip param when looking up caller")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip": "1.2.3.4",
			"location": map[string]interface{}{
				"country_name": "TestCountry",
				"city":         "TestCity",
				"latitude":     "0.0",
				"longitude":    "0.0",
			},
		})
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()

	// No IPs configured: should look up caller's own IP.
	c.Spec.IPs = nil

	results := make(chan any, 10)
	err := fetchIPGeolocation(context.Background(), c, nil, results)
	require.NoError(t, err)
	close(results)

	count := 0
	for range results {
		count++
	}
	assert.Equal(t, 1, count)
}

func TestFetchIPGeolocationAPIError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Invalid API key",
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()

	c.Spec.IPs = []string{"8.8.8.8"}

	results := make(chan any, 10)
	err := fetchIPGeolocation(context.Background(), c, nil, results)
	// The resolver logs warnings and continues; it should not return an error for 401.
	require.NoError(t, err)
	close(results)

	count := 0
	for range results {
		count++
	}
	assert.Equal(t, 0, count, "should have zero results on auth error")
}

// runGeo runs fetchIPGeolocation against handler with the given spec toggles
// and returns the single resolved row.
func runGeo(t *testing.T, handler http.HandlerFunc, includeSecurity, includeAbuse bool) map[string]scalar.Scalar {
	t.Helper()
	return runGeoSpec(t, handler, func(sp *client.Spec) {
		sp.IncludeSecurity = includeSecurity
		sp.IncludeAbuse = includeAbuse
	})
}

// runGeoSpec is like runGeo but lets the caller adjust any spec option.
func runGeoSpec(t *testing.T, handler http.HandlerFunc, mutate func(*client.Spec)) map[string]scalar.Scalar {
	t.Helper()
	c, cleanup := client.TestClient(t, handler)
	defer cleanup()
	c.Spec.IPs = []string{"91.128.103.196"}
	mutate(&c.Spec)

	results := make(chan any, 2)
	require.NoError(t, fetchIPGeolocation(context.Background(), c, nil, results))
	close(results)

	var items []any
	for it := range results {
		items = append(items, it)
	}
	require.Len(t, items, 1)
	return resolveRow(t, buildTable(t, IPGeolocationTable()), items[0])
}

func TestFetchIPGeolocationWithSecurityAndAbuse(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "security,abuse", r.URL.Query().Get("include"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip":       "91.128.103.196",
			"location": map[string]interface{}{"country_name": "Sweden", "city": "Stockholm"},
			"security": securityFixture(),
			"abuse": map[string]interface{}{
				"route":         "91.128.0.0/14",
				"country":       "SE",
				"name":          "Tele2 IP Registry",
				"organization":  "ORG-NCC1-RIPE",
				"kind":          "group",
				"address":       "Tele2 Sverige AB, IP Registry, Torshamnsgatan 17 164 40 Kista SWEDEN",
				"emails":        []string{"abuse@tele2.com"},
				"phone_numbers": []string{"+46856264210"},
			},
		})
	}

	row := runGeo(t, handler, true, true)

	// Non-security data still flows.
	assert.Equal(t, "Sweden", row["country_name"].Get())

	// All 27 security fields, as unprefixed columns.
	for _, f := range officialSecurityFields {
		require.Contains(t, row, f)
		assert.True(t, row[f].IsValid(), "security column %s should be populated", f)
	}
	assert.Equal(t, int64(90), row["threat_score"].Get())
	assert.Equal(t, "ai_crawler", row["bot_type"].Get())
	assert.Equal(t, true, row["is_known_good_bot"].Get())
	assert.Equal(t, "Zscaler", row["corporate_gateway_provider_name"].Get())
	assert.Equal(t, "NetNut,ProxyScrape,Oxy Labs,DataImpulse", row["proxy_provider_names"].Get())
	assert.Equal(t, "2026-07-31", row["vpn_last_seen"].Get())

	// All 8 abuse fields, prefixed with abuse_ so they don't clash with
	// location/network columns (country, route, ...).
	wantAbuse := map[string]any{
		"abuse_route":         "91.128.0.0/14",
		"abuse_country":       "SE",
		"abuse_name":          "Tele2 IP Registry",
		"abuse_organization":  "ORG-NCC1-RIPE",
		"abuse_kind":          "group",
		"abuse_address":       "Tele2 Sverige AB, IP Registry, Torshamnsgatan 17 164 40 Kista SWEDEN",
		"abuse_emails":        "abuse@tele2.com",
		"abuse_phone_numbers": "+46856264210",
	}
	for col, v := range wantAbuse {
		require.Contains(t, row, col)
		assert.True(t, row[col].IsValid(), "abuse column %s should be populated", col)
		assert.Equal(t, v, row[col].Get(), "column %s", col)
	}
}

// When security/abuse are not requested, their columns must be NULL ("unknown"),
// not false/0 which would read as "checked and clean".
func TestFetchIPGeolocationSecurityAbuseNullWhenNotRequested(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("include"), "no include param when neither module is enabled")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip":       "91.128.103.196",
			"location": map[string]interface{}{"country_name": "Sweden"},
		})
	}

	row := runGeo(t, handler, false, false)

	assert.True(t, row["country_name"].IsValid())
	for _, f := range officialSecurityFields {
		assert.False(t, row[f].IsValid(), "security column %s should be NULL when include_security=false", f)
	}
	for _, f := range officialAbuseFields {
		assert.False(t, row["abuse_"+f].IsValid(), "abuse column abuse_%s should be NULL when include_abuse=false", f)
	}
}

// Requesting only security leaves the abuse columns NULL and vice versa.
func TestFetchIPGeolocationOnlySecurityRequested(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "security", r.URL.Query().Get("include"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip":       "91.128.103.196",
			"security": map[string]interface{}{"threat_score": 0, "is_vpn": false},
		})
	}

	row := runGeo(t, handler, true, false)

	// A genuine "clean" result is a real false/0, not NULL.
	assert.True(t, row["threat_score"].IsValid())
	assert.Equal(t, int64(0), row["threat_score"].Get())
	assert.True(t, row["is_vpn"].IsValid())
	assert.Equal(t, false, row["is_vpn"].Get())
	// Abuse was not requested.
	assert.False(t, row["abuse_route"].IsValid())
	assert.False(t, row["abuse_emails"].IsValid())
}

// network.is_cdn / network.cdn_provider_name (official IP Geolocation API docs).
func TestFetchIPGeolocationNetworkCDNFields(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip": "104.16.0.1",
			"network": map[string]interface{}{
				"connection_type":   "",
				"route":             "104.16.0.0/13",
				"is_anycast":        true,
				"is_cdn":            true,
				"cdn_provider_name": "Cloudflare",
			},
		})
	}

	row := runGeo(t, handler, false, false)
	assert.Equal(t, "104.16.0.0/13", row["route"].Get())
	assert.Equal(t, true, row["is_anycast"].Get())
	assert.Equal(t, true, row["is_cdn"].Get())
	assert.Equal(t, "Cloudflare", row["cdn_provider_name"].Get())
}

func TestGeolocationNetworkColumnsPresent(t *testing.T) {
	cols := columnNames(t, IPGeolocationTable())
	for _, c := range []string{"connection_type", "route", "is_anycast", "is_cdn", "cdn_provider_name"} {
		assert.True(t, cols[c], "missing network column %q", c)
	}
}

// Values from the official docs (include=geo_accuracy and include=liveHostname).
func TestFetchIPGeolocationGeoAccuracyAndHostname(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "geo_accuracy,liveHostname", r.URL.Query().Get("include"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip":       "91.128.103.196",
			"hostname": "host196.tele2.example",
			"location": map[string]interface{}{
				"country_name":    "Sweden",
				"city":            "Stockholm",
				"locality":        "Stockholm",
				"accuracy_radius": "4.355",
				"confidence":      "high",
			},
		})
	}

	row := runGeoSpec(t, handler, func(sp *client.Spec) {
		sp.IncludeGeoAccuracy = true
		sp.IncludeHostname = true
		sp.HostnameLookup = "live"
	})

	assert.Equal(t, "Sweden", row["country_name"].Get())
	assert.Equal(t, "Stockholm", row["locality"].Get())
	assert.Equal(t, "4.355", row["accuracy_radius"].Get())
	assert.Equal(t, "high", row["confidence"].Get())
	assert.Equal(t, "host196.tele2.example", row["hostname"].Get())
}

func TestFetchIPGeolocationGeoAccuracyHostnameNullWhenNotRequested(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.False(t, r.URL.Query().Has("include"), "no include param when nothing optional is enabled")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip":       "91.128.103.196",
			"location": map[string]interface{}{"country_name": "Sweden"},
		})
	}

	row := runGeo(t, handler, false, false)
	for _, col := range []string{"locality", "accuracy_radius", "confidence", "dma_code", "hostname"} {
		assert.False(t, row[col].IsValid(), "column %s should be NULL when not requested", col)
	}
}

// Each option is independent: enabling only geo accuracy must not populate hostname.
func TestFetchIPGeolocationOnlyGeoAccuracyRequested(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "geo_accuracy", r.URL.Query().Get("include"))
		w.Header().Set("Content-Type", "application/json")
		// Locality may legitimately be empty for some IPs.
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip":       "91.128.103.196",
			"location": map[string]interface{}{"city": "Stockholm", "locality": "", "accuracy_radius": "9.333", "confidence": "low"},
		})
	}

	row := runGeoSpec(t, handler, func(sp *client.Spec) { sp.IncludeGeoAccuracy = true })
	assert.True(t, row["locality"].IsValid(), "requested but empty is '' not NULL")
	assert.Equal(t, "", row["locality"].Get())
	assert.Equal(t, "low", row["confidence"].Get())
	assert.False(t, row["hostname"].IsValid())
}

// With everything enabled, all includes are combined in a stable order and the
// default hostname mode is database-then-live.
func TestFetchIPGeolocationAllIncludesOrder(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "security,abuse,geo_accuracy,dma_code,hostnameFallbackLive", r.URL.Query().Get("include"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ip": "91.128.103.196", "hostname": "91.128.103.196"})
	}
	row := runGeoSpec(t, handler, func(sp *client.Spec) {
		sp.IncludeSecurity, sp.IncludeAbuse, sp.IncludeGeoAccuracy, sp.IncludeDMACode, sp.IncludeHostname = true, true, true, true, true
	})
	// Unresolvable hostnames come back as the queried IP; that is stored as-is.
	assert.Equal(t, "91.128.103.196", row["hostname"].Get())
}

func TestFetchIPGeolocationInvalidHostnameLookupFailsBeforeRequest(t *testing.T) {
	calls := 0
	c, cleanup := client.TestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer cleanup()
	c.Spec.IPs = []string{"8.8.8.8"}
	c.Spec.IncludeHostname = true
	c.Spec.HostnameLookup = "dns"

	err := fetchIPGeolocation(context.Background(), c, nil, make(chan any, 1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hostname_lookup")
	assert.Equal(t, 0, calls)
}

func TestGeolocationGeoAccuracyAndHostnameColumnsPresent(t *testing.T) {
	cols := columnNames(t, IPGeolocationTable())
	for _, c := range []string{"hostname", "locality", "accuracy_radius", "confidence", "dma_code"} {
		assert.True(t, cols[c], "missing column %q", c)
	}
}

func TestFetchIPGeolocationDMACode(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "dma_code", r.URL.Query().Get("include"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip":       "8.8.8.8",
			"location": map[string]interface{}{"country_name": "United States", "state_prov": "California", "dma_code": "807"},
		})
	}

	row := runGeoSpec(t, handler, func(sp *client.Spec) { sp.IncludeDMACode = true })
	assert.Equal(t, "807", row["dma_code"].Get())
	// Independent of the other optional location modules.
	for _, col := range []string{"locality", "accuracy_radius", "confidence", "hostname"} {
		assert.False(t, row[col].IsValid(), "column %s should be NULL", col)
	}
}

// Outside the US the API has no DMA code: requested-but-empty is ” not NULL.
func TestFetchIPGeolocationDMACodeEmptyOutsideUS(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip":       "91.128.103.196",
			"location": map[string]interface{}{"country_name": "Sweden"},
		})
	}
	row := runGeoSpec(t, handler, func(sp *client.Spec) { sp.IncludeDMACode = true })
	assert.True(t, row["dma_code"].IsValid())
	assert.Equal(t, "", row["dma_code"].Get())
}
