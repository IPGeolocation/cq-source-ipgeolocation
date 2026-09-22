package ipgeolocation

// IPGeolocation represents the full response from the /v3/ipgeo endpoint.
type IPGeolocation struct {
	IP       string `json:"ip"`
	Domain   string `json:"domain,omitempty"`
	Hostname string `json:"hostname,omitempty"`

	Location        *Location        `json:"location,omitempty"`
	CountryMetadata *CountryMetadata `json:"country_metadata,omitempty"`
	Network         *Network         `json:"network,omitempty"`
	Currency        *Currency        `json:"currency,omitempty"`
	ASN             *ASN             `json:"asn,omitempty"`
	Company         *Company         `json:"company,omitempty"`
	Security        *Security        `json:"security,omitempty"`
	Abuse           *AbuseContact    `json:"abuse,omitempty"`
	TimeZone        *TimeZoneInfo    `json:"time_zone,omitempty"`
	UserAgent       *UserAgentInfo   `json:"user_agent,omitempty"`
}

// Location holds geographic location data.
type Location struct {
	ContinentCode      string `json:"continent_code"`
	ContinentName      string `json:"continent_name"`
	CountryCode2       string `json:"country_code2"`
	CountryCode3       string `json:"country_code3"`
	CountryName        string `json:"country_name"`
	CountryNameOfficial string `json:"country_name_official"`
	CountryCapital     string `json:"country_capital"`
	StateProv          string `json:"state_prov"`
	StateCode          string `json:"state_code"`
	District           string `json:"district"`
	City               string `json:"city"`
	Locality           string `json:"locality,omitempty"`
	AccuracyRadius     string `json:"accuracy_radius,omitempty"`
	Confidence         string `json:"confidence,omitempty"`
	DMACode            string `json:"dma_code,omitempty"`
	Zipcode            string `json:"zipcode"`
	Latitude           string `json:"latitude"`
	Longitude          string `json:"longitude"`
	IsEU               bool   `json:"is_eu"`
	CountryFlag        string `json:"country_flag"`
	GeonameID          string `json:"geoname_id"`
	CountryEmoji       string `json:"country_emoji"`
}

// CountryMetadata holds country-level metadata.
type CountryMetadata struct {
	CallingCode string   `json:"calling_code"`
	TLD         string   `json:"tld"`
	Languages   []string `json:"languages"`
}

// Network holds network/route information.
type Network struct {
	ConnectionType  string `json:"connection_type"`
	Route           string `json:"route"`
	IsAnycast       bool   `json:"is_anycast"`
	IsCDN           bool   `json:"is_cdn"`
	CDNProviderName string `json:"cdn_provider_name"`
}

// Currency holds the currency associated with the IP's country.
type Currency struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

// ASN holds Autonomous System Number information.
type ASN struct {
	ASNumber      string `json:"as_number"`
	Organization  string `json:"organization"`
	Country       string `json:"country"`
	Type          string `json:"type,omitempty"`
	Domain        string `json:"domain,omitempty"`
	DateAllocated string `json:"date_allocated,omitempty"`
	RIR           string `json:"rir,omitempty"`
}

// Company holds the company/ISP information for the IP.
type Company struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Domain string `json:"domain"`
}

