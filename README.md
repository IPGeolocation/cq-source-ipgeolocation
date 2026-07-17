# CloudQuery IPGeolocation.io Source Plugin

A [CloudQuery](https://cloudquery.io) source plugin that fetches IP intelligence data from the [IPGeolocation.io](https://ipgeolocation.io) API v3 and loads it into any supported CloudQuery destination (PostgreSQL, BigQuery, S3, etc.).

## Tables

| Table | Endpoint | Description | Credits | Plan |
|---|---|---|---|---|
| `ipgeolocation_ip_geolocation` | `/v3/ipgeo` | Location, ASN, company, network, currency, timezone (+ optional security/abuse) | 1-4/IP | Free+ |
| `ipgeolocation_ip_security` | `/v3/security` | Threat score, VPN/proxy/Tor/relay/bot/spam detection, provider names | 2/IP | Paid |
| `ipgeolocation_abuse_contact` | `/v3/abuse` | Abuse team name, emails, phones, postal address per network block | 1/IP | Paid |
| `ipgeolocation_asn` | `/v3/asn` | ASN name, org, routes, peers, upstreams, downstreams, allocation status | 1/IP or ASN | Paid |
| `ipgeolocation_user_agent` | `/v3/user-agent` | Browser, device, engine, OS detection from user-agent strings | 1/UA | Paid |

## Quick Start

```bash
# 1. Get free API key: https://app.ipgeolocation.io/signup
export IPGEOLOCATION_API_KEY="your-key"

# 2. Build
git clone <repo> && cd cq-source-ipgeolocation
go build -o cq-source-ipgeolocation

# 3. Run plugin (Terminal 1)
./cq-source-ipgeolocation serve

# 4. Sync (Terminal 2)
cloudquery sync example/02_free_plan_geolocation_only.yml

# 5. Query
sqlite3 ipgeo_free_plan.db "SELECT ip, country_name, city FROM ipgeolocation_ip_geolocation;"
```

## Configuration Reference

| Field | Type | Default | Description |
|---|---|---|---|
| `api_key` | `string` | *required* | IPGeolocation.io API key |
| `ips` | `[]string` | `[]` | IPs to look up. Empty = auto-detect caller IP |
| `asns` | `[]string` | `[]` | AS numbers for the ASN table (e.g. `["15169"]`) |
| `user_agents` | `[]string` | `[]` | User-agent strings for the UA table |
| `include_security` | `bool` | `false` | Add security data to geolocation results (paid) |
| `include_abuse` | `bool` | `false` | Add abuse data to geolocation results (paid) |
| `endpoint` | `string` | `https://api.ipgeolocation.io` | API base URL |
| `timeout` | `string` | `30s` | HTTP request timeout |
| `retry_attempts` | `int` | `3` | Max retries for transient errors |
| `concurrency` | `int` | `10` | Parallel table syncs |
| `rate_limit` | `int` | `15` | Max API requests per second |

## Example Configs

See the `example/` directory for 8 ready-to-use configs covering every scenario:

| File | Scenario |
|---|---|
| `01_all_tables.yml` | All 5 tables, full features (paid plan) |
| `02_free_plan_geolocation_only.yml` | Free plan, geolocation only |
| `03_security_focused.yml` | Threat detection across IP types |
| `04_asn_and_abuse.yml` | Network topology + abuse contacts |
| `05_user_agent_parsing.yml` | Browser/bot detection |
| `06_edge_cases.yml` | IPv6, bogon IPs, invalid IPs |
| `07_postgresql.yml` | PostgreSQL destination |
| `08_auto_detect_caller_ip.yml` | Auto-detect own public IP |

## Development

```bash
go build ./...          # Build
go test ./... -v        # Run tests (34 tests, no API key needed)
```

## License

Apache License 2.0
