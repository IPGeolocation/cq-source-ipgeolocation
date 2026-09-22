package services

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudquery/plugin-sdk/v4/schema"

	"github.com/IPGeolocation/cq-source-ipgeolocation/internal/ipgeolocation"
)

// SecurityFields is the complete `security` object of the IPGeolocation.io API,
// flattened for CloudQuery. It is defined once and embedded by every table that
// exposes security data (ipgeolocation_ip_geolocation and
// ipgeolocation_ip_security), so the two can never drift apart.
//
// Array fields (provider names) are stored comma-separated, matching the
// convention used elsewhere in this plugin.
//
// Reference: https://ipgeolocation.io/documentation/ip-security-api.html
type SecurityFields struct {
	ThreatScore int `json:"threat_score"`

	IsTor bool `json:"is_tor"`

	IsProxy              bool   `json:"is_proxy"`
	ProxyProviderNames   string `json:"proxy_provider_names"`
	ProxyConfidenceScore int    `json:"proxy_confidence_score"`
	ProxyLastSeen        string `json:"proxy_last_seen"`
	IsResidentialProxy   bool   `json:"is_residential_proxy"`

	IsVPN              bool   `json:"is_vpn"`
	VPNProviderNames   string `json:"vpn_provider_names"`
	VPNConfidenceScore int    `json:"vpn_confidence_score"`
	VPNLastSeen        string `json:"vpn_last_seen"`

	IsRelay           bool   `json:"is_relay"`
	RelayProviderName string `json:"relay_provider_name"`

	IsAnonymous     bool `json:"is_anonymous"`
	IsKnownAttacker bool `json:"is_known_attacker"`

	IsBot              bool   `json:"is_bot"`
	BotConfidenceScore int    `json:"bot_confidence_score"`
	BotOperatorName    string `json:"bot_operator_name"`
	BotType            string `json:"bot_type"`
	IsKnownGoodBot     bool   `json:"is_known_good_bot"`
	BotLastSeen        string `json:"bot_last_seen"`

	IsSpam bool `json:"is_spam"`

	IsCloudProvider   bool   `json:"is_cloud_provider"`
	CloudProviderName string `json:"cloud_provider_name"`

	IsCorporateGateway           bool   `json:"is_corporate_gateway"`
	CorporateGatewayType         string `json:"corporate_gateway_type"`
	CorporateGatewayProviderName string `json:"corporate_gateway_provider_name"`
}

// newSecurityFields converts the API security object. A nil input yields the
// zero value; callers that need to distinguish "not fetched" should keep a nil
// pointer instead of calling this.
func newSecurityFields(s *ipgeolocation.Security) SecurityFields {
	if s == nil {
		return SecurityFields{}
	}
	return SecurityFields{
		ThreatScore: s.ThreatScore,

		IsTor: s.IsTor,

		IsProxy:              s.IsProxy,
		ProxyProviderNames:   joinStrings(s.ProxyProviderNames),
		ProxyConfidenceScore: s.ProxyConfidenceScore,
		ProxyLastSeen:        s.ProxyLastSeen,
		IsResidentialProxy:   s.IsResidentialProxy,

		IsVPN:              s.IsVPN,
		VPNProviderNames:   joinStrings(s.VPNProviderNames),
		VPNConfidenceScore: s.VPNConfidenceScore,
		VPNLastSeen:        s.VPNLastSeen,

		IsRelay:           s.IsRelay,
		RelayProviderName: s.RelayProviderName,

		IsAnonymous:     s.IsAnonymous,
		IsKnownAttacker: s.IsKnownAttacker,

		IsBot:              s.IsBot,
		BotConfidenceScore: s.BotConfidenceScore,
		BotOperatorName:    s.BotOperatorName,
		BotType:            s.BotType,
		IsKnownGoodBot:     s.IsKnownGoodBot,
		BotLastSeen:        s.BotLastSeen,

		IsSpam: s.IsSpam,

		IsCloudProvider:   s.IsCloudProvider,
		CloudProviderName: s.CloudProviderName,

		IsCorporateGateway:           s.IsCorporateGateway,
		CorporateGatewayType:         s.CorporateGatewayType,
		CorporateGatewayProviderName: s.CorporateGatewayProviderName,
	}
}

// AbuseFields is the complete `abuse` object of the IPGeolocation.io API,
// flattened for CloudQuery. ipgeolocation_abuse_contact embeds it directly
// (columns: route, country, name, ...); ipgeolocation_ip_geolocation nests it
// so the columns are prefixed (abuse_route, abuse_country, abuse_name, ...) and
// cannot collide with the location/network columns of the same names.
//
// Reference: https://ipgeolocation.io/documentation/ip-abuse-contact-api.html
type AbuseFields struct {
	Route        string `json:"route"`
	Country      string `json:"country"`
	Name         string `json:"name"`
	Organization string `json:"organization"`
	Kind         string `json:"kind"`
	Address      string `json:"address"`
	Emails       string `json:"emails"`        // comma-separated
	PhoneNumbers string `json:"phone_numbers"` // comma-separated
}

// newAbuseFields converts the API abuse object. A nil input yields the zero value.
func newAbuseFields(a *ipgeolocation.AbuseContact) AbuseFields {
	if a == nil {
		return AbuseFields{}
	}
	return AbuseFields{
		Route:        a.Route,
		Country:      a.Country,
		Name:         a.Name,
		Organization: a.Organization,
		Kind:         a.Kind,
		Address:      a.Address,
		Emails:       joinStrings(a.Emails),
		PhoneNumbers: joinStrings(a.PhoneNumbers),
	}
}

// nullSafeResolver is a transformers.ResolverTransformer that resolves a
// (possibly dotted) Go field path against the synced item, leaving the column
// NULL when any pointer along the path is nil.
//
// It exists because the SDK's default path resolver panics when it has to walk
// through a nil embedded pointer. We need nil pointers to represent optional
// API modules that were never requested (e.g. security data when
// include_security is false) so those columns read as NULL ("unknown") rather
// than false/0 ("checked and clean").
func nullSafeResolver(_ reflect.StructField, path string) schema.ColumnResolver {
	parts := strings.Split(path, ".")
	return func(_ context.Context, _ schema.ClientMeta, r *schema.Resource, c schema.Column) error {
		v := reflect.ValueOf(r.Item)
		for _, part := range parts {
			for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
				if v.IsNil() {
					return nil // leave the column NULL
				}
				v = v.Elem()
			}
			if v.Kind() != reflect.Struct {
				return fmt.Errorf("resolving %s: %s is not a struct", path, v.Type())
			}
			sf, ok := v.Type().FieldByName(part)
			if !ok {
				return fmt.Errorf("resolving %s: no field %q in %s", path, part, v.Type())
			}
			fv, err := v.FieldByIndexErr(sf.Index)
			if err != nil {
				return nil // nil embedded pointer along the way: leave NULL
			}
			v = fv
		}
		// A pointer leaf (e.g. *string) means "optional": nil is NULL, otherwise
		// the pointed-to value is stored.
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return nil
			}
			v = v.Elem()
		}
		return r.Set(c.Name, v.Interface())
	}
}

// strPtr returns a pointer to s, for optional (NULL-able) string columns.
func strPtr(s string) *string { return &s }
