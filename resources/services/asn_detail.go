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
//
// The base columns are always populated. The last five (routes, peers,
// upstreams, downstreams, whois_response) are optional modules of the /v3/asn
// endpoint that can be very large, so they are fetched only when listed in the
// asn_include spec option. They are NULL when not requested, and an empty
// string when requested but the ASN has none.
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

	// Optional modules (asn_include). NULL unless requested.
	Routes        *string `json:"routes"`         // comma-separated CIDR prefixes
	Peers         *string `json:"peers"`          // comma-separated AS numbers
	Upstreams     *string `json:"upstreams"`      // comma-separated AS numbers
	Downstreams   *string `json:"downstreams"`    // comma-separated AS numbers
	WhoisResponse *string `json:"whois_response"` // raw WHOIS record text
}

// ASNDetailTable returns the table definition for detailed ASN lookups.
func ASNDetailTable() *schema.Table {
	return &schema.Table{
		Name:        "ipgeolocation_asn",
		Description: "Detailed ASN (Autonomous System Number) data from IPGeolocation.io /v3/asn endpoint. Returns the AS name, organization, type, allocation status and IPv4/IPv6 route counts for each configured IP or ASN. The large network-relationship modules (peers, upstreams, downstreams, routes) and the raw WHOIS text are opt-in: list the ones you want in the asn_include spec option; the matching columns are NULL otherwise. Costs 1 credit per lookup.",
		Resolver:    fetchASNDetail,
		Transform: transformers.TransformWithStruct(&ASNDetailFlat{},
			transformers.WithPrimaryKeys("QueryIP", "QueryASN"),
			transformers.WithResolverTransformer(nullSafeResolver),
		),
	}
}

func fetchASNDetail(ctx context.Context, meta schema.ClientMeta, _ *schema.Resource, res chan<- any) error {
	c := meta.(*client.Client)

	// Optional modules to request (peers, upstreams, ...). Empty by default so
	// the API returns only the compact base response.
	includes, err := c.Spec.ASNIncludeModules()
	if err != nil {
		return err
	}

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

		resp, err := c.IPGeo.GetASNByIP(ctx, ip, includes...)
		if err != nil {
			c.Logger.Warn().Str("ip", ip).Err(err).Msg("failed to fetch ASN detail, skipping")
			continue
		}
		if resp.ASN == nil {
			continue
		}

		flat := flattenASNDetail(ip, "", resp.ASN, includes)
		res <- flat
	}

	// Look up ASN details for each configured ASN number.
	for _, asn := range c.Spec.ASNs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := c.IPGeo.GetASNByNumber(ctx, asn, includes...)
		if err != nil {
			c.Logger.Warn().Str("asn", asn).Err(err).Msg("failed to fetch ASN detail, skipping")
			continue
		}
		if resp.ASN == nil {
			continue
		}

		flat := flattenASNDetail("", asn, resp.ASN, includes)
		res <- flat
	}

	return nil
}

func flattenASNDetail(ip, queryASN string, a *ipgeolocation.ASNDetail, includes []string) *ASNDetailFlat {
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
	}

	// Only populate the optional modules that were requested; everything else
	// stays nil (NULL) so "not fetched" is distinguishable from "none".
	for _, inc := range includes {
		switch inc {
		case ipgeolocation.ASNIncludeRoutes:
			f.Routes = strPtr(strings.Join(a.Routes, ","))
		case ipgeolocation.ASNIncludePeers:
			f.Peers = strPtr(joinASNumbers(a.Peers))
		case ipgeolocation.ASNIncludeUpstreams:
			f.Upstreams = strPtr(joinASNumbers(a.Upstreams))
		case ipgeolocation.ASNIncludeDownstreams:
			f.Downstreams = strPtr(joinASNumbers(a.Downstreams))
		case ipgeolocation.ASNIncludeWhoisResponse:
			f.WhoisResponse = strPtr(a.WhoisResponse)
		}
	}

	return f
}

// joinASNumbers joins the AS numbers of a peer/upstream/downstream list with commas.
func joinASNumbers(rels []ipgeolocation.ASNRelation) string {
	nums := make([]string, len(rels))
	for i, r := range rels {
		nums[i] = r.ASNumber
	}
	return strings.Join(nums, ",")
}

// joinStrings is a shared helper to join string slices with commas.
func joinStrings(ss []string) string {
	return strings.Join(ss, ",")
}
