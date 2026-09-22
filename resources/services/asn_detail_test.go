package services

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/IPGeolocation/cq-source-ipgeolocation/client"
	"github.com/cloudquery/plugin-sdk/v4/scalar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hetznerASN is a /v3/asn payload in the shape of the official docs. The
// optional modules are only returned when they are requested via `include`,
// exactly as the real API does.
func hetznerASN(include string) map[string]interface{} {
	asn := map[string]interface{}{
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
	}
	has := map[string]bool{}
	for _, m := range strings.Split(include, ",") {
		has[m] = true
	}
	if has["routes"] {
		asn["routes"] = []string{"49.12.0.0/16", "49.13.0.0/16"}
	}
	if has["peers"] {
		asn["peers"] = []map[string]string{
			{"as_number": "AS3356", "description": "Level 3", "country": "US"},
			{"as_number": "AS6939", "description": "Hurricane Electric", "country": "US"},
		}
	}
	if has["upstreams"] {
		asn["upstreams"] = []map[string]string{{"as_number": "AS1299", "description": "Arelion", "country": "SE"}}
	}
	if has["downstreams"] {
		asn["downstreams"] = []map[string]string{{"as_number": "AS213230", "description": "Hetzner Finland", "country": "FI"}}
	}
	if has["whois_response"] {
		asn["whois_response"] = "ASNumber: 24940\nASName: HETZNER-AS"
	}
	return map[string]interface{}{"ip": "49.12.0.0", "asn": asn}
}

// runASN performs a by-IP ASN sync with the given asn_include config and
// returns the request's `include` query value (and whether it was present)
// together with the resolved table row.
func runASN(t *testing.T, asnInclude []string) (include string, present bool, row map[string]scalar.Scalar) {
	t.Helper()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/asn", r.URL.Path)
		assert.Equal(t, "49.12.0.0", r.URL.Query().Get("ip"))
		include = r.URL.Query().Get("include")
		present = r.URL.Query().Has("include")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(hetznerASN(include))
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()
	c.Spec.IPs = []string{"49.12.0.0"}
	c.Spec.ASNInclude = asnInclude

	results := make(chan any, 10)
	require.NoError(t, fetchASNDetail(context.Background(), c, nil, results))
	close(results)

	var items []any
	for it := range results {
		items = append(items, it)
	}
	require.Len(t, items, 1)
	return include, present, resolveRow(t, buildTable(t, ASNDetailTable()), items[0])
}

var asnOptionalColumns = []string{"routes", "peers", "upstreams", "downstreams", "whois_response"}

func TestFetchASNDetailByIPDefaultIsCompact(t *testing.T) {
	include, present, row := runASN(t, nil)

	// By default nothing optional is requested: no include param at all.
	assert.False(t, present, "must not send include= by default, got %q", include)

	// Base fields are populated...
	assert.Equal(t, "49.12.0.0", row["query_ip"].Get())
	assert.Equal(t, "", row["query_asn"].Get())
	assert.Equal(t, "AS24940", row["as_number"].Get())
	assert.Equal(t, "Hetzner Online GmbH", row["organization"].Get())
	assert.Equal(t, "HOSTING", row["type"].Get())
	assert.Equal(t, "hetzner.com", row["domain"].Get())
	assert.Equal(t, "HETZNER-AS", row["asn_name"].Get())
	assert.Equal(t, "ASSIGNED", row["allocation_status"].Get())
	assert.Equal(t, "84", row["num_of_ipv4_routes"].Get())
	assert.Equal(t, "6", row["num_of_ipv6_routes"].Get())
	assert.Equal(t, "RIPE", row["rir"].Get())

	// ...and the heavy optional columns are NULL.
	for _, col := range asnOptionalColumns {
		assert.False(t, row[col].IsValid(), "column %s should be NULL when not configured", col)
	}
}

