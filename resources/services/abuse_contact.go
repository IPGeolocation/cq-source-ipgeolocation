package services

import (
	"context"

	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"

	"github.com/IPGeolocation/cq-source-ipgeolocation/client"
	"github.com/IPGeolocation/cq-source-ipgeolocation/internal/ipgeolocation"
)

// AbuseContactFlat is a flattened abuse contact response for CloudQuery columns.
type AbuseContactFlat struct {
	IP           string `json:"ip"`
	Route        string `json:"route"`
	Country      string `json:"country"`
	Name         string `json:"name"`
	Organization string `json:"organization"`
	Kind         string `json:"kind"`
	Address      string `json:"address"`
	Emails       string `json:"emails"`
	PhoneNumbers string `json:"phone_numbers"`
}

// AbuseContactTable returns the table definition for dedicated abuse contact lookups.
func AbuseContactTable() *schema.Table {
	return &schema.Table{
		Name:        "ipgeolocation_abuse_contact",
		Description: "Abuse contact information from IPGeolocation.io /v3/abuse endpoint. Returns the responsible abuse contact name, email addresses, phone numbers, and postal address for each configured IP's network. Requires a paid plan. Costs 1 credit per lookup.",
		Resolver:    fetchAbuseContact,
		Transform:   transformers.TransformWithStruct(&AbuseContactFlat{}, transformers.WithPrimaryKeys("IP")),
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
		IP:           ip,
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
