# Configuration

Everything is set in the `spec` block of a `source` entry in your CloudQuery config file.

## Spec Fields

| Field | Type | Default | Description |
|---|---|---|---|
| `api_key` | string | required | Your IPGeolocation.io API key |
| `ips` | list of strings | `[]` | IP addresses to look up. An empty list means the caller's own public IP is used |
| `asns` | list of strings | `[]` | AS numbers for the ASN table, for example `["15169", "13335"]` |
| `user_agents` | list of strings | `[]` | User agent strings for the user agent table |
| `include_security` | bool | `false` | Adds the full security object to geolocation rows. Paid plans |
| `include_abuse` | bool | `false` | Adds the full abuse contact object to geolocation rows. Paid plans |
| `include_geo_accuracy` | bool | `false` | Adds `locality`, `accuracy_radius` and `confidence`. Paid plans |
| `include_dma_code` | bool | `false` | Adds `dma_code`. Filled for US locations only. Paid plans |
| `include_hostname` | bool | `false` | Adds the reverse DNS `hostname`. Paid plans |
| `hostname_lookup` | string | `fallback_live` | Hostname resolution mode: `database`, `live` or `fallback_live` |
| `asn_include` | list of strings | `[]` | Optional ASN modules: `peers`, `upstreams`, `downstreams`, `routes`, `whois_response`, or `"*"` |
| `endpoint` | string | `https://api.ipgeolocation.io` | API base URL. Change it only for a custom or self hosted endpoint |
| `timeout` | string | `30s` | HTTP request timeout, written as a Go duration such as `10s` or `1m` |
| `retry_attempts` | int | `3` | Retries for transient failures. Must be 0 or higher |
| `concurrency` | int | `10` | Tables synced in parallel. Must be 1 or higher |
| `rate_limit` | int | `15` | Maximum API requests per second. Must be 1 or higher |
| `user_agent` | string | `cq-source-ipgeolocation/1.0` | Value of the User-Agent header the plugin sends |

Invalid values are rejected before the first API call, so a typo costs you nothing in credits. A bad duration string, a negative retry count, a concurrency below 1, an unknown `asn_include` entry or an unknown `hostname_lookup` value all fail the sync immediately with a message naming the field.

## Which Inputs Feed Which Tables

| Table | Reads from |
|---|---|
| `ipgeolocation_ip_geolocation` | `ips`, or the caller IP when `ips` is empty |
| `ipgeolocation_ip_security` | `ips`, or the caller IP when `ips` is empty |
| `ipgeolocation_abuse_contact` | `ips`, or the caller IP when `ips` is empty |
| `ipgeolocation_asn` | `ips` and `asns`. Falls back to the caller IP only when both are empty |
| `ipgeolocation_user_agent` | `user_agents`. Skipped entirely when the list is empty |

The ASN table emits one row per entry in `ips` and one row per entry in `asns`. Rows from an IP lookup carry `query_ip` and leave `query_asn` empty; rows from a direct AS number lookup do the reverse.

## Free Plan Example

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
    ips: ["8.8.8.8", "1.1.1.1"]
```

## Paid Plan Example

```yaml
kind: source
spec:
  name: ipgeolocation
  registry: grpc
  path: "localhost:7777"
  tables: ["*"]
  destinations: ["postgresql"]
  spec:
    api_key: "${IPGEOLOCATION_API_KEY}"
    ips: ["8.8.8.8", "2.56.188.34"]
    asns: ["15169"]
    user_agents: ["Mozilla/5.0 (Windows NT 10.0; Win64; x64)"]
    include_security: true
    include_abuse: true
```

## Security and Abuse in the Geolocation Table

`include_security` and `include_abuse` pull the complete security and abuse objects into `ipgeolocation_ip_geolocation`, which saves you a separate sync of `ipgeolocation_ip_security` and `ipgeolocation_abuse_contact` when one wide table is easier to query.

Security columns keep their plain names (`threat_score`, `is_vpn`, `bot_type`). Abuse columns are prefixed with `abuse_` (`abuse_route`, `abuse_emails`) so they cannot collide with the location and network columns of the same name.

When a module is off, its columns are NULL rather than `false` or `0`. That distinction matters: `is_vpn = false` means the API checked and found nothing, while `is_vpn IS NULL` means nobody asked. Write `WHERE is_vpn = true` rather than `WHERE is_vpn != false` and you will not be caught out.

Each module adds to the credit cost of the request. A plain geolocation lookup is 1 credit and reaches 4 with both security and abuse included.

## Geo Accuracy, DMA Code and Hostname

Three more optional modules extend `ipgeolocation_ip_geolocation`. All three need a paid plan, and none of them adds credits beyond the base lookup.

```yaml
spec:
  api_key: "${IPGEOLOCATION_API_KEY}"
  ips: ["91.128.103.196"]
  include_geo_accuracy: true
  include_dma_code: true
  include_hostname: true
  hostname_lookup: "live"
