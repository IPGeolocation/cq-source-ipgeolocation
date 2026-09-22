package client

import (
	"fmt"
	"strings"
	"time"

	"github.com/IPGeolocation/cq-source-ipgeolocation/internal/ipgeolocation"
)

const (
	defaultEndpoint      = "https://api.ipgeolocation.io"
	defaultTimeout       = 30 * time.Second
	defaultRetryAttempts = 3
	defaultConcurrency   = 10
	defaultRateLimit     = 15 // requests per second
	defaultUserAgent     = "cq-source-ipgeolocation/1.0"
)

// Spec holds the configuration for the IPGeolocation source plugin.
// It is parsed from the user's YAML configuration.
type Spec struct {
	// APIKey is the IPGeolocation.io API key used for authentication.
	APIKey string `json:"api_key"`

	// Endpoint is the base URL for the IPGeolocation.io API.
	// Defaults to "https://api.ipgeolocation.io".
	Endpoint string `json:"endpoint"`

	// Timeout is the HTTP client timeout as a duration string (e.g. "30s").
	// Defaults to "30s".
	Timeout string `json:"timeout"`

	// RetryAttempts is the number of times to retry a failed API request.
	// Defaults to 3.
	RetryAttempts int `json:"retry_attempts"`

	// Concurrency controls how many tables are synced in parallel.
	// Defaults to 10.
	Concurrency int `json:"concurrency"`

	// RateLimit is the maximum number of API requests per second.
	// Defaults to 15.
	RateLimit int `json:"rate_limit"`

	// UserAgent is the User-Agent header sent with every request.
	// Defaults to "cq-source-ipgeolocation/1.0".
	UserAgent string `json:"user_agent"`

	// IPs is a list of IP addresses to look up using the IP Geolocation endpoint.
	// If empty, the plugin looks up the caller's own public IP.
	IPs []string `json:"ips"`

	// UserAgents is a list of user-agent strings to parse via the User-Agent API.
	UserAgents []string `json:"user_agents"`

	// ASNs is a list of AS numbers (e.g. "24940", "15169") to look up via the dedicated ASN API.
	ASNs []string `json:"asns"`

	// IncludeSecurity adds IP security/threat data to geolocation results (paid plans).
	IncludeSecurity bool `json:"include_security"`

	// IncludeAbuse adds abuse contact data to geolocation results (paid plans).
	IncludeAbuse bool `json:"include_abuse"`

	// IncludeGeoAccuracy adds the location accuracy fields (locality,
	// accuracy_radius, confidence) to geolocation results (paid plans).
	IncludeGeoAccuracy bool `json:"include_geo_accuracy"`

	// IncludeDMACode adds the DMA (Designated Market Area) code to geolocation
	// results (paid plans). Only populated for US locations.
	IncludeDMACode bool `json:"include_dma_code"`

	// IncludeHostname adds the reverse-DNS hostname to geolocation results
	// (paid plans). How it is resolved is controlled by HostnameLookup.
	IncludeHostname bool `json:"include_hostname"`

	// HostnameLookup selects how the hostname is resolved when IncludeHostname
	// is true: "database" (fast, from IPGeolocation.io's hostname database, but
	// experimental), "live" (accurate live lookup, adds latency) or
	// "fallback_live" (database first, live lookup if not found).
	// Defaults to "fallback_live".
	HostnameLookup string `json:"hostname_lookup"`

	// ASNInclude selects which optional modules the ipgeolocation_asn table
	// requests from the /v3/asn endpoint. Allowed values: "peers", "upstreams",
	// "downstreams", "routes", "whois_response", or "*" for all of them.
	//
	// Defaults to empty: only the base ASN fields are fetched. Peers, upstreams,
	// downstreams, routes and WHOIS text can be very large for well-connected
	// ASNs, so they are strictly opt-in.
	ASNInclude []string `json:"asn_include"`
}

// SetDefaults populates zero-valued fields with sensible defaults.
func (s *Spec) SetDefaults() {
	if s.Endpoint == "" {
		s.Endpoint = defaultEndpoint
	}
	if s.Timeout == "" {
		s.Timeout = defaultTimeout.String()
	}
	if s.RetryAttempts == 0 {
		s.RetryAttempts = defaultRetryAttempts
	}
	if s.Concurrency == 0 {
		s.Concurrency = defaultConcurrency
	}
	if s.RateLimit == 0 {
		s.RateLimit = defaultRateLimit
	}
	if s.UserAgent == "" {
		s.UserAgent = defaultUserAgent
	}
}

