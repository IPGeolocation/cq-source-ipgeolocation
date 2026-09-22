package services

import (
	"context"

	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"

	"github.com/IPGeolocation/cq-source-ipgeolocation/client"
	"github.com/IPGeolocation/cq-source-ipgeolocation/internal/ipgeolocation"
)

// IPSecurityFlat is a flattened security/threat response for CloudQuery columns.
// It exposes every field of the API's `security` object (see SecurityFields).
type IPSecurityFlat struct {
	IP string `json:"ip"`
	SecurityFields
}

// IPSecurityTable returns the table definition for dedicated IP security lookups.
func IPSecurityTable() *schema.Table {
	return &schema.Table{
		Name:        "ipgeolocation_ip_security",
		Description: "IP security and threat intelligence from IPGeolocation.io /v3/security endpoint. Returns the full security object: threat score; VPN, proxy, residential proxy, Tor and relay detection with provider names, confidence scores and last-seen dates; bot classification (type, operator, known-good-bot flag); spam and known-attacker flags; cloud provider and corporate gateway attribution. Requires a paid plan. Costs 2 credits per lookup.",
		Resolver:    fetchIPSecurity,
		Transform: transformers.TransformWithStruct(&IPSecurityFlat{},
			transformers.WithPrimaryKeys("IP"),
			transformers.WithUnwrapAllEmbeddedStructs(),
		),
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
		IP:             ip,
		SecurityFields: newSecurityFields(s),
	}
}
