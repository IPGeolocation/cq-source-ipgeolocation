package plugin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IPGeolocation/cq-source-ipgeolocation/client"
	source "github.com/IPGeolocation/cq-source-ipgeolocation/resources/plugin"
	"github.com/cloudquery/plugin-sdk/v4/plugin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginResponseSizeSetting(t *testing.T) {
	largeCity := strings.Repeat("x", 1100000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		city := "Small City"
		if r.URL.Query().Get("ip") == "8.8.8.8" {
			city = largeCity
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ip": r.URL.Query().Get("ip"), "location": map[string]string{"city": city}})
	}))
	t.Cleanup(server.Close)
	for _, limit := range []int64{-1, 1, 2} {
		spec := client.Spec{APIKey: "test-key", Endpoint: server.URL, IPs: []string{"8.8.8.8", "1.1.1.1"}, RetryAttempts: 1}
		if limit > 0 {
			spec.MaxResponseMB = &limit
		}
		encoded, err := json.Marshal(spec)
		require.NoError(t, err)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		p := source.Plugin()
		var logs bytes.Buffer
		p.SetLogger(zerolog.New(zerolog.SyncWriter(&logs)))
		require.NoError(t, p.Init(ctx, encoded, plugin.NewClientOptions{}))
		messages, err := p.SyncAll(ctx, plugin.SyncOptions{Tables: []string{"ipgeolocation_ip_geolocation"}})
		require.NoError(t, err)
		var ips []string
		for _, insert := range messages.GetInserts() {
			for row := 0; row < int(insert.Record.NumRows()); row++ {
				for col, field := range insert.Record.Schema().Fields() {
					if field.Name == "ip" {
						ips = append(ips, insert.Record.Column(col).GetOneForMarshal(row).(string))
					}
				}
			}
			insert.Record.Release()
		}
		if limit == 1 {
			assert.Equal(t, []string{"1.1.1.1"}, ips)
			assert.Contains(t, logs.String(), "response exceeds the configured size limit")
		} else {
			assert.ElementsMatch(t, spec.IPs, ips)
			assert.NotContains(t, logs.String(), "response exceeds")
		}
		require.NoError(t, p.Close(ctx))
		cancel()
	}
}

func TestPluginLogsHideAPIKey(t *testing.T) {
	const key = "test-only-log-key"
	for _, kind := range []string{"api", "network"} {
		t.Run(kind, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "invalid key " + key})
			}))
			t.Cleanup(server.Close)
			if kind == "network" {
				server.Close()
			}
			spec := client.Spec{APIKey: key, Endpoint: server.URL, IPs: []string{"8.8.8.8"}, UserAgents: []string{"test-browser"}, RetryAttempts: 1}
			encoded, err := json.Marshal(spec)
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			p := source.Plugin()
			var logs bytes.Buffer
			p.SetLogger(zerolog.New(zerolog.SyncWriter(&logs)))
			require.NoError(t, p.Init(ctx, encoded, plugin.NewClientOptions{}))
			defer func() { require.NoError(t, p.Close(context.Background())) }()
			messages, err := p.SyncAll(ctx, plugin.SyncOptions{Tables: []string{"ipgeolocation_*"}})
			for _, insert := range messages.GetInserts() {
				insert.Record.Release()
			}
			require.NoError(t, err)
			assert.NotContains(t, logs.String(), key)
			assert.Contains(t, logs.String(), "[REDACTED]")
			assert.Contains(t, logs.String(), "skipping")
			if kind == "api" {
				assert.EqualValues(t, 5, calls.Load())
			}
		})
	}
}

func TestPluginRejectsInvalidLimitsBeforeRequests(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	t.Cleanup(server.Close)
	for _, fields := range []map[string]any{
		{"timeout": "0s"}, {"timeout": "-1s"}, {"max_response_mb": 0}, {"max_response_mb": -1}, {"max_response_mb": 1.5},
	} {
		fields["api_key"], fields["endpoint"] = "test-key", server.URL
		encoded, err := json.Marshal(fields)
		require.NoError(t, err)
		p := source.Plugin()
		require.Error(t, p.Init(context.Background(), encoded, plugin.NewClientOptions{}))
	}
	assert.Zero(t, calls.Load())
}