// Validate checks required fields and returns a descriptive error if any are missing or invalid.
func (s *Spec) Validate() error {
	if s.APIKey == "" {
		return fmt.Errorf("api_key is required: get one from https://app.ipgeolocation.io/signup")
	}
	if _, err := time.ParseDuration(s.Timeout); err != nil {
		return fmt.Errorf("invalid timeout %q: %w", s.Timeout, err)
	}
	if s.RetryAttempts < 0 {
		return fmt.Errorf("retry_attempts must be >= 0, got %d", s.RetryAttempts)
	}
	if s.Concurrency < 1 {
		return fmt.Errorf("concurrency must be >= 1, got %d", s.Concurrency)
	}
	if s.RateLimit < 1 {
		return fmt.Errorf("rate_limit must be >= 1, got %d", s.RateLimit)
	}
	if _, err := s.ASNIncludeModules(); err != nil {
		return err
	}
	if _, err := s.hostnameAPIValue(); err != nil {
		return err
	}
	return nil
}

// Accepted hostname_lookup values and the /v3/ipgeo include value each maps to.
const (
	hostnameLookupDatabase     = "database"
	hostnameLookupLive         = "live"
	hostnameLookupFallbackLive = "fallback_live"
)

// hostnameAPIValue maps hostname_lookup to the API's include value, applying
// the default and rejecting unknown modes (regardless of include_hostname, so a
// typo is caught early).
func (s *Spec) hostnameAPIValue() (string, error) {
	switch strings.ToLower(strings.TrimSpace(s.HostnameLookup)) {
	case "", hostnameLookupFallbackLive:
		return "hostnameFallbackLive", nil
	case hostnameLookupDatabase:
		return "hostname", nil
	case hostnameLookupLive:
		return "liveHostname", nil
	default:
		return "", fmt.Errorf("invalid hostname_lookup %q: valid values are %s, %s, %s",
			s.HostnameLookup, hostnameLookupDatabase, hostnameLookupLive, hostnameLookupFallbackLive)
	}
}

// IPGeolocationIncludes returns the optional /v3/ipgeo modules to request for
// the ipgeolocation_ip_geolocation table, in a stable order: security, abuse,
// geo_accuracy, dma_code, then the hostname mode. It returns nil when none is enabled.
func (s *Spec) IPGeolocationIncludes() ([]string, error) {
	hostname, err := s.hostnameAPIValue()
	if err != nil {
		return nil, err
	}

	var includes []string
	if s.IncludeSecurity {
		includes = append(includes, "security")
	}
	if s.IncludeAbuse {
		includes = append(includes, "abuse")
	}
	if s.IncludeGeoAccuracy {
		includes = append(includes, "geo_accuracy")
	}
	if s.IncludeDMACode {
		includes = append(includes, "dma_code")
	}
	if s.IncludeHostname {
		includes = append(includes, hostname)
	}
	return includes, nil
}

// ASNIncludeModules returns the normalised list of optional /v3/asn modules
// selected through asn_include: lower-cased, de-duplicated, "*" expanded, and
// in canonical order. It returns nil when nothing is selected, and an error
// naming the valid options when an unknown value is configured.
func (s *Spec) ASNIncludeModules() ([]string, error) {
	selected := make(map[string]bool, len(ipgeolocation.ASNIncludeOptions))
	for _, raw := range s.ASNInclude {
		v := strings.ToLower(strings.TrimSpace(raw))
		switch {
		case v == "":
			continue
		case v == "*":
			for _, opt := range ipgeolocation.ASNIncludeOptions {
				selected[opt] = true
			}
		case isASNIncludeOption(v):
			selected[v] = true
		default:
			return nil, fmt.Errorf("invalid asn_include value %q: valid values are %s, or \"*\" for all",
				raw, strings.Join(ipgeolocation.ASNIncludeOptions, ", "))
		}
	}

	var out []string
	for _, opt := range ipgeolocation.ASNIncludeOptions {
		if selected[opt] {
			out = append(out, opt)
		}
	}
	return out, nil
}

func isASNIncludeOption(v string) bool {
	for _, opt := range ipgeolocation.ASNIncludeOptions {
		if v == opt {
			return true
		}
	}
	return false
}

// TimeoutDuration parses and returns the configured timeout.
func (s *Spec) TimeoutDuration() time.Duration {
	d, err := time.ParseDuration(s.Timeout)
	if err != nil {
		return defaultTimeout
	}
	return d
}
