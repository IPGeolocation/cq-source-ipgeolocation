package services

import (
	"context"
	"strings"

	"github.com/cloudquery/plugin-sdk/v4/schema"
	"github.com/cloudquery/plugin-sdk/v4/transformers"

	"github.com/IPGeolocation/cq-source-ipgeolocation/client"
	"github.com/IPGeolocation/cq-source-ipgeolocation/internal/ipgeolocation"
)

// ASNDetailFlat is a flattened ASN detail response for CloudQuery columns.
// The dedicated /v3/asn endpoint returns richer data than the basic asn object
// inside /v3/ipgeo, including routes, peers, upstreams, and downstreams.
type ASNDetailFlat struct {
	QueryIP          string `json:"query_ip"`
	QueryASN         string `json:"query_asn"`
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
	Routes           string `json:"routes"`      // comma-separated CIDR prefixes
	Peers            string `json:"peers"`       // comma-separated AS numbers
	Upstreams        string `json:"upstreams"`   // comma-separated AS numbers
	Downstreams      string `json:"downstreams"` // comma-separated AS numbers
}

// ASNDetailTable returns the table definition for detailed ASN lookups.
func ASNDetailTable() *schema.Table {
	return &schema.Table{
		Name:        "ipgeolocation_asn",
		Description: "Detailed ASN (Autonomous System Number) data from IPGeolocation.io /v3/asn endpoint. Returns the AS name, organization, allocation status, IPv4/IPv6 route counts, and network relationships (peers, upstreams, downstreams, routes) for each configured IP or ASN. Costs 1 credit per lookup.",
		Resolver:    fetchASNDetail,
		Transform:   transformers.TransformWithStruct(&ASNDetailFlat{}, transformers.WithPrimaryKeys("QueryIP", "QueryASN")),
	}
}

func fetchASNDetail(ctx context.Context, meta schema.ClientMeta, _ *schema.Resource, res chan<- any) error {
	c := meta.(*client.Client)

	// Look up ASN details for each configured IP.
	ips := c.Spec.IPs
	if len(ips) == 0 && len(c.Spec.ASNs) == 0 {
		ips = []string{""}
	}

	for _, ip := range ips {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := c.IPGeo.GetASNByIP(ctx, ip)
		if err != nil {
			c.Logger.Warn().Str("ip", ip).Err(err).Msg("failed to fetch ASN detail, skipping")
			continue
		}
		if resp.ASN == nil {
			continue
		}

		flat := flattenASNDetail(ip, "", resp.ASN)
		res <- flat
	}

	// Look up ASN details for each configured ASN number.
	for _, asn := range c.Spec.ASNs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := c.IPGeo.GetASNByNumber(ctx, asn)
		if err != nil {
			c.Logger.Warn().Str("asn", asn).Err(err).Msg("failed to fetch ASN detail, skipping")
			continue
		}
		if resp.ASN == nil {
			continue
		}

		flat := flattenASNDetail("", asn, resp.ASN)
		res <- flat
	}

	return nil
}

func flattenASNDetail(ip, queryASN string, a *ipgeolocation.ASNDetail) *ASNDetailFlat {
	f := &ASNDetailFlat{
		QueryIP:          ip,
		QueryASN:         queryASN,
		ASNumber:         a.ASNumber,
		Organization:     a.Organization,
		Country:          a.Country,
		Type:             a.Type,
		Domain:           a.Domain,
		DateAllocated:    a.DateAllocated,
		ASNName:          a.ASNName,
		AllocationStatus: a.AllocationStatus,
		NumIPv4Routes:    a.NumIPv4Routes,
		NumIPv6Routes:    a.NumIPv6Routes,
		RIR:              a.RIR,
		Routes:           strings.Join(a.Routes, ","),
	}

	peerNums := make([]string, len(a.Peers))
	for i, p := range a.Peers {
		peerNums[i] = p.ASNumber
	}
	f.Peers = strings.Join(peerNums, ",")

	upNums := make([]string, len(a.Upstreams))
	for i, u := range a.Upstreams {
		upNums[i] = u.ASNumber
	}
	f.Upstreams = strings.Join(upNums, ",")

	downNums := make([]string, len(a.Downstreams))
	for i, d := range a.Downstreams {
		downNums[i] = d.ASNumber
	}
	f.Downstreams = strings.Join(downNums, ",")

	return f
}

// joinStrings is a shared helper to join string slices with commas.
func joinStrings(ss []string) string {
	return strings.Join(ss, ",")
}
