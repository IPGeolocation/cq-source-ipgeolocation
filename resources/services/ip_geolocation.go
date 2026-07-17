package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"

	"github.com/anthropic/cq-source-ipgeolocation/client"
	"github.com/anthropic/cq-source-ipgeolocation/internal/ipgeolocation"
)

// IPGeolocationFlat is a flattened representation of the IP Geolocation API response.
// CloudQuery tables work best with flat structs rather than deeply nested JSON.
type IPGeolocationFlat struct {
	// Core
	IP       string `json:"ip"`
	Domain   string `json:"domain"`
	Hostname string `json:"hostname"`

	// Location
	ContinentCode       string `json:"continent_code"`
	ContinentName       string `json:"continent_name"`
	CountryCode2        string `json:"country_code2"`
	CountryCode3        string `json:"country_code3"`
	CountryName         string `json:"country_name"`
	CountryNameOfficial string `json:"country_name_official"`
	CountryCapital      string `json:"country_capital"`
	StateProv           string `json:"state_prov"`
	StateCode           string `json:"state_code"`
	District            string `json:"district"`
	City                string `json:"city"`
	Zipcode             string `json:"zipcode"`
	Latitude            string `json:"latitude"`
	Longitude           string `json:"longitude"`
	IsEU                bool   `json:"is_eu"`
	GeonameID           string `json:"geoname_id"`

	// Country metadata
	CallingCode string `json:"calling_code"`
	TLD         string `json:"tld"`
	Languages   string `json:"languages"` // comma-separated

	// Network
	ConnectionType string `json:"connection_type"`
	Route          string `json:"route"`
	IsAnycast      bool   `json:"is_anycast"`

	// Currency
	CurrencyCode   string `json:"currency_code"`
	CurrencyName   string `json:"currency_name"`
	CurrencySymbol string `json:"currency_symbol"`

	// ASN
	ASNumber         string `json:"as_number"`
	ASNOrganization  string `json:"asn_organization"`
	ASNCountry       string `json:"asn_country"`
	ASNType          string `json:"asn_type"`
	ASNDomain        string `json:"asn_domain"`
	ASNDateAllocated string `json:"asn_date_allocated"`
	ASNRIR           string `json:"asn_rir"`

	// Company
	CompanyName   string `json:"company_name"`
	CompanyType   string `json:"company_type"`
	CompanyDomain string `json:"company_domain"`

	// Timezone
	TimezoneName             string  `json:"timezone_name"`
	TimezoneOffset           float64 `json:"timezone_offset"`
	TimezoneOffsetWithDST    float64 `json:"timezone_offset_with_dst"`
	TimezoneCurrentTime      string  `json:"timezone_current_time"`
	TimezoneCurrentTimeUnix  float64 `json:"timezone_current_time_unix"`
	TimezoneAbbreviation     string  `json:"timezone_abbreviation"`
	TimezoneIsDST            bool    `json:"timezone_is_dst"`

	// Security (optional, paid plans)
	ThreatScore        int    `json:"threat_score"`
	IsTor              bool   `json:"is_tor"`
	IsProxy            bool   `json:"is_proxy"`
	IsResidentialProxy bool   `json:"is_residential_proxy"`
	IsVPN              bool   `json:"is_vpn"`
	IsRelay            bool   `json:"is_relay"`
	IsAnonymous        bool   `json:"is_anonymous"`
	IsKnownAttacker    bool   `json:"is_known_attacker"`
	IsBot              bool   `json:"is_bot"`
	IsSpam             bool   `json:"is_spam"`
	IsCloudProvider    bool   `json:"is_cloud_provider"`
	CloudProviderName  string `json:"cloud_provider_name"`

	// Abuse (optional, paid plans)
	AbuseRoute   string `json:"abuse_route"`
	AbuseName    string `json:"abuse_name"`
	AbuseEmails  string `json:"abuse_emails"` // comma-separated
	AbuseCountry string `json:"abuse_country"`
}

// IPGeolocationTable returns the table definition for IP geolocation lookups.
func IPGeolocationTable() *schema.Table {
	return &schema.Table{
		Name:        "ipgeolocation_ip_geolocation",
		Description: "IP Geolocation data from IPGeolocation.io /v3/ipgeo endpoint. Returns location, ASN, company, timezone, network, currency, and optionally security and abuse data for each configured IP address.",
		Resolver:    fetchIPGeolocation,
		Transform:   transformers.TransformWithStruct(&IPGeolocationFlat{}, transformers.WithPrimaryKeys("IP")),
	}
}

