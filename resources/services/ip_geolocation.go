package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"

	"github.com/IPGeolocation/cq-source-ipgeolocation/client"
	"github.com/IPGeolocation/cq-source-ipgeolocation/internal/ipgeolocation"
)

// IPGeolocationFlat is a flattened representation of the IP Geolocation API response.
// CloudQuery tables work best with flat structs rather than deeply nested JSON.
type IPGeolocationFlat struct {
	// Core
	IP     string `json:"ip"`
	Domain string `json:"domain"`

	// Hostname is only requested when include_hostname is true (resolution mode
	// set by hostname_lookup); NULL otherwise. If the hostname cannot be
	// resolved the API returns the queried IP itself.
	Hostname *string `json:"hostname"`

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

	// Geo accuracy (optional; only requested when include_geo_accuracy is
	// true, NULL otherwise): locality/neighbourhood, accuracy radius in km
	// around latitude/longitude, and confidence (low, medium, high).
	Locality       *string `json:"locality"`
	AccuracyRadius *string `json:"accuracy_radius"`
	Confidence     *string `json:"confidence"`

	// DMACode is the Designated Market Area code (optional; only requested when
	// include_dma_code is true, NULL otherwise). The API only fills it for US
	// locations, so it is an empty string elsewhere.
	DMACode *string `json:"dma_code"`

	// Country metadata
	CallingCode string `json:"calling_code"`
	TLD         string `json:"tld"`
	Languages   string `json:"languages"` // comma-separated

	// Network
	ConnectionType  string `json:"connection_type"`
	Route           string `json:"route"`
	IsAnycast       bool   `json:"is_anycast"`
	IsCDN           bool   `json:"is_cdn"`
	CDNProviderName string `json:"cdn_provider_name"`

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
	TimezoneName            string  `json:"timezone_name"`
	TimezoneOffset          float64 `json:"timezone_offset"`
	TimezoneOffsetWithDST   float64 `json:"timezone_offset_with_dst"`
	TimezoneCurrentTime     string  `json:"timezone_current_time"`
	TimezoneCurrentTimeUnix float64 `json:"timezone_current_time_unix"`
	TimezoneAbbreviation    string  `json:"timezone_abbreviation"`
	TimezoneIsDST           bool    `json:"timezone_is_dst"`

	// Security data (optional; only requested when include_security is true).
	// Exposes every field of the API's `security` object as unprefixed columns
	// (threat_score, is_vpn, bot_type, is_corporate_gateway, ...). The pointer is
	// nil when the data was not requested, which makes all of these columns NULL
	// instead of a misleading false/0.
	*SecurityFields

	// Abuse contact data (optional; only requested when include_abuse is true).
	// Exposes every field of the API's `abuse` object as abuse_-prefixed columns
	// (abuse_route, abuse_country, abuse_name, abuse_organization, abuse_kind,
	// abuse_address, abuse_emails, abuse_phone_numbers). nil => NULL columns.
	Abuse *AbuseFields `json:"abuse"`
}

// IPGeolocationTable returns the table definition for IP geolocation lookups.
func IPGeolocationTable() *schema.Table {
	return &schema.Table{
		Name:        "ipgeolocation_ip_geolocation",
		Description: "IP Geolocation data from IPGeolocation.io /v3/ipgeo endpoint. Returns location, ASN, company, timezone, network, currency, and optionally the full security object (include_security) and the full abuse contact object (include_abuse) for each configured IP address. Security and abuse columns are NULL when not requested.",
		Resolver:    fetchIPGeolocation,
		Transform: transformers.TransformWithStruct(&IPGeolocationFlat{},
			transformers.WithPrimaryKeys("IP"),
			transformers.WithUnwrapAllEmbeddedStructs(),
			transformers.WithUnwrapStructFields("Abuse"),
			transformers.WithResolverTransformer(nullSafeResolver),
		),
	}
}

func fetchIPGeolocation(ctx context.Context, meta schema.ClientMeta, _ *schema.Resource, res chan<- any) error {
	c := meta.(*client.Client)

	ips := c.Spec.IPs
	if len(ips) == 0 {
		// No IPs configured: look up the caller's own IP.
		ips = []string{""}
	}

	includes, err := c.Spec.IPGeolocationIncludes()
	if err != nil {
		return err
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

		flat := flattenIPGeolocation(geo, geoOptions{
			GeoAccuracy: c.Spec.IncludeGeoAccuracy,
			DMACode:     c.Spec.IncludeDMACode,
			Hostname:    c.Spec.IncludeHostname,
		})
		res <- flat
	}

	return nil
}

// geoOptions says which optional /v3/ipgeo location modules were requested.
type geoOptions struct {
	GeoAccuracy bool
	DMACode     bool
	Hostname    bool
}

// flattenIPGeolocation converts the nested API response into a flat struct.
//
// opts mirrors the spec options: the matching columns are only populated when
// the module was requested, so NULL means "not fetched".
func flattenIPGeolocation(g *ipgeolocation.IPGeolocation, opts geoOptions) *IPGeolocationFlat {
	f := &IPGeolocationFlat{
		IP:     g.IP,
		Domain: g.Domain,
	}
	if opts.Hostname {
		f.Hostname = strPtr(g.Hostname)
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
		if opts.GeoAccuracy {
			f.Locality = strPtr(g.Location.Locality)
			f.AccuracyRadius = strPtr(g.Location.AccuracyRadius)
			f.Confidence = strPtr(g.Location.Confidence)
		}
		if opts.DMACode {
			f.DMACode = strPtr(g.Location.DMACode)
		}
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
		f.IsCDN = g.Network.IsCDN
		f.CDNProviderName = g.Network.CDNProviderName
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
		sec := newSecurityFields(g.Security)
		f.SecurityFields = &sec
	}

	if g.Abuse != nil {
		ab := newAbuseFields(g.Abuse)
		f.Abuse = &ab
	}

	return f
}

// Ensure the flat struct is used (compiler check).
var _ fmt.Stringer = (*IPGeolocationFlat)(nil)

func (f *IPGeolocationFlat) String() string {
	return fmt.Sprintf("IPGeolocation{ip=%s, country=%s, city=%s}", f.IP, f.CountryName, f.City)
}
