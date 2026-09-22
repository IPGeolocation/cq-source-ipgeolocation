# Overview

The IPGeolocation.io source plugin connects the [IPGeolocation.io v3 API](https://ipgeolocation.io/documentation.html) to [CloudQuery](https://cloudquery.io), so IP intelligence lands in your warehouse as regular tables instead of as JSON you have to parse yourself.

You give it a list of IP addresses, AS numbers or user agent strings. On each sync it calls the API, flattens the nested responses into columns, and writes rows to whichever destination you configured: PostgreSQL, SQLite, BigQuery, Snowflake, S3, ClickHouse, and the others CloudQuery supports.

## What You Get

- Location data: country, state, city, district, postal code, coordinates, timezone and local currency
- Network and ASN data: AS number, organization, RIR, allocation status and route counts, with peers, upstreams, downstreams, announced prefixes and raw WHOIS available when you ask for them
- Security signals: threat score, VPN, proxy, residential proxy, Tor, relay, bot, spam, known attacker, cloud provider and corporate gateway detection
- Abuse contacts: the team name, email addresses, phone numbers and postal address registered for the network block
- User agent parsing: browser, version, device, brand, rendering engine and operating system

## When This Plugin Is a Good Fit

The plugin is built for batch enrichment on a schedule rather than for per request lookups in an application. Typical jobs:

- Enriching a fixed set of IPs (office egress ranges, vendor endpoints, allowlisted partners) so the data sits next to your infrastructure tables
- Building a security reference table that SIEM queries can join against, without the SIEM itself making API calls
- Auditing which countries and networks your known IPs resolve to, for compliance reporting
- Mapping ASN ownership and transit relationships for a group of prefixes you care about

If you need a lookup inside a request handler, call the [IP Geolocation API](https://ipgeolocation.io/documentation/ip-geolocation-api.html) directly or use one of the [official SDKs](https://ipgeolocation.io/documentation.html#api-sdks) instead. CloudQuery is a batch tool and the sync runs on your schedule, not on your traffic.

## Tables

| Table | Endpoint | Primary key | Plan |
|---|---|---|---|
| `ipgeolocation_ip_geolocation` | `/v3/ipgeo` | `ip` | Free |
| `ipgeolocation_ip_security` | `/v3/security` | `ip` | Paid |
| `ipgeolocation_abuse_contact` | `/v3/abuse` | `ip` | Paid |
| `ipgeolocation_asn` | `/v3/asn` | `query_ip`, `query_asn` | Paid |
| `ipgeolocation_user_agent` | `/v3/user-agent` | `user_agent_string` | Paid |

See [Tables](tables.md) for every column and its type.

## Getting an API Key

Create an account at [app.ipgeolocation.io/signup](https://app.ipgeolocation.io/signup) and copy the key from your dashboard. No card is needed for the free Developer plan, which allows 1,000 requests per day and covers `ipgeolocation_ip_geolocation`. The security, abuse, ASN and user agent endpoints need a paid plan. Current limits are on the [pricing page](https://ipgeolocation.io/pricing.html).

## Quick Start

```bash
git clone https://github.com/IPGeolocation/cq-source-ipgeolocation
cd cq-source-ipgeolocation
go build -o cq-source-ipgeolocation
./cq-source-ipgeolocation serve
```

With the plugin serving on `localhost:7777`, write a config:

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
    connection_string: "./ipgeo.db"
```

Then sync and query:

```bash
export IPGEOLOCATION_API_KEY="your_api_key_here"
cloudquery sync ipgeo.yml
sqlite3 ipgeo.db "SELECT ip, country_name, city FROM ipgeolocation_ip_geolocation;"
```

[Configuration](configuration.md) covers the rest of the spec fields.

## Source Code and License

The plugin is open source under the MIT license. Code, issues and example configs are at [github.com/IPGeolocation/cq-source-ipgeolocation](https://github.com/IPGeolocation/cq-source-ipgeolocation).
