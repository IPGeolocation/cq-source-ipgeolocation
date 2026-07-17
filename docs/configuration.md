# Configuration Reference

The plugin is configured through the `spec` block of a `source` entry in your CloudQuery config file.

## Fields

| Field              | Type       | Default                        | Description                                          |
| ------------------ | ---------- | ------------------------------- | ----------------------------------------------------- |
| `api_key`          | `string`   | *required*                      | Your IPGeolocation.io API key                          |
| `ips`              | `[]string` | `[]`                             | IPs to look up. Empty means the caller's own public IP is auto-detected |
| `asns`             | `[]string` | `[]`                             | AS numbers to query for the ASN table (e.g. `["15169"]`) |
| `user_agents`      | `[]string` | `[]`                             | User-agent strings to parse for the User-Agent table   |
| `include_security` | `bool`     | `false`                          | Add security data (threat score, VPN/proxy/Tor detection) to geolocation results — requires a paid plan |
| `include_abuse`    | `bool`     | `false`                          | Add abuse-contact data to geolocation results — requires a paid plan |
| `endpoint`         | `string`   | `https://api.ipgeolocation.io`   | API base URL. Override only for custom or self-hosted endpoints |
| `timeout`          | `string`   | `30s`                            | HTTP request timeout, e.g. `10s`, `1m`                  |
| `retry_attempts`   | `int`      | `3`                              | Maximum retries for transient errors                    |
| `concurrency`      | `int`      | `10`                             | Number of table syncs to run in parallel                |
| `rate_limit`       | `int`      | `15`                             | Maximum API requests per second                          |

## Example: Free Plan (Geolocation Only)

```yaml
kind: source
spec:
  name: "ipgeolocation"
  path: "ipgeolocation-io/ipgeolocation-io"
  version: "v1.0.0"
  tables: ["ipgeolocation_ip_geolocation"]
  destinations: ["sqlite"]
  spec:
    api_key: "${IPGEOLOCATION_API_KEY}"
    ips: ["8.8.8.8", "1.1.1.1"]
```

## Example: Paid Plan, All Tables With Security and Abuse Data

```yaml
kind: source
spec:
  name: "ipgeolocation"
  path: "ipgeolocation-io/ipgeolocation-io"
  version: "v1.0.0"
  tables: ["*"]
  destinations: ["sqlite"]
  spec:
    api_key: "${IPGEOLOCATION_API_KEY}"
    ips: ["8.8.8.8"]
    asns: ["15169"]
    user_agents: ["Mozilla/5.0 (Windows NT 10.0; Win64; x64)"]
    include_security: true
    include_abuse: true
```

## Auto-Detecting the Caller's IP

Leave `ips` empty to have the plugin resolve and use the machine's own public IP address:

```yaml
spec:
  api_key: "${IPGEOLOCATION_API_KEY}"
```

## Environment Variables

Reference environment variables directly in the config using `${VAR_NAME}` syntax, as shown above with `IPGEOLOCATION_API_KEY`. This avoids committing API keys to source control.