func fetchIPGeolocation(ctx context.Context, meta schema.ClientMeta, _ *schema.Resource, res chan<- any) error {
	c := meta.(*client.Client)

	ips := c.Spec.IPs
	if len(ips) == 0 {
		// No IPs configured: look up the caller's own IP.
		ips = []string{""}
	}

	var includes []string
	if c.Spec.IncludeSecurity {
		includes = append(includes, "security")
	}
	if c.Spec.IncludeAbuse {
		includes = append(includes, "abuse")
	}

	for _, ip := range ips {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		geo, err := c.IPGeo.GetIPGeolocation(ctx, ip, includes...)
		if err != nil {
			c.Logger.Warn().Str("ip", ip).Err(err).Msg("failed to fetch IP geolocation, skipping")
			continue
		}

		flat := flattenIPGeolocation(geo)
		res <- flat
	}

	return nil
}

// flattenIPGeolocation converts the nested API response into a flat struct.
func flattenIPGeolocation(g *ipgeolocation.IPGeolocation) *IPGeolocationFlat {
	f := &IPGeolocationFlat{
		IP:       g.IP,
		Domain:   g.Domain,
		Hostname: g.Hostname,
	}

	if g.Location != nil {
		f.ContinentCode = g.Location.ContinentCode
		f.ContinentName = g.Location.ContinentName
		f.CountryCode2 = g.Location.CountryCode2
		f.CountryCode3 = g.Location.CountryCode3
		f.CountryName = g.Location.CountryName
		f.CountryNameOfficial = g.Location.CountryNameOfficial
		f.CountryCapital = g.Location.CountryCapital
		f.StateProv = g.Location.StateProv
		f.StateCode = g.Location.StateCode
		f.District = g.Location.District
		f.City = g.Location.City
		f.Zipcode = g.Location.Zipcode
		f.Latitude = g.Location.Latitude
		f.Longitude = g.Location.Longitude
		f.IsEU = g.Location.IsEU
		f.GeonameID = g.Location.GeonameID
	}

	if g.CountryMetadata != nil {
		f.CallingCode = g.CountryMetadata.CallingCode
		f.TLD = g.CountryMetadata.TLD
		f.Languages = strings.Join(g.CountryMetadata.Languages, ",")
	}

	if g.Network != nil {
		f.ConnectionType = g.Network.ConnectionType
		f.Route = g.Network.Route
		f.IsAnycast = g.Network.IsAnycast
	}

	if g.Currency != nil {
		f.CurrencyCode = g.Currency.Code
		f.CurrencyName = g.Currency.Name
		f.CurrencySymbol = g.Currency.Symbol
	}

	if g.ASN != nil {
		f.ASNumber = g.ASN.ASNumber
		f.ASNOrganization = g.ASN.Organization
		f.ASNCountry = g.ASN.Country
		f.ASNType = g.ASN.Type
		f.ASNDomain = g.ASN.Domain
		f.ASNDateAllocated = g.ASN.DateAllocated
		f.ASNRIR = g.ASN.RIR
	}

	if g.Company != nil {
		f.CompanyName = g.Company.Name
		f.CompanyType = g.Company.Type
		f.CompanyDomain = g.Company.Domain
	}

	if g.TimeZone != nil {
		f.TimezoneName = g.TimeZone.Name
		f.TimezoneOffset = g.TimeZone.Offset
		f.TimezoneOffsetWithDST = g.TimeZone.OffsetWithDST
		f.TimezoneCurrentTime = g.TimeZone.CurrentTime
		f.TimezoneCurrentTimeUnix = g.TimeZone.CurrentTimeUnix
		f.TimezoneAbbreviation = g.TimeZone.CurrentTZAbbreviation
		f.TimezoneIsDST = g.TimeZone.IsDST
	}

	if g.Security != nil {
		f.ThreatScore = g.Security.ThreatScore
		f.IsTor = g.Security.IsTor
		f.IsProxy = g.Security.IsProxy
		f.IsResidentialProxy = g.Security.IsResidentialProxy
		f.IsVPN = g.Security.IsVPN
		f.IsRelay = g.Security.IsRelay
		f.IsAnonymous = g.Security.IsAnonymous
		f.IsKnownAttacker = g.Security.IsKnownAttacker
		f.IsBot = g.Security.IsBot
		f.IsSpam = g.Security.IsSpam
		f.IsCloudProvider = g.Security.IsCloudProvider
		f.CloudProviderName = g.Security.CloudProviderName
	}

	if g.Abuse != nil {
		f.AbuseRoute = g.Abuse.Route
		f.AbuseName = g.Abuse.Name
		f.AbuseEmails = strings.Join(g.Abuse.Emails, ",")
		f.AbuseCountry = g.Abuse.Country
	}

	return f
}

// Ensure the flat struct is used (compiler check).
var _ fmt.Stringer = (*IPGeolocationFlat)(nil)

func (f *IPGeolocationFlat) String() string {
	return fmt.Sprintf("IPGeolocation{ip=%s, country=%s, city=%s}", f.IP, f.CountryName, f.City)
}
