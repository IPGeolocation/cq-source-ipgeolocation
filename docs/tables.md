# Tables

The plugin exposes five tables. Column names follow the API field names. Array values (provider names, email addresses, phone numbers, AS numbers, routes) are stored as comma separated text rather than JSON, which keeps them portable across every CloudQuery destination.

Every table also carries the CloudQuery system columns `_cq_id` and `_cq_parent_id`.

| Table | Endpoint | Primary key | Credits per request | Plan |
|---|---|---|---|---|
| `ipgeolocation_ip_geolocation` | `/v3/ipgeo` | `ip` | 1, up to 4 with optional modules | Free |
| `ipgeolocation_ip_security` | `/v3/security` | `ip` | 2 | Paid |
| `ipgeolocation_abuse_contact` | `/v3/abuse` | `ip` | 1 | Paid |
| `ipgeolocation_asn` | `/v3/asn` | `query_ip`, `query_asn` | 1 | Paid |
| `ipgeolocation_user_agent` | `/v3/user-agent` | `user_agent_string` | 1 | Paid |

## ipgeolocation_ip_geolocation

One row per entry in `ips`, or a single row for the caller's public IP when `ips` is empty. Source: [IP Geolocation API](https://ipgeolocation.io/documentation/ip-geolocation-api.html).

### Identity

| Column | Type | Description |
|---|---|---|
| `ip` | text | The IP that was looked up. Primary key |
| `domain` | text | Domain associated with the IP, when the API has one |
| `hostname` | text | Reverse DNS hostname. NULL unless `include_hostname` is true |

### Location

| Column | Type | Description |
|---|---|---|
| `continent_code` | text | Two letter continent code |
| `continent_name` | text | Continent name |
| `country_code2` | text | ISO 3166-1 alpha-2 country code |
| `country_code3` | text | ISO 3166-1 alpha-3 country code |
| `country_name` | text | Common country name |
| `country_name_official` | text | Official country name |
| `country_capital` | text | Capital city |
| `state_prov` | text | State or province |
| `state_code` | text | ISO 3166-2 subdivision code |
| `district` | text | District or county |
| `city` | text | City |
| `zipcode` | text | Postal code |
| `latitude` | text | Latitude, stored as text |
| `longitude` | text | Longitude, stored as text |
| `is_eu` | boolean | True when the country is an EU member state |
| `geoname_id` | text | GeoNames identifier |
| `locality` | text | Locality or neighbourhood. NULL unless `include_geo_accuracy` is true |
| `accuracy_radius` | text | Accuracy radius in kilometres. NULL unless `include_geo_accuracy` is true |
| `confidence` | text | `low`, `medium` or `high`. NULL unless `include_geo_accuracy` is true |
| `dma_code` | text | Designated Market Area code, US only. NULL unless `include_dma_code` is true |

Latitude, longitude and accuracy radius come back from the API as strings and are stored that way. Cast them in SQL when you need arithmetic, for example `CAST(latitude AS DOUBLE PRECISION)`.

### Country metadata

| Column | Type | Description |
|---|---|---|
| `calling_code` | text | International dialling prefix |
| `tld` | text | Country code top level domain |
| `languages` | text | Official languages, comma separated |
| `currency_code` | text | ISO 4217 currency code |
| `currency_name` | text | Currency name |
| `currency_symbol` | text | Currency symbol |

### Network

| Column | Type | Description |
|---|---|---|
| `connection_type` | text | Connection type reported for the network |
| `route` | text | CIDR block the IP belongs to |
| `is_anycast` | boolean | True when the prefix is anycast |
| `is_cdn` | boolean | True when the IP belongs to a CDN |
| `cdn_provider_name` | text | CDN operator name |

### ASN and company

| Column | Type | Description |
|---|---|---|
| `as_number` | text | AS number, for example `AS15169` |
| `asn_organization` | text | Organization that holds the AS number |
| `asn_country` | text | Country of registration |
| `asn_type` | text | ASN type, such as ISP, HOSTING, EDUCATION, GOVERNMENT or BUSINESS |
| `asn_domain` | text | Primary domain of the organization |
| `asn_date_allocated` | text | Allocation date |
| `asn_rir` | text | Regional Internet Registry: ARIN, RIPE, APNIC, LACNIC or AFRINIC |
| `company_name` | text | Company operating the IP |
| `company_type` | text | Company type |
| `company_domain` | text | Company domain |

### Timezone

