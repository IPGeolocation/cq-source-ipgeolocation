# Troubleshooting

## The Sync Fails Before Any Request

Validation runs before the first API call, so these failures cost no credits.

| Message | Cause |
|---|---|
| `api_key is required` | No `api_key` in the spec, or the environment variable it references is unset |
| `invalid timeout` | `timeout` is not a Go duration. Use `30s` or `1m`, not `30` |
| `retry_attempts must be >= 0` | Negative value |
| `concurrency must be >= 1` | Zero or negative value |
| `rate_limit must be >= 1` | Zero or negative value |
| `invalid asn_include value` | A value outside `peers`, `upstreams`, `downstreams`, `routes`, `whois_response` and `"*"` |
| `invalid hostname_lookup` | A value outside `database`, `live` and `fallback_live`. Checked even when `include_hostname` is false |

If `${IPGEOLOCATION_API_KEY}` reaches the plugin unexpanded, the shell running `cloudquery sync` did not have the variable exported. Confirm with `echo $IPGEOLOCATION_API_KEY` in the same shell.

## API Error Codes

| Status | Meaning | What to do |
|---|---|---|
| 400 | Malformed request, usually an invalid IP or AS number | Check the entry the log names |
| 401 | Invalid key, or an endpoint your plan does not include | Confirm the key, then check whether the table needs a paid plan |
| 403 | Key is disabled or the request came from a blocked origin | Check the key's status in the dashboard |
| 404 | No data for the queried resource | Expected for some IPv6 prefixes and unallocated AS numbers |
| 423 | Bogon address: private, loopback, reserved or multicast | Remove it from `ips`. These ranges have no public location |
| 429 | Rate limit or daily quota exhausted | Lower `rate_limit`, or check the dashboard for remaining credits |
| 5xx | Server side fault | Retried automatically. Persistent failures are worth reporting |

The plugin retries 429, 500, 502, 503 and 504 along with network failures, backing off 1 second, then 2, then 4. Other 4xx responses are returned at once, because a 400 or a 401 will not improve on a second attempt.

## A Table Is Empty After a Successful Sync

A failed lookup is logged as a warning and skipped so the rest of the sync continues. An empty table therefore usually means every lookup failed for the same reason.

Run the sync with `--log-level debug` and look for `failed to fetch` lines, which name the IP and the underlying error.

Common causes:

- `ipgeolocation_user_agent` is empty because `user_agents` is empty. The table is skipped entirely in that case, without an error.
- The paid tables are empty on a free plan. The API returns 401 for each IP, each one is logged and skipped, and the sync still reports success.
- Every IP in the list is private or reserved. The API answers 423 for all of them.
- `asns` is set but `ipgeolocation_asn` is not in the `tables` list, so no ASN lookups ran.

## Columns Are NULL When You Expected Values

NULL means the module was never requested. Check the matching spec flag:

| NULL columns | Set this |
|---|---|
| `threat_score`, `is_vpn`, `is_proxy`, `bot_type` and the rest of the security block | `include_security: true` |
| `abuse_route`, `abuse_emails`, `abuse_phone_numbers` and the rest of the abuse block | `include_abuse: true` |
| `locality`, `accuracy_radius`, `confidence` | `include_geo_accuracy: true` |
| `dma_code` | `include_dma_code: true` |
| `hostname` | `include_hostname: true` |
| `peers`, `upstreams`, `downstreams`, `routes`, `whois_response` | `asn_include` |

An empty string is different. It means the module ran and the API had nothing to return, which is normal for `dma_code` outside the US, for `locality` on some IPs, and for `upstreams` on a tier 1 network that buys transit from nobody.

## The ASN Table Is Slow or Enormous

`asn_include: ["*"]` pulls peers, upstreams, downstreams, every announced prefix and the raw WHOIS record for each ASN. Large transit networks have route lists in the tens of thousands, and all of it lands in one text column.

Name only the modules you query. Most work needs `upstreams` and `downstreams`, or nothing at all.

## Hitting Rate Limits

`rate_limit` is a token bucket in requests per second, with the burst set to the same number. The default of 15 suits most paid plans. Lower it if your plan is tighter or if other systems share the key.

`concurrency` controls how many tables sync at once and does not raise your request rate, since the limiter applies globally.

Remember that a sync can spend more credits than it makes requests. Five IPs against `ipgeolocation_ip_security` is 5 requests but 10 credits.

## Duplicate or Missing Rows in the ASN Table

`ipgeolocation_asn` is keyed on `query_ip` and `query_asn` together. Looking up an IP and separately looking up the AS number that IP belongs to produces two rows describing the same ASN, one with `query_ip` filled and one with `query_asn` filled. That is intended: it records how you asked, not just what came back.

If you want one row per ASN, query by AS number only and leave `ips` out, or deduplicate on `as_number` at query time.

## Verifying the Connection

The smallest possible check is one IP against the geolocation table on a free key:

```yaml
kind: source
spec:
  name: ipgeolocation
  registry: grpc
  path: "localhost:7777"
  tables: ["ipgeolocation_ip_geolocation"]
  destinations: ["sqlite"]
  spec:
    api_key: "${IPGEOLOCATION_API_KEY}"
    ips: ["8.8.8.8"]
---
kind: destination
spec:
  name: sqlite
  path: cloudquery/sqlite
  version: "v2.9.3"
  spec:
    connection_string: "./check.db"
```

```bash
cloudquery sync check.yml
sqlite3 check.db "SELECT ip, country_name, city, company_name FROM ipgeolocation_ip_geolocation;"
```

A row naming the United States and Google confirms the key, the plugin and the destination are all working.

## Getting Help

- Plugin bugs and feature requests: [github.com/IPGeolocation/cq-source-ipgeolocation/issues](https://github.com/IPGeolocation/cq-source-ipgeolocation/issues)
- API questions, credits and plan limits: [IPGeolocation.io support](https://ipgeolocation.io/contact.html)
- CloudQuery CLI behaviour: [CloudQuery documentation](https://docs.cloudquery.io)