// Security holds IP security / threat intelligence data.
//
// It mirrors the `security` object returned by /v3/security and by /v3/ipgeo
// (when include=security is passed). Field list reference:
// https://ipgeolocation.io/documentation/ip-security-api.html
type Security struct {
	ThreatScore int `json:"threat_score"`

	IsTor bool `json:"is_tor"`

	IsProxy              bool     `json:"is_proxy"`
	ProxyProviderNames   []string `json:"proxy_provider_names"`
	ProxyConfidenceScore int      `json:"proxy_confidence_score"`
	ProxyLastSeen        string   `json:"proxy_last_seen"`
	IsResidentialProxy   bool     `json:"is_residential_proxy"`

	IsVPN              bool     `json:"is_vpn"`
	VPNProviderNames   []string `json:"vpn_provider_names"`
	VPNConfidenceScore int      `json:"vpn_confidence_score"`
	VPNLastSeen        string   `json:"vpn_last_seen"`

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

// AbuseContact holds abuse contact details for an IP's network.
//
// It mirrors the `abuse` object returned by /v3/abuse and by /v3/ipgeo (when
// include=abuse is passed). Field list reference:
// https://ipgeolocation.io/documentation/ip-abuse-contact-api.html
type AbuseContact struct {
	Route        string   `json:"route"`
	Country      string   `json:"country"`
	Name         string   `json:"name"`
	Organization string   `json:"organization"`
	Kind         string   `json:"kind"`
	Address      string   `json:"address"`
	Emails       []string `json:"emails"`
	PhoneNumbers []string `json:"phone_numbers"`
}

// TimeZoneInfo holds timezone data returned within the ipgeo or timezone endpoint.
type TimeZoneInfo struct {
	Name                     string      `json:"name"`
	Offset                   float64     `json:"offset"`
	OffsetWithDST            float64     `json:"offset_with_dst"`
	CurrentTime              string      `json:"current_time"`
	CurrentTimeUnix          float64     `json:"current_time_unix"`
	CurrentTZAbbreviation    string      `json:"current_tz_abbreviation"`
	CurrentTZFullName        string      `json:"current_tz_full_name"`
	StandardTZAbbreviation   string      `json:"standard_tz_abbreviation"`
	StandardTZFullName       string      `json:"standard_tz_full_name"`
	IsDST                    bool        `json:"is_dst"`
	DSTSavings               float64     `json:"dst_savings"`
	DSTExists                bool        `json:"dst_exists"`
	DSTTZAbbreviation        string      `json:"dst_tz_abbreviation"`
	DSTTZFullName            string      `json:"dst_tz_full_name"`
	DSTStart                 *DSTDetails `json:"dst_start,omitempty"`
	DSTEnd                   *DSTDetails `json:"dst_end,omitempty"`
}

// DSTDetails holds the details for a DST transition.
type DSTDetails struct {
	UTCTime        string `json:"utc_time"`
	Duration       string `json:"duration"`
	Gap            bool   `json:"gap"`
	DateTimeAfter  string `json:"date_time_after"`
	DateTimeBefore string `json:"date_time_before"`
	Overlap        bool   `json:"overlap"`
}

// TimezoneResponse represents the full response from the /v3/timezone endpoint.
type TimezoneResponse struct {
	Timezone string       `json:"timezone"`
	IP       string       `json:"ip,omitempty"`
	Location *Location    `json:"location,omitempty"`
	TimeZone *TimeZoneInfo `json:"time_zone,omitempty"`
}

// AstronomyResponse represents the response from the /v3/astronomy endpoint.
type AstronomyResponse struct {
	Location  string       `json:"location,omitempty"`
	IP        string       `json:"ip,omitempty"`
	Date      string       `json:"date"`
	Sunrise   string       `json:"sunrise"`
	Sunset    string       `json:"sunset"`
	SolarNoon string       `json:"solar_noon"`
	DayLength string       `json:"day_length"`
	Moonrise  string       `json:"moonrise"`
	Moonset   string       `json:"moonset"`
	MoonPhase float64      `json:"moon_phase"`
	MoonIllumination float64 `json:"moon_illumination"`
	MoonAge   float64      `json:"moon_age"`
	MoonAngle float64      `json:"moon_angle"`
	MoonDistance float64    `json:"moon_distance"`
	MoonParallacticAngle float64 `json:"moon_parallactic_angle"`
	SunAltitude  float64   `json:"sun_altitude"`
	SunDistance   float64   `json:"sun_distance"`
	SunAzimuth   float64   `json:"sun_azimuth"`
	MoonAltitude float64   `json:"moon_altitude"`
	MoonAzimuth  float64   `json:"moon_azimuth"`
}

// UserAgentDevice holds device info from the user-agent parser.
type UserAgentDevice struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Brand string `json:"brand"`
	CPU   string `json:"cpu"`
}

