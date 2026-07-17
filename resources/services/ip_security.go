package services

import (
	"context"

	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"

	"github.com/anthropic/cq-source-ipgeolocation/client"
	"github.com/anthropic/cq-source-ipgeolocation/internal/ipgeolocation"
)

// IPSecurityFlat is a flattened security/threat response for CloudQuery columns.
type IPSecurityFlat struct {
	IP                   string `json:"ip"`
	ThreatScore          int    `json:"threat_score"`
	IsTor                bool   `json:"is_tor"`
	IsProxy              bool   `json:"is_proxy"`
	ProxyProviderNames   string `json:"proxy_provider_names"`
	ProxyConfidenceScore int    `json:"proxy_confidence_score"`
	ProxyLastSeen        string `json:"proxy_last_seen"`
	IsResidentialProxy   bool   `json:"is_residential_proxy"`
	IsVPN                bool   `json:"is_vpn"`
	VPNProviderNames     string `json:"vpn_provider_names"`
	VPNConfidenceScore   int    `json:"vpn_confidence_score"`
	VPNLastSeen          string `json:"vpn_last_seen"`
	IsRelay              bool   `json:"is_relay"`
	RelayProviderName    string `json:"relay_provider_name"`
	IsAnonymous          bool   `json:"is_anonymous"`
	IsKnownAttacker      bool   `json:"is_known_attacker"`
	IsBot                bool   `json:"is_bot"`
	IsSpam               bool   `json:"is_spam"`
	IsCloudProvider      bool   `json:"is_cloud_provider"`
	CloudProviderName    string `json:"cloud_provider_name"`
}

// IPSecurityTable returns the table definition for dedicated IP security lookups.
func IPSecurityTable() *schema.Table {
	return &schema.Table{
		Name:        "ipgeolocation_ip_security",
		Description: "IP security and threat intelligence from IPGeolocation.io /v3/security endpoint. Returns threat score, VPN/proxy/Tor detection, bot classification, and provider attribution for each configured IP. Requires a paid plan. Costs 2 credits per lookup.",
		Resolver:    fetchIPSecurity,
		Transform:   transformers.TransformWithStruct(&IPSecurityFlat{}, transformers.WithPrimaryKeys("IP")),
	}
}

func fetchIPSecurity(ctx context.Context, meta schema.ClientMeta, _ *schema.Resource, res chan<- any) error {
	c := meta.(*client.Client)

	ips := c.Spec.IPs
	if len(ips) == 0 {
		ips = []string{""}
	}

	for _, ip := range ips {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := c.IPGeo.GetSecurity(ctx, ip)
		if err != nil {
			c.Logger.Warn().Str("ip", ip).Err(err).Msg("failed to fetch IP security, skipping")
			continue
		}
		if resp == nil || resp.Security == nil {
			continue
		}

		// Use the IP from the API response (handles caller-IP auto-detection).
		resolvedIP := resp.IP
		if resolvedIP == "" {
			resolvedIP = ip
		}

		flat := flattenSecurity(resolvedIP, resp.Security)
		res <- flat
	}

	return nil
}

func flattenSecurity(ip string, s *ipgeolocation.Security) *IPSecurityFlat {
	return &IPSecurityFlat{
		IP:                   ip,
		ThreatScore:          s.ThreatScore,
		IsTor:                s.IsTor,
		IsProxy:              s.IsProxy,
		ProxyProviderNames:   joinStrings(s.ProxyProviderNames),
		ProxyConfidenceScore: s.ProxyConfidenceScore,
		ProxyLastSeen:        s.ProxyLastSeen,
		IsResidentialProxy:   s.IsResidentialProxy,
		IsVPN:                s.IsVPN,
		VPNProviderNames:     joinStrings(s.VPNProviderNames),
		VPNConfidenceScore:   s.VPNConfidenceScore,
		VPNLastSeen:          s.VPNLastSeen,
		IsRelay:              s.IsRelay,
		RelayProviderName:    s.RelayProviderName,
		IsAnonymous:          s.IsAnonymous,
		IsKnownAttacker:      s.IsKnownAttacker,
		IsBot:                s.IsBot,
		IsSpam:               s.IsSpam,
		IsCloudProvider:      s.IsCloudProvider,
		CloudProviderName:    s.CloudProviderName,
	}
}
