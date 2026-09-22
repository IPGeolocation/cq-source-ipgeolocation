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

func TestFetchUserAgent(t *testing.T) {
	mockResponse := map[string]interface{}{
		"user_agent_string": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"name":              "Chrome",
		"type":              "Browser",
		"version":           "120.0.0.0",
		"version_major":     "120",
		"device": map[string]interface{}{
			"name":  "Windows Desktop",
			"type":  "Desktop",
			"brand": "Unknown",
			"cpu":   "Intel x86_64",
		},
		"engine": map[string]interface{}{
			"name":          "Blink",
			"type":          "Browser",
			"version":       "120.0.0.0",
			"version_major": "120",
		},
		"operating_system": map[string]interface{}{
			"name":          "Windows",
			"type":          "Desktop",
			"version":       "10",
			"version_major": "10",
			"build":         "22631",
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Official API: POST /v3/user-agent with a JSON body {"uaString": "..."}.
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v3/user-agent", r.URL.Path)
		var body map[string]string
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36", body["uaString"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResponse)
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()

	c.Spec.UserAgents = []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"}

	results := make(chan any, 10)
	err := fetchUserAgent(context.Background(), c, nil, results)
	require.NoError(t, err)
	close(results)

	var collected []*UserAgentFlat
	for item := range results {
		flat, ok := item.(*UserAgentFlat)
		require.True(t, ok)
		collected = append(collected, flat)
	}

	require.Len(t, collected, 1)
	ua := collected[0]
	assert.Equal(t, "Chrome", ua.Name)
	assert.Equal(t, "Browser", ua.Type)
	assert.Equal(t, "120", ua.VersionMajor)
	assert.Equal(t, "Windows Desktop", ua.DeviceName)
	assert.Equal(t, "Desktop", ua.DeviceType)
	assert.Equal(t, "Blink", ua.EngineName)
	assert.Equal(t, "Windows", ua.OSName)
	assert.Equal(t, "22631", ua.OSBuild)
}

func TestFetchUserAgentEmpty(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("API should not be called when no user agents are configured")
	})

	c, cleanup := client.TestClient(t, handler)
	defer cleanup()

	c.Spec.UserAgents = nil

	results := make(chan any, 10)
	err := fetchUserAgent(context.Background(), c, nil, results)
	require.NoError(t, err)
	close(results)

	count := 0
	for range results {
		count++
	}
	assert.Equal(t, 0, count, "should have zero results when no user agents configured")
}
