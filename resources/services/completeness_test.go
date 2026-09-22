package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/IPGeolocation/cq-source-ipgeolocation/internal/ipgeolocation"
)

// The field lists below are transcribed from the official IPGeolocation.io API
// documentation and are deliberately independent of the Go models, so that a
// field missing from either the model or a table is caught here:
//
//   - security: https://ipgeolocation.io/documentation/ip-security-api.html
//   - abuse:    https://ipgeolocation.io/documentation/ip-abuse-contact-api.html
//   - asn:      https://ipgeolocation.io/documentation/asn-api.html
//
// If the API adds a field, add it here first; these tests then point at every
// place that has to change.
var (
	officialSecurityFields = []string{
		"threat_score",
		"is_tor",
		"is_proxy", "proxy_provider_names", "proxy_confidence_score", "proxy_last_seen", "is_residential_proxy",
		"is_vpn", "vpn_provider_names", "vpn_confidence_score", "vpn_last_seen",
		"is_relay", "relay_provider_name",
		"is_anonymous", "is_known_attacker",
		"is_bot", "bot_confidence_score", "bot_operator_name", "bot_type", "is_known_good_bot", "bot_last_seen",
		"is_spam",
		"is_cloud_provider", "cloud_provider_name",
		"is_corporate_gateway", "corporate_gateway_type", "corporate_gateway_provider_name",
	}

	officialAbuseFields = []string{
		"route", "country", "name", "organization", "kind", "address", "emails", "phone_numbers",
	}

	officialASNFields = []string{
		"as_number", "organization", "country", "type", "domain", "date_allocated", "asn_name",
		"allocation_status", "num_of_ipv4_routes", "num_of_ipv6_routes", "rir",
		"routes", "peers", "upstreams", "downstreams", "whois_response",
	}
)

func TestOfficialFieldCounts(t *testing.T) {
	// Guards against the lists above being edited by accident.
	assert.Len(t, officialSecurityFields, 27)
	assert.Len(t, officialAbuseFields, 8)
	assert.Len(t, officialASNFields, 16)
}

func TestAPIModelsMatchOfficialDocs(t *testing.T) {
	assert.ElementsMatch(t, officialSecurityFields, jsonTags(ipgeolocation.Security{}), "ipgeolocation.Security model")
	assert.ElementsMatch(t, officialAbuseFields, jsonTags(ipgeolocation.AbuseContact{}), "ipgeolocation.AbuseContact model")
	assert.ElementsMatch(t, officialASNFields, jsonTags(ipgeolocation.ASNDetail{}), "ipgeolocation.ASNDetail model")
}

func TestSecurityFieldsInBothTables(t *testing.T) {
	dedicated := columnNames(t, IPSecurityTable())
	geo := columnNames(t, IPGeolocationTable())

	for _, f := range officialSecurityFields {
		assert.True(t, dedicated[f], "ipgeolocation_ip_security is missing column %q", f)
		assert.True(t, geo[f], "ipgeolocation_ip_geolocation is missing security column %q", f)
	}
}

func TestAbuseFieldsInBothTables(t *testing.T) {
	dedicated := columnNames(t, AbuseContactTable())
	geo := columnNames(t, IPGeolocationTable())

	for _, f := range officialAbuseFields {
		assert.True(t, dedicated[f], "ipgeolocation_abuse_contact is missing column %q", f)
		assert.True(t, geo["abuse_"+f], "ipgeolocation_ip_geolocation is missing abuse column %q", "abuse_"+f)
	}
}

func TestASNFieldsInTable(t *testing.T) {
	cols := columnNames(t, ASNDetailTable())
	for _, f := range officialASNFields {
		assert.True(t, cols[f], "ipgeolocation_asn is missing column %q", f)
	}
}

// Existing column names must not change: users may already query them.
func TestGeolocationKeepsLegacySecurityAndAbuseColumns(t *testing.T) {
	geo := columnNames(t, IPGeolocationTable())
	legacy := []string{
		"threat_score", "is_tor", "is_proxy", "is_residential_proxy", "is_vpn", "is_relay",
		"is_anonymous", "is_known_attacker", "is_bot", "is_spam", "is_cloud_provider", "cloud_provider_name",
		"abuse_route", "abuse_name", "abuse_emails", "abuse_country",
	}
	for _, c := range legacy {
		assert.True(t, geo[c], "legacy column %q was removed or renamed", c)
	}
}

func TestPrimaryKeysUnchanged(t *testing.T) {
	pk := func(tbl interface{ PrimaryKeys() []string }) []string { return tbl.PrimaryKeys() }
	assert.Equal(t, []string{"ip"}, pk(buildTable(t, IPGeolocationTable())))
	assert.Equal(t, []string{"ip"}, pk(buildTable(t, IPSecurityTable())))
	assert.Equal(t, []string{"ip"}, pk(buildTable(t, AbuseContactTable())))
	assert.ElementsMatch(t, []string{"query_ip", "query_asn"}, pk(buildTable(t, ASNDetailTable())))
}