```

| Option | Columns added | Notes |
|---|---|---|
| `include_geo_accuracy` | `locality`, `accuracy_radius`, `confidence` | `accuracy_radius` is a radius in kilometres stored as text. `confidence` is `low`, `medium` or `high`. `locality` is empty for some IPs |
| `include_dma_code` | `dma_code` | Designated Market Area code. The API fills it for US locations and returns an empty string elsewhere |
| `include_hostname` | `hostname` | When the hostname cannot be resolved, the API returns the queried IP itself |

`hostname_lookup` picks the resolution strategy:

| Value | Behaviour |
|---|---|
| `database` | Looks the name up in the IPGeolocation.io hostname database. Fast, but the API documents it as experimental and it can return unexpected values |
| `live` | Performs a live reverse DNS lookup. More accurate, adds latency to every request |
| `fallback_live` | Tries the database first and falls back to a live lookup when there is no match. This is the default |

The value is validated even when `include_hostname` is false, so a typo is caught on the first sync rather than months later when you turn hostnames on.

## Controlling the Size of the ASN Table

The `/v3/asn` endpoint can return enormous payloads for well connected networks: tens of thousands of announced prefixes, thousands of peers, and a full raw WHOIS record. None of that is requested unless you name it in `asn_include`.

```yaml
spec:
  api_key: "${IPGEOLOCATION_API_KEY}"
  asns: ["15169", "24940"]

  # Omit asn_include for compact rows with base fields only.
  # Just the transit providers and announced prefixes:
  asn_include: ["upstreams", "routes"]

  # Everything: peers, upstreams, downstreams, routes, whois_response
  # asn_include: ["*"]
```

| Value | Column | Content |
|---|---|---|
| `peers` | `peers` | Peer AS numbers, comma separated |
| `upstreams` | `upstreams` | Upstream or transit AS numbers, comma separated |
| `downstreams` | `downstreams` | Downstream customer AS numbers, comma separated |
| `routes` | `routes` | Announced CIDR prefixes, comma separated |
| `whois_response` | `whois_response` | Raw WHOIS record text |

Worth knowing:

- Values are case insensitive and duplicates are collapsed. An unrecognised value fails validation before any request goes out.
- Columns you did not select are NULL. A selected column holds an empty string when the ASN genuinely has none, so "none exist" stays distinguishable from "not fetched".
- The setting applies to lookups by IP and by AS number alike, and does not change the credit cost, which stays at 1 per lookup.

Large transit networks are the reason this is opt in. AS1 and similar tier 1 networks have peer and route lists in the thousands, and pulling them for every sync makes the table slow to write and awkward to query.

## Using the Caller's Own IP

Leave `ips` out and the geolocation, security and abuse tables look up the public IP of whatever machine runs the sync. This is useful for checking what a network egresses as.

```yaml
spec:
  api_key: "${IPGEOLOCATION_API_KEY}"
```

## Rate Limiting, Retries and Timeouts

The plugin holds a token bucket limiter set to `rate_limit` requests per second with a burst of the same size. Keep it under the ceiling for your plan and you will not see 429s during a large sync.

Retries cover 429, 500, 502, 503 and 504 responses along with network level failures. Backoff doubles each time, so the waits are 1 second, then 2, then 4. Other 4xx responses are returned immediately, because retrying a 400 or a 401 will never succeed.

`timeout` applies per HTTP request, not to the sync as a whole. Raise it if you enable `hostname_lookup: "live"`, since live reverse DNS is the slowest thing the API does.

`concurrency` controls how many tables sync at once. It does not multiply your request rate, which the limiter caps regardless.

## Selecting Tables

Use the `tables` field on the source spec to pick what syncs:

```yaml
tables:
  - "ipgeolocation_ip_geolocation"
  - "ipgeolocation_ip_security"
```

`tables: ["*"]` syncs everything. On a free plan that produces authorization errors for the four paid tables, so list only the geolocation table instead.

## Environment Variables

Reference environment variables with `${VAR_NAME}` anywhere in the config, as the examples do with `${IPGEOLOCATION_API_KEY}`. Keep keys out of version control this way.

```bash
export IPGEOLOCATION_API_KEY="your_api_key_here"
cloudquery sync ipgeo.yml
```
