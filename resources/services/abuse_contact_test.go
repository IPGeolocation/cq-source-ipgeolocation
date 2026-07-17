package services

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/anthropic/cq-source-ipgeolocation/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchAbuseContact(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/abuse", r.URL.Path)
		assert.Equal(t, "1.0.0.0", r.URL.Query().Get("ip"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip": "1.0.0.0",
			"abuse": map[string]interface{}{
				"route":         "1.0.0.0/24",
				"country":       "AU",
				"name":          "IRT-APNICRANDNET-AU",
				"organization":  "",
				"kind":          "group",
				"address":       "PO Box 3646\nSouth Brisbane, QLD 4101\nAustralia",
				"emails":        []string{"helpdesk@apnic.net"},
				"phone_numbers": []string{"+61 7 3858 3100"},
			},
		})
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()
	c.Spec.IPs = []string{"1.0.0.0"}

	results := make(chan any, 10)
	err := fetchAbuseContact(context.Background(), c, nil, results)
	require.NoError(t, err)
	close(results)

	var collected []*AbuseContactFlat
	for item := range results {
		flat, ok := item.(*AbuseContactFlat)
		require.True(t, ok)
		collected = append(collected, flat)
	}

	require.Len(t, collected, 1)
	ab := collected[0]
	assert.Equal(t, "1.0.0.0", ab.IP)
	assert.Equal(t, "1.0.0.0/24", ab.Route)
	assert.Equal(t, "AU", ab.Country)
	assert.Equal(t, "IRT-APNICRANDNET-AU", ab.Name)
	assert.Equal(t, "group", ab.Kind)
	assert.Equal(t, "helpdesk@apnic.net", ab.Emails)
	assert.Equal(t, "+61 7 3858 3100", ab.PhoneNumbers)
	assert.Contains(t, ab.Address, "South Brisbane")
}

func TestFetchAbuseContactMultipleEmails(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip": "8.8.8.8",
			"abuse": map[string]interface{}{
				"route":         "8.8.8.0/24",
				"country":       "US",
				"name":          "Google Abuse",
				"emails":        []string{"abuse@google.com", "network-abuse@google.com"},
				"phone_numbers": []string{"+1-650-253-0000", "+1-650-253-0001"},
			},
		})
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()
	c.Spec.IPs = []string{"8.8.8.8"}

	results := make(chan any, 10)
	err := fetchAbuseContact(context.Background(), c, nil, results)
	require.NoError(t, err)
	close(results)

	var collected []*AbuseContactFlat
	for item := range results {
		collected = append(collected, item.(*AbuseContactFlat))
	}
	require.Len(t, collected, 1)
	assert.Equal(t, "abuse@google.com,network-abuse@google.com", collected[0].Emails)
	assert.Equal(t, "+1-650-253-0000,+1-650-253-0001", collected[0].PhoneNumbers)
}

func TestFetchAbuseContactAPIError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Abuse data requires paid plan"})
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()
	c.Spec.IPs = []string{"8.8.8.8"}

	results := make(chan any, 10)
	err := fetchAbuseContact(context.Background(), c, nil, results)
	require.NoError(t, err, "resolver should log and skip, not return error")
	close(results)

	count := 0
	for range results {
		count++
	}
	assert.Equal(t, 0, count)
}