func TestFetchASNDetailIncludeSelectedModules(t *testing.T) {
	// Mixed case / duplicates / arbitrary order are normalised to canonical order.
	include, present, row := runASN(t, []string{"Routes", "peers", "PEERS"})

	assert.True(t, present)
	assert.Equal(t, "peers,routes", include)

	assert.True(t, row["peers"].IsValid())
	assert.Equal(t, "AS3356,AS6939", row["peers"].Get())
	assert.True(t, row["routes"].IsValid())
	assert.Equal(t, "49.12.0.0/16,49.13.0.0/16", row["routes"].Get())

	// Not selected => NULL.
	for _, col := range []string{"upstreams", "downstreams", "whois_response"} {
		assert.False(t, row[col].IsValid(), "column %s should be NULL when not selected", col)
	}
	// Base data unaffected.
	assert.Equal(t, "AS24940", row["as_number"].Get())
}

func TestFetchASNDetailIncludeEverything(t *testing.T) {
	include, _, row := runASN(t, []string{"*"})

	// "*" is expanded client-side to the explicit list.
	assert.Equal(t, "peers,upstreams,downstreams,routes,whois_response", include)

	assert.Equal(t, "AS3356,AS6939", row["peers"].Get())
	assert.Equal(t, "AS1299", row["upstreams"].Get())
	assert.Equal(t, "AS213230", row["downstreams"].Get())
	assert.Equal(t, "49.12.0.0/16,49.13.0.0/16", row["routes"].Get())
	assert.Equal(t, "ASNumber: 24940\nASName: HETZNER-AS", row["whois_response"].Get())
}

func TestFetchASNDetailIncludeOnlyWhois(t *testing.T) {
	include, _, row := runASN(t, []string{"whois_response"})

	assert.Equal(t, "whois_response", include)
	assert.True(t, row["whois_response"].IsValid())
	for _, col := range []string{"peers", "upstreams", "downstreams", "routes"} {
		assert.False(t, row[col].IsValid(), "column %s should be NULL", col)
	}
}

// Requested but the ASN genuinely has none: empty string, not NULL, so
// "none exist" stays distinguishable from "not fetched".
func TestFetchASNDetailRequestedButEmptyIsNotNull(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"asn": map[string]interface{}{"as_number": "AS64500", "organization": "Tiny Org"},
		})
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()
	c.Spec.ASNs = []string{"64500"}
	c.Spec.ASNInclude = []string{"peers", "routes"}

	results := make(chan any, 1)
	require.NoError(t, fetchASNDetail(context.Background(), c, nil, results))
	close(results)

	row := resolveRow(t, buildTable(t, ASNDetailTable()), <-results)
	assert.True(t, row["peers"].IsValid())
	assert.Equal(t, "", row["peers"].Get())
	assert.True(t, row["routes"].IsValid())
	assert.Equal(t, "", row["routes"].Get())
	assert.False(t, row["upstreams"].IsValid(), "not requested => NULL")
}

func TestFetchASNDetailInvalidIncludeFailsBeforeAnyRequest(t *testing.T) {
	calls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ })

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()
	c.Spec.IPs = []string{"49.12.0.0"}
	c.Spec.ASNInclude = []string{"peers", "bogus"}

	results := make(chan any, 1)
	err := fetchASNDetail(context.Background(), c, nil, results)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
	assert.Equal(t, 0, calls, "no API credits should be spent on an invalid config")
}

// The include selection applies to lookups by ASN number too.
func TestFetchASNDetailByNumberHonoursInclude(t *testing.T) {
	var include string
	var present bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "24940", r.URL.Query().Get("asn"))
		include = r.URL.Query().Get("include")
		present = r.URL.Query().Has("include")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(hetznerASN(include))
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()
	c.Spec.ASNs = []string{"24940"}

	// Default: nothing requested.
	results := make(chan any, 1)
	require.NoError(t, fetchASNDetail(context.Background(), c, nil, results))
	close(results)
	row := resolveRow(t, buildTable(t, ASNDetailTable()), <-results)
	assert.False(t, present)
	assert.False(t, row["peers"].IsValid())
	assert.Equal(t, "24940", row["query_asn"].Get())

	// Configured: only upstreams.
	c.Spec.ASNInclude = []string{"upstreams"}
	results = make(chan any, 1)
	require.NoError(t, fetchASNDetail(context.Background(), c, nil, results))
	close(results)
	row = resolveRow(t, buildTable(t, ASNDetailTable()), <-results)
	assert.Equal(t, "upstreams", include)
	assert.Equal(t, "AS1299", row["upstreams"].Get())
	assert.False(t, row["peers"].IsValid())
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