| Column | Type | Description |
|---|---|---|
| `timezone_name` | text | IANA timezone name, for example `America/New_York` |
| `timezone_offset` | double | UTC offset in seconds, ignoring DST |
| `timezone_offset_with_dst` | double | UTC offset in seconds, including DST |
| `timezone_current_time` | text | Local time at the moment of the lookup |
| `timezone_current_time_unix` | double | Same moment as a Unix timestamp |
| `timezone_abbreviation` | text | Timezone abbreviation, for example EST |
| `timezone_is_dst` | boolean | True when DST is in effect |

### Security

These are NULL unless `include_security` is true. They carry the same names and meanings as in `ipgeolocation_ip_security`, which is documented below.

`threat_score`, `is_tor`, `is_proxy`, `proxy_provider_names`, `proxy_confidence_score`, `proxy_last_seen`, `is_residential_proxy`, `is_vpn`, `vpn_provider_names`, `vpn_confidence_score`, `vpn_last_seen`, `is_relay`, `relay_provider_name`, `is_anonymous`, `is_known_attacker`, `is_bot`, `bot_confidence_score`, `bot_operator_name`, `bot_type`, `is_known_good_bot`, `bot_last_seen`, `is_spam`, `is_cloud_provider`, `cloud_provider_name`, `is_corporate_gateway`, `corporate_gateway_type`, `corporate_gateway_provider_name`

### Abuse contact

These are NULL unless `include_abuse` is true. They match the columns of `ipgeolocation_abuse_contact` with an `abuse_` prefix, which avoids a collision with the network `route` column and the ASN `country` column.

`abuse_route`, `abuse_country`, `abuse_name`, `abuse_organization`, `abuse_kind`, `abuse_address`, `abuse_emails`, `abuse_phone_numbers`

## ipgeolocation_ip_security

