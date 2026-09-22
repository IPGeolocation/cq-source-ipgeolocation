package services

import (
	"context"

	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"

	"github.com/IPGeolocation/cq-source-ipgeolocation/client"
	"github.com/IPGeolocation/cq-source-ipgeolocation/internal/ipgeolocation"
)

// AbuseContactFlat is a flattened abuse contact response for CloudQuery columns.
// It exposes every field of the API's `abuse` object (see AbuseFields).
type AbuseContactFlat struct {
	IP string `json:"ip"`
	AbuseFields
}

// AbuseContactTable returns the table definition for dedicated abuse contact lookups.
func AbuseContactTable() *schema.Table {
	return &schema.Table{
		Name:        "ipgeolocation_abuse_contact",
		Description: "Abuse contact information from IPGeolocation.io /v3/abuse endpoint. Returns the full abuse object: the abuse-handling route, registration country, contact name, organization, kind (group/individual), postal address, email addresses and phone numbers for each configured IP's network. Requires a paid plan. Costs 1 credit per lookup.",
		Resolver:    fetchAbuseContact,
		Transform: transformers.TransformWithStruct(&AbuseContactFlat{},
			transformers.WithPrimaryKeys("IP"),
			transformers.WithUnwrapAllEmbeddedStructs(),
		),
	}
}

func fetchAbuseContact(ctx context.Context, meta schema.ClientMeta, _ *schema.Resource, res chan<- any) error {
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

		resp, err := c.IPGeo.GetAbuseContact(ctx, ip)
		if err != nil {
			c.Logger.Warn().Str("ip", ip).Err(err).Msg("failed to fetch abuse contact, skipping")
			continue
		}
		if resp == nil || resp.Abuse == nil {
			continue
		}

		resolvedIP := resp.IP
		if resolvedIP == "" {
			resolvedIP = ip
		}

		flat := flattenAbuseContact(resolvedIP, resp.Abuse)
		res <- flat
	}

	return nil
}

func flattenAbuseContact(ip string, a *ipgeolocation.AbuseContact) *AbuseContactFlat {
	return &AbuseContactFlat{
		IP:          ip,
		AbuseFields: newAbuseFields(a),
	}
}
