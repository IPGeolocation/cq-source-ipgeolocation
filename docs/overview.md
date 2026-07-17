# IPGeolocation.io Source Plugin

The IPGeolocation.io source plugin fetches IP intelligence data from the [IPGeolocation.io](https://ipgeolocation.io) API v3 and syncs it into any CloudQuery-supported destination, including PostgreSQL, BigQuery, S3, Snowflake, and SQLite.

Use this plugin to enrich IP addresses, autonomous system numbers (ASNs), and user-agent strings with location, network, security, and abuse-contact data, then query the results alongside the rest of your infrastructure and security data in your data warehouse.

## What This Plugin Provides

- **IP geolocation**: country, region, city, coordinates, timezone, and currency for any IP address
- **Network and ASN data**: organization, routes, peers, upstreams, downstreams, and allocation status
- **Security intelligence**: threat scores, VPN/proxy/Tor/relay/bot/spam detection
- **Abuse contacts**: abuse team names, emails, phone numbers, and postal addresses per network block
- **User-agent parsing**: browser, device, engine, and operating system detection

## Getting an API Key

Sign up for a free API key at [app.ipgeolocation.io/signup](https://app.ipgeolocation.io/signup). The free plan covers the `ipgeolocation_ip_geolocation` table; the remaining tables require a paid plan.

## Quick Start

```bash
# 1. Set your API key
export IPGEOLOCATION_API_KEY="your-key"

# 2. Configure a source spec (see Configuration doc for all fields)
cat <<'EOF' > ipgeo.yml
kind: source
spec:
  name: "ipgeolocation"
  path: "ipgeolocation-io/ipgeolocation"
  version: "v1.0.0"
  tables: ["ipgeolocation_ip_geolocation"]
  destinations: ["sqlite"]
  spec:
    api_key: "${IPGEOLOCATION_API_KEY}"
    ips: ["8.8.8.8"]
---
kind: destination
spec:
  name: "sqlite"
  path: "cloudquery/sqlite"
  version: "v2.10.12"
  spec:
    connection_string: "./ipgeo.db"
EOF

# 3. Sync
cloudquery sync ipgeo.yml

# 4. Query
sqlite3 ipgeo.db "SELECT ip, country_name, city FROM ipgeolocation_ip_geolocation;"
```

See the Configuration and Tables docs for the full set of options and available data.

## Source Code

The plugin is open source under the Apache License 2.0. Find the code, issues, and example configs at [github.com/IPGeolocation/cq-source-ipgeolocation](https://github.com/IPGeolocation/cq-source-ipgeolocation).