One row per entry in `ips`. Requires a paid plan and costs 2 credits per lookup. Source: [IP Security API](https://ipgeolocation.io/documentation/ip-security-api.html).

| Column | Type | Description |
|---|---|---|
| `ip` | text | The IP that was looked up. Primary key |
| `threat_score` | integer | Composite risk score from 0 to 100 |
| `is_tor` | boolean | Tor exit node |
| `is_proxy` | boolean | Known proxy |
| `proxy_provider_names` | text | Proxy operators, comma separated |
| `proxy_confidence_score` | integer | Proxy detection confidence from 0 to 100 |
| `proxy_last_seen` | text | Date the IP was last seen acting as a proxy |
| `is_residential_proxy` | boolean | Proxy running on a residential connection |
| `is_vpn` | boolean | Known VPN exit node |
| `vpn_provider_names` | text | VPN operators, comma separated |
| `vpn_confidence_score` | integer | VPN detection confidence from 0 to 100 |
| `vpn_last_seen` | text | Date the IP was last seen as a VPN exit |
| `is_relay` | boolean | Relay network such as iCloud Private Relay |
| `relay_provider_name` | text | Relay operator |
| `is_anonymous` | boolean | Any form of anonymisation detected |
| `is_known_attacker` | boolean | Flagged for attack activity |
| `is_bot` | boolean | Automated traffic detected |
| `bot_confidence_score` | integer | Bot detection confidence from 0 to 100 |
| `bot_operator_name` | text | Who runs the bot, for example Google |
| `bot_type` | text | Bot classification, such as search engine crawler or scraper |
| `is_known_good_bot` | boolean | Declared, well behaved crawler |
| `bot_last_seen` | text | Date the IP was last seen behaving as a bot |
| `is_spam` | boolean | Present on spam block lists |
| `is_cloud_provider` | boolean | Cloud or hosting infrastructure |
| `cloud_provider_name` | text | Cloud provider name |
| `is_corporate_gateway` | boolean | Shared corporate egress point |
| `corporate_gateway_type` | text | Gateway type |
| `corporate_gateway_provider_name` | text | Gateway operator |

`is_corporate_gateway` is worth a rule of its own in any blocking logic. A single corporate gateway can front thousands of employees, so a block there is far wider than it looks.

## ipgeolocation_abuse_contact

One row per entry in `ips`. Requires a paid plan and costs 1 credit per lookup. Source: [IP Abuse Contact API](https://ipgeolocation.io/documentation/ip-abuse-contact-api.html).

| Column | Type | Description |
|---|---|---|
| `ip` | text | The IP that was looked up. Primary key |
| `route` | text | CIDR block the abuse contact covers |
| `country` | text | Country of the registrant |
| `name` | text | Contact or incident response team name |
| `organization` | text | Organization responsible for the block |
| `kind` | text | Contact kind, such as group or individual |
| `address` | text | Postal address |
| `emails` | text | Abuse email addresses, comma separated |
| `phone_numbers` | text | Abuse phone numbers, comma separated |

`route` is the block the contact is registered for, not the single address. Use it when you need to report or block a range rather than one host.

## ipgeolocation_asn

One row per entry in `ips` plus one row per entry in `asns`. Requires a paid plan and costs 1 credit per lookup. Source: [ASN API](https://ipgeolocation.io/documentation/asn-api.html).

| Column | Type | Description |
|---|---|---|
| `query_ip` | text | The IP this row was looked up by. Empty for AS number lookups. Primary key |
| `query_asn` | text | The AS number this row was looked up by. Empty for IP lookups. Primary key |
| `as_number` | text | AS number returned by the API |
| `organization` | text | Organization holding the ASN |
| `country` | text | Country of registration |
| `type` | text | ASN type: ISP, HOSTING, EDUCATION, GOVERNMENT or BUSINESS |
| `domain` | text | Primary domain of the organization |
| `date_allocated` | text | Allocation date |
| `asn_name` | text | Short registered name |
| `allocation_status` | text | Registry status, for example assigned |
| `num_of_ipv4_routes` | text | Count of announced IPv4 prefixes, stored as text |
| `num_of_ipv6_routes` | text | Count of announced IPv6 prefixes, stored as text |
| `rir` | text | Regional Internet Registry |
| `peers` | text | Peer AS numbers, comma separated. NULL unless requested |
| `upstreams` | text | Upstream AS numbers, comma separated. NULL unless requested |
| `downstreams` | text | Downstream AS numbers, comma separated. NULL unless requested |
| `routes` | text | Announced CIDR prefixes, comma separated. NULL unless requested |
| `whois_response` | text | Raw WHOIS record. NULL unless requested |

The route counts arrive as strings, so sort them with a cast:

```sql
SELECT as_number, organization, num_of_ipv4_routes
FROM ipgeolocation_asn
ORDER BY CAST(num_of_ipv4_routes AS INTEGER) DESC;
```

The last five columns are controlled by [`asn_include`](configuration.md#controlling-the-size-of-the-asn-table). NULL means the module was not requested; an empty string means it was requested and the ASN has nothing to report.

## ipgeolocation_user_agent

One row per entry in `user_agents`. The table is skipped when the list is empty. Requires a paid plan and costs 1 credit per string. Source: [User Agent API](https://ipgeolocation.io/documentation/user-agent-api.html).

| Column | Type | Description |
|---|---|---|
| `user_agent_string` | text | The raw string that was parsed. Primary key |
| `name` | text | Client name, for example Chrome or Googlebot |
| `type` | text | Client type, such as browser, bot or library |
| `version` | text | Full version string |
| `version_major` | text | Major version only |
| `device_name` | text | Device model |
| `device_type` | text | Device category, such as Desktop, Mobile or Tablet |
| `device_brand` | text | Manufacturer |
| `device_cpu` | text | CPU architecture |
| `engine_name` | text | Rendering engine, for example Blink or WebKit |
| `engine_type` | text | Engine category |
| `engine_version` | text | Engine version |
| `engine_version_major` | text | Engine major version |
| `os_name` | text | Operating system |
| `os_type` | text | OS family |
| `os_version` | text | OS version |
| `os_version_major` | text | OS major version |
| `os_build` | text | Build identifier |

## Notes on NULL Handling

Optional modules use NULL to mean "not fetched" and never fall back to a zero value. A false in `is_vpn` came from the API; a NULL means `include_security` was off. The same applies to `hostname`, `locality`, `accuracy_radius`, `confidence`, `dma_code`, and the five optional ASN columns.

Filters should therefore be written positively. `WHERE is_tor = true` behaves as expected whether or not the module ran, while `WHERE is_tor != true` silently drops NULL rows in most SQL engines.

## Notes on Credits

The credit figures above are IPGeolocation.io API credits, not CloudQuery costs. A sync of five IPs against `ipgeolocation_ip_security` spends 10 credits. Turning on `include_security` and `include_abuse` raises a geolocation lookup from 1 credit to as many as 4.

Requesting the same data twice, once through `include_security` on the geolocation table and again through `ipgeolocation_ip_security`, bills for both. Pick one shape and stay with it.
