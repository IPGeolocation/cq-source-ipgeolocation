package plugin_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/IPGeolocation/cq-source-ipgeolocation/client"
	source "github.com/IPGeolocation/cq-source-ipgeolocation/resources/plugin"
	"github.com/cloudquery/plugin-sdk/v4/message"
	"github.com/cloudquery/plugin-sdk/v4/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginSync(t *testing.T) {
	const ip = "8.8.8.8"
	const ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0"

	for _, tc := range []struct {
		name         string
		optional     bool
		filteredSelf bool
	}{
		{name: "all_tables_default_modules"},
		{name: "all_tables_optional_modules", optional: true},
		{name: "filtered_table_and_caller_ip", filteredSelf: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Response shapes follow the official v3 documentation, as in the resolver tests.
			security := map[string]any{"threat_score": 75, "is_proxy": true, "proxy_provider_names": []string{"Example Proxy", "Other Proxy"}}
			abuse := map[string]any{"emails": []string{"abuse@example.com", "security@example.com"}, "phone_numbers": []string{"+1-555-0100"}}
			geo := map[string]any{
				"ip": ip, "location": map[string]any{"country_code2": "US", "city": "Mountain View", "is_eu": false},
				"country_metadata": map[string]any{"languages": []string{"en-US", "es-US"}},
				"asn":              map[string]any{"as_number": "AS15169", "organization": "Google LLC"},
				"time_zone":        map[string]any{"name": "America/Los_Angeles", "offset": -8.0},
			}
			asn := map[string]any{"as_number": "AS15169", "organization": "Google LLC"}
			geoInclude, asnInclude := "", ""
			geoRow := map[string]any{
				"ip": ip, "city": "Mountain View", "country_code2": "US", "is_eu": false,
				"languages": "en-US,es-US", "asn_organization": "Google LLC", "timezone_offset": -8.0,
				"threat_score": nil, "abuse_emails": nil, "hostname": nil, "accuracy_radius": nil, "dma_code": nil,
			}
			var routes, peers any
			if tc.optional {
				geoInclude = "security,abuse,geo_accuracy,dma_code,hostnameFallbackLive"
				asnInclude = "peers,routes"
				geo["security"], geo["abuse"], geo["hostname"] = security, abuse, "dns.google"
				location := geo["location"].(map[string]any)
				location["accuracy_radius"], location["dma_code"] = "5.0", "807"
				asn["routes"] = []string{"8.8.8.0/24", "8.8.4.0/24"}
				asn["peers"] = []map[string]string{{"as_number": "AS3356"}, {"as_number": "AS1299"}}
				routes, peers = "8.8.8.0/24,8.8.4.0/24", "AS3356,AS1299"
				geoRow["threat_score"], geoRow["abuse_emails"], geoRow["hostname"] = int64(75), "abuse@example.com,security@example.com", "dns.google"
				geoRow["accuracy_radius"], geoRow["dma_code"] = "5.0", "807"
			}
			wantRows := map[string][]map[string]any{
				"ipgeolocation_ip_geolocation": {geoRow},
				"ipgeolocation_ip_security": {{
					"ip": ip, "threat_score": int64(75), "is_proxy": true, "proxy_provider_names": "Example Proxy,Other Proxy",
				}},
				"ipgeolocation_abuse_contact": {{
					"ip": ip, "emails": "abuse@example.com,security@example.com", "phone_numbers": "+1-555-0100",
				}},
				"ipgeolocation_asn": {
					{"query_ip": ip, "query_asn": "", "as_number": "AS15169", "routes": routes, "peers": peers, "upstreams": nil},
					{"query_ip": "", "query_asn": "15169", "as_number": "AS15169", "routes": routes, "peers": peers, "upstreams": nil},
				},
				"ipgeolocation_user_agent": {{
					"user_agent_string": ua, "name": "Chrome", "device_name": "Windows Desktop", "os_name": "Windows",
				}},
			}
			wantCalls := map[string]int{"/v3/ipgeo": 1, "/v3/security": 1, "/v3/abuse": 1, "/v3/asn": 2, "/v3/user-agent": 1}
			options := plugin.SyncOptions{Tables: []string{"ipgeolocation_*"}, DeterministicCQID: true}
			if tc.filteredSelf {
				for table := range wantRows {
					if table != "ipgeolocation_ip_geolocation" {
						options.SkipTables = append(options.SkipTables, table)
						delete(wantRows, table)
					}
				}
				wantCalls = map[string]int{"/v3/ipgeo": 1}
			}

			var mu sync.Mutex
			calls := make(map[string]int)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				calls[r.URL.Path]++
				mu.Unlock()
				assert.Equal(t, "test-api-key", r.URL.Query().Get("apiKey"))
				assert.Equal(t, "cq-sync-test", r.Header.Get("User-Agent"))
				assert.Equal(t, "application/json", r.Header.Get("Accept"))
				w.Header().Set("Content-Type", "application/json")
				var response any
				if r.URL.Path == "/v3/user-agent" {
					assert.Equal(t, http.MethodPost, r.Method)
					assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
					var body map[string]string
					assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					assert.Equal(t, map[string]string{"uaString": ua}, body)
					response = map[string]any{
						"user_agent_string": ua, "name": "Chrome",
						"device": map[string]string{"name": "Windows Desktop"}, "operating_system": map[string]string{"name": "Windows"},
					}
				} else {
					assert.Equal(t, http.MethodGet, r.Method)
					if r.URL.Path == "/v3/asn" && r.URL.Query().Has("asn") {
						assert.Equal(t, "15169", r.URL.Query().Get("asn"))
						assert.False(t, r.URL.Query().Has("ip"))
					} else if tc.filteredSelf {
						assert.False(t, r.URL.Query().Has("ip"), "caller-IP lookup must omit ip")
					} else {
						assert.Equal(t, ip, r.URL.Query().Get("ip"))
					}
					switch r.URL.Path {
					case "/v3/ipgeo":
						assert.Equal(t, geoInclude, r.URL.Query().Get("include"))
						response = geo
					case "/v3/security":
						response = map[string]any{"ip": ip, "security": security}
					case "/v3/abuse":
						response = map[string]any{"ip": ip, "abuse": abuse}
					case "/v3/asn":
						assert.Equal(t, asnInclude, r.URL.Query().Get("include"))
						response = map[string]any{"asn": asn}
					default:
						t.Errorf("unexpected API path: %s", r.URL.Path)
						http.Error(w, "endpoint not defined", http.StatusNotImplemented)
						return
					}
				}
				assert.NoError(t, json.NewEncoder(w).Encode(response))
			}))
			defer server.Close()

			spec := client.Spec{
				APIKey: "test-api-key", Endpoint: server.URL, UserAgent: "cq-sync-test",
				Timeout: "2s", RetryAttempts: 1, Concurrency: 5, RateLimit: 1000,
				IPs: []string{ip}, ASNs: []string{"15169"}, UserAgents: []string{ua},
				IncludeSecurity: tc.optional, IncludeAbuse: tc.optional, IncludeGeoAccuracy: tc.optional,
				IncludeDMACode: tc.optional, IncludeHostname: tc.optional,
			}
			if tc.optional {
				spec.ASNInclude = []string{"Routes", "PEERS", "peers"}
			}
			if tc.filteredSelf {
				spec.IPs = nil
			}
			specBytes, err := json.Marshal(spec)
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			p := source.Plugin()
			require.NoError(t, p.Init(ctx, specBytes, plugin.NewClientOptions{}))
			defer func() { assert.NoError(t, p.Close(context.Background())) }()
			messages, err := p.SyncAll(ctx, options)
			for _, insert := range messages.GetInserts() {
				defer insert.Record.Release()
			}
			require.NoError(t, err)

			var migrated []string
			gotRows := make(map[string][]map[string]any)
			for _, msg := range messages {
				switch m := msg.(type) {
				case *message.SyncMigrateTable:
					migrated = append(migrated, m.Table.Name)
				case *message.SyncInsert:
					table := m.GetTable().Name
					require.Contains(t, wantRows, table)
					for row := 0; row < int(m.Record.NumRows()); row++ {
						values := make(map[string]any)
						for col, field := range m.Record.Schema().Fields() {
							values[field.Name] = m.Record.Column(col).GetOneForMarshal(row)
						}
						assert.NotEmpty(t, values["_cq_id"], "table %s must emit a CloudQuery ID", table)
						selected := make(map[string]any)
						for column := range wantRows[table][0] {
							require.Contains(t, values, column, "table %s", table)
							selected[column] = values[column]
						}
						gotRows[table] = append(gotRows[table], selected)
					}
				default:
					t.Fatalf("unexpected sync message: %#v", msg)
				}
			}
			var wantTables []string
			for table, rows := range wantRows {
				wantTables = append(wantTables, table)
				assert.ElementsMatch(t, rows, gotRows[table], "table %s", table)
			}
			assert.ElementsMatch(t, wantTables, migrated)
			mu.Lock()
			assert.Equal(t, wantCalls, calls, "sync must call only the selected endpoints")
			mu.Unlock()
		})
	}
}
