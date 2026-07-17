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

func TestFetchASNDetailByIP(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/asn", r.URL.Path)
		assert.Equal(t, "49.12.0.0", r.URL.Query().Get("ip"))
		assert.Contains(t, r.URL.Query().Get("include"), "peers")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip": "49.12.0.0",
			"asn": map[string]interface{}{
				"as_number":          "AS24940",
				"organization":       "Hetzner Online GmbH",
				"country":            "DE",
				"type":               "HOSTING",
				"domain":             "hetzner.com",
				"date_allocated":     "2002-06-03",
				"asn_name":           "HETZNER-AS",
				"allocation_status":  "ASSIGNED",
				"num_of_ipv4_routes": "84",
				"num_of_ipv6_routes": "6",
				"rir":                "RIPE",
				"routes":             []string{"49.12.0.0/16", "49.13.0.0/16"},
				"peers": []map[string]string{
					{"as_number": "AS3356", "description": "Level 3", "country": "US"},
					{"as_number": "AS6939", "description": "Hurricane Electric", "country": "US"},
				},
				"upstreams": []map[string]string{
					{"as_number": "AS1299", "description": "Arelion", "country": "SE"},
				},
				"downstreams": []map[string]string{
					{"as_number": "AS213230", "description": "Hetzner Finland", "country": "FI"},
				},
			},
		})
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()
	c.Spec.IPs = []string{"49.12.0.0"}

	results := make(chan any, 10)
	err := fetchASNDetail(context.Background(), c, nil, results)
	require.NoError(t, err)
	close(results)

	var collected []*ASNDetailFlat
	for item := range results {
		flat, ok := item.(*ASNDetailFlat)
		require.True(t, ok)
		collected = append(collected, flat)
	}

	require.Len(t, collected, 1)
	asn := collected[0]
	assert.Equal(t, "49.12.0.0", asn.QueryIP)
	assert.Equal(t, "", asn.QueryASN)
	assert.Equal(t, "AS24940", asn.ASNumber)
	assert.Equal(t, "Hetzner Online GmbH", asn.Organization)
	assert.Equal(t, "HOSTING", asn.Type)
	assert.Equal(t, "hetzner.com", asn.Domain)
	assert.Equal(t, "HETZNER-AS", asn.ASNName)
	assert.Equal(t, "ASSIGNED", asn.AllocationStatus)
	assert.Equal(t, "84", asn.NumIPv4Routes)
	assert.Equal(t, "6", asn.NumIPv6Routes)
	assert.Equal(t, "RIPE", asn.RIR)
	assert.Equal(t, "49.12.0.0/16,49.13.0.0/16", asn.Routes)
	assert.Equal(t, "AS3356,AS6939", asn.Peers)
	assert.Equal(t, "AS1299", asn.Upstreams)
	assert.Equal(t, "AS213230", asn.Downstreams)
}

func TestFetchASNDetailByASN(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/asn", r.URL.Path)
		asnParam := r.URL.Query().Get("asn")
		assert.Equal(t, "15169", asnParam)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"asn": map[string]interface{}{
				"as_number":          "AS15169",
				"organization":       "Google LLC",
				"country":            "US",
				"type":               "BUSINESS",
				"domain":             "google.com",
				"asn_name":           "GOOGLE",
				"num_of_ipv4_routes": "500",
				"num_of_ipv6_routes": "200",
				"rir":                "ARIN",
			},
		})
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()
	c.Spec.IPs = nil
	c.Spec.ASNs = []string{"15169"}

	results := make(chan any, 10)
	err := fetchASNDetail(context.Background(), c, nil, results)
	require.NoError(t, err)
	close(results)

	var collected []*ASNDetailFlat
	for item := range results {
		collected = append(collected, item.(*ASNDetailFlat))
	}

	require.Len(t, collected, 1)
	asn := collected[0]
	assert.Equal(t, "", asn.QueryIP)
	assert.Equal(t, "15169", asn.QueryASN)
	assert.Equal(t, "AS15169", asn.ASNumber)
	assert.Equal(t, "Google LLC", asn.Organization)
	assert.Equal(t, "GOOGLE", asn.ASNName)
}

func TestFetchASNDetailBothIPAndASN(t *testing.T) {
	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"asn": map[string]interface{}{
				"as_number":    "AS1",
				"organization": "Test Org",
			},
		})
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()
	c.Spec.IPs = []string{"1.1.1.1"}
	c.Spec.ASNs = []string{"24940", "15169"}

	results := make(chan any, 10)
	err := fetchASNDetail(context.Background(), c, nil, results)
	require.NoError(t, err)
	close(results)

	count := 0
	for range results {
		count++
	}
	assert.Equal(t, 3, count, "should have 1 IP + 2 ASN results")
	assert.Equal(t, 3, callCount)
}