// UserAgentEngine holds rendering engine info.
type UserAgentEngine struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Version      string `json:"version"`
	VersionMajor string `json:"version_major"`
}

// UserAgentOS holds OS info from the user-agent parser.
type UserAgentOS struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Version      string `json:"version"`
	VersionMajor string `json:"version_major"`
	Build        string `json:"build"`
}

// UserAgentInfo holds parsed user-agent details.
type UserAgentInfo struct {
	UserAgentString string           `json:"user_agent_string"`
	Name            string           `json:"name"`
	Type            string           `json:"type"`
	Version         string           `json:"version"`
	VersionMajor    string           `json:"version_major"`
	Device          *UserAgentDevice `json:"device,omitempty"`
	Engine          *UserAgentEngine `json:"engine,omitempty"`
	OperatingSystem *UserAgentOS     `json:"operating_system,omitempty"`
}

// UserAgentResponse represents the full response from /v3/user-agent.
type UserAgentResponse struct {
	UserAgentString string           `json:"user_agent_string"`
	Name            string           `json:"name"`
	Type            string           `json:"type"`
	Version         string           `json:"version"`
	VersionMajor    string           `json:"version_major"`
	Device          *UserAgentDevice `json:"device,omitempty"`
	Engine          *UserAgentEngine `json:"engine,omitempty"`
	OperatingSystem *UserAgentOS     `json:"operating_system,omitempty"`
}

// ASNRelation represents a peer, upstream, or downstream ASN.
type ASNRelation struct {
	ASNumber    string `json:"as_number"`
	Description string `json:"description"`
	Country     string `json:"country"`
}

// ASNDetail holds the detailed ASN response from the dedicated /v3/asn endpoint.
type ASNDetail struct {
	ASNumber         string `json:"as_number"`
	Organization     string `json:"organization"`
	Country          string `json:"country"`
	Type             string `json:"type"`
	Domain           string `json:"domain"`
	DateAllocated    string `json:"date_allocated"`
	ASNName          string `json:"asn_name"`
	AllocationStatus string `json:"allocation_status"`
	NumIPv4Routes    string `json:"num_of_ipv4_routes"`
	NumIPv6Routes    string `json:"num_of_ipv6_routes"`
	RIR              string `json:"rir"`

	// Optional modules: only present when requested through the `include`
	// query parameter (see ASNIncludeOptions).
	Routes        []string      `json:"routes,omitempty"`
	Peers         []ASNRelation `json:"peers,omitempty"`
	Upstreams     []ASNRelation `json:"upstreams,omitempty"`
	Downstreams   []ASNRelation `json:"downstreams,omitempty"`
	WhoisResponse string        `json:"whois_response,omitempty"`
}

// ASNInclude* are the optional modules the /v3/asn endpoint can add to its
// default response through the `include` query parameter. Without any of them
// the API returns only the base ASN fields.
const (
	ASNIncludePeers         = "peers"
	ASNIncludeUpstreams     = "upstreams"
	ASNIncludeDownstreams   = "downstreams"
	ASNIncludeRoutes        = "routes"
	ASNIncludeWhoisResponse = "whois_response"
)

// ASNIncludeOptions lists every value accepted by the /v3/asn `include`
// parameter, in a stable canonical order.
var ASNIncludeOptions = []string{
	ASNIncludePeers,
	ASNIncludeUpstreams,
	ASNIncludeDownstreams,
	ASNIncludeRoutes,
	ASNIncludeWhoisResponse,
}

// ASNResponse is the top-level response from /v3/asn.
type ASNResponse struct {
	IP  string     `json:"ip,omitempty"`
	ASN *ASNDetail `json:"asn"`
}
