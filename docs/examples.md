# Example Configurations

Eight ready to run configs live in the `example/` directory of the repository. Each one serves the plugin over gRPC on `localhost:7777`, so start the binary first:

```bash
go build -o cq-source-ipgeolocation
./cq-source-ipgeolocation serve
```

Then, in another terminal:

```bash
export IPGEOLOCATION_API_KEY="your_api_key_here"
cloudquery sync example/02_free_plan_geolocation_only.yml
```

| File | Scenario | Plan |
|---|---|---|
| `01_all_tables.yml` | All five tables with every optional module enabled | Paid |
| `02_free_plan_geolocation_only.yml` | Geolocation table only, three IPs, 3 credits | Free |
| `03_security_focused.yml` | Threat detection across clean, VPN, Tor and cloud IPs | Paid |
| `04_asn_and_abuse.yml` | Network topology through `asn_include`, plus abuse contacts | Paid |
| `05_user_agent_parsing.yml` | Browsers, crawlers, HTTP libraries and malformed strings | Paid |
| `06_edge_cases.yml` | IPv6, loopback, private ranges and invalid addresses | Free |
| `07_postgresql.yml` | Same shape as config 1 against a PostgreSQL destination | Paid |
| `08_auto_detect_caller_ip.yml` | No `ips` configured, so the caller's public IP is used | Free |

Each config writes to its own SQLite file, so you can run several and compare results.

## Sample Queries

### Highest risk IPs first

```sql
SELECT ip, threat_score, is_vpn, is_proxy, is_tor, vpn_provider_names
FROM ipgeolocation_ip_security
WHERE threat_score > 50
ORDER BY threat_score DESC;
```

### Location of high threat IPs

```sql
SELECT g.ip, g.country_name, g.city, g.company_name, s.threat_score, s.is_vpn
FROM ipgeolocation_ip_geolocation g
JOIN ipgeolocation_ip_security s ON g.ip = s.ip
WHERE s.threat_score > 30;
```

### Undeclared automation

Separates traffic that behaves like a bot from crawlers that identify themselves properly.

```sql
SELECT ip, bot_type, bot_operator_name, bot_confidence_score, bot_last_seen
FROM ipgeolocation_ip_security
WHERE is_bot = true AND is_known_good_bot = false;
```

### Corporate gateways you should not block

```sql
SELECT ip, corporate_gateway_provider_name, corporate_gateway_type
FROM ipgeolocation_ip_security
WHERE is_corporate_gateway = true;
```

### Largest networks in the ASN table

```sql
SELECT as_number, organization, type, country, num_of_ipv4_routes
FROM ipgeolocation_asn
ORDER BY CAST(num_of_ipv4_routes AS INTEGER) DESC;
```

### Transit providers for each ASN

Requires `asn_include: ["upstreams"]`.

```sql
SELECT as_number, organization, upstreams
FROM ipgeolocation_asn
WHERE upstreams IS NOT NULL AND upstreams != '';
```

### Abuse contacts for an incident report

```sql
SELECT ip, route, organization, emails, phone_numbers, country
FROM ipgeolocation_abuse_contact
WHERE emails != '';
```

### Mobile traffic by operating system

```sql
SELECT os_name, os_version_major, device_brand, COUNT(*) AS hits
FROM ipgeolocation_user_agent
WHERE device_type = 'Mobile'
GROUP BY os_name, os_version_major, device_brand
ORDER BY hits DESC;
```

### Country breakdown

```sql
SELECT country_name, COUNT(*) AS ips
FROM ipgeolocation_ip_geolocation
GROUP BY country_name
ORDER BY ips DESC;
```

The boolean literals above are written for PostgreSQL. SQLite stores booleans as 0 and 1, so use `is_bot = 1` there.

## Building From Source

```bash
git clone https://github.com/IPGeolocation/cq-source-ipgeolocation
cd cq-source-ipgeolocation
go build -o cq-source-ipgeolocation
```

Go 1.22.7 or later is required.

## Running Tests

```bash
go test ./... -v
```

The suite uses a stub HTTP server and needs no API key, so it runs offline and spends no credits. It covers spec validation, the API client's retry and rate limiting paths, response flattening for all five tables, and a completeness check that every API field reaches a column.
