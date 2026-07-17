package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IPGeolocation/cq-source-ipgeolocation/client"
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
