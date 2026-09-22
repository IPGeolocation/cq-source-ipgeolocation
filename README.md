# CloudQuery Source Plugin for IPGeolocation.io

A [CloudQuery](https://cloudquery.io) source plugin that pulls IP intelligence from the [IPGeolocation.io](https://ipgeolocation.io) v3 API and writes it into any CloudQuery destination: PostgreSQL, SQLite, BigQuery, Snowflake, S3, and the rest.

Point it at a list of IP addresses, AS numbers, or user agent strings. It returns geolocation, network and ASN ownership, VPN and proxy detection, threat scores, abuse contacts, and parsed browser data as ordinary database tables you can join against your own logs.

Full documentation lives in [`docs/`](docs/). Start with the [integration guide](docs/integration-guide.md) if you want the end to end walkthrough.

## Tables

| Table | Endpoint | Data | Credits | Plan |
|---|---|---|---|---|
| `ipgeolocation_ip_geolocation` | `/v3/ipgeo` | Country, city, coordinates, timezone, currency, ASN, company, network and CDN, plus optional security, abuse, accuracy, DMA code and hostname | 1 base, up to 4 with security and abuse | Free |
| `ipgeolocation_ip_security` | `/v3/security` | Threat score, VPN, proxy, residential proxy, Tor, relay, bot, spam, cloud and corporate gateway signals | 2 | Paid |
| `ipgeolocation_abuse_contact` | `/v3/abuse` | Abuse team name, emails, phone numbers and postal address for the network holding the IP | 1 | Paid |
| `ipgeolocation_asn` | `/v3/asn` | AS name, organization, RIR, allocation status, route counts, with peers, upstreams, downstreams, routes and WHOIS available on request | 1 | Paid |
| `ipgeolocation_user_agent` | `/v3/user-agent` | Browser, device, rendering engine and operating system parsed from a user agent string | 1 | Paid |

Column by column reference: [`docs/tables.md`](docs/tables.md).

## Requirements

- [CloudQuery CLI](https://docs.cloudquery.io/docs/quickstart) v5 or later
- Go 1.22.7 or later, to build the plugin binary
- An IPGeolocation.io API key. The free Developer plan gives you 1,000 requests a day and covers the geolocation table. Sign up at [app.ipgeolocation.io/signup](https://app.ipgeolocation.io/signup).

## Quick Start

```bash
# 1. Build the plugin
git clone https://github.com/IPGeolocation/cq-source-ipgeolocation
cd cq-source-ipgeolocation
go build -o cq-source-ipgeolocation

# 2. Serve it on localhost:7777 (leave this running)
./cq-source-ipgeolocation serve
```

In a second terminal:

```bash
export IPGEOLOCATION_API_KEY="your_api_key_here"
cloudquery sync example/02_free_plan_geolocation_only.yml
sqlite3 ipgeo_free_plan.db "SELECT ip, country_name, city FROM ipgeolocation_ip_geolocation;"
```

A minimal config looks like this:

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
---
kind: destination
spec:
  name: sqlite
  path: cloudquery/sqlite
  version: "v2.9.3"
  spec:
    connection_string: "./ipgeo.db"
```

## Configuration

| Field | Type | Default | Description |
|---|---|---|---|
| `api_key` | string | required | Your IPGeolocation.io API key |
| `ips` | list of strings | `[]` | IP addresses to look up. Leave it out to use the caller's own public IP |
| `asns` | list of strings | `[]` | AS numbers for the ASN table, for example `["15169"]` |
| `user_agents` | list of strings | `[]` | User agent strings for the user agent table |
| `include_security` | bool | `false` | Adds the security object to geolocation rows. Paid plans |
| `include_abuse` | bool | `false` | Adds the abuse contact object to geolocation rows. Paid plans |
| `include_geo_accuracy` | bool | `false` | Adds `locality`, `accuracy_radius` and `confidence`. Paid plans |
| `include_dma_code` | bool | `false` | Adds `dma_code`, filled for US locations only. Paid plans |
| `include_hostname` | bool | `false` | Adds the reverse DNS `hostname`. Paid plans |
| `hostname_lookup` | string | `fallback_live` | How the hostname is resolved: `database`, `live` or `fallback_live` |
| `asn_include` | list of strings | `[]` | Optional ASN modules: `peers`, `upstreams`, `downstreams`, `routes`, `whois_response`, or `"*"` for all |
| `endpoint` | string | `https://api.ipgeolocation.io` | API base URL |
| `timeout` | string | `30s` | HTTP request timeout |
| `retry_attempts` | int | `3` | Retries for transient errors such as 429 and 5xx |
| `concurrency` | int | `10` | Tables synced in parallel |
| `rate_limit` | int | `15` | Maximum API requests per second |
| `user_agent` | string | `cq-source-ipgeolocation/1.0` | User-Agent header sent with every request |

Details and worked examples: [`docs/configuration.md`](docs/configuration.md).

## Example Configs

The `example/` directory has eight configs you can run as is.

| File | What it covers |
|---|---|
| `01_all_tables.yml` | All five tables with every optional module turned on |
| `02_free_plan_geolocation_only.yml` | Geolocation table on the free plan |
| `03_security_focused.yml` | Threat detection across clean, VPN, Tor and cloud IPs |
| `04_asn_and_abuse.yml` | Network topology through `asn_include`, plus abuse contacts |
| `05_user_agent_parsing.yml` | Browser, crawler and HTTP library detection |
| `06_edge_cases.yml` | IPv6, private ranges, bogons and malformed input |
| `07_postgresql.yml` | PostgreSQL destination |
| `08_auto_detect_caller_ip.yml` | Looking up the machine's own public IP |

## Documentation

- [Integration guide](docs/integration-guide.md): install, configure, sync, query
- [Overview](docs/overview.md): what the plugin does and when to use it
- [Configuration](docs/configuration.md): every spec field, with examples
- [Tables](docs/tables.md): every column and type
- [Examples](docs/examples.md): the bundled configs and sample SQL
- [Troubleshooting](docs/troubleshooting.md): error codes and empty tables
- [FAQ](docs/faq.md)

## Development

```bash
go build ./...     # compile
go test ./... -v   # unit tests, no API key needed
go vet ./...
```

The tests use a stub HTTP server, so they run offline and make no API calls.

## License

MIT. See [LICENSE](LICENSE).
