# Example Configurations

The plugin repository includes eight ready-to-use example configs covering common scenarios. Clone the repository and find them under `example/`.

| File                                 | Scenario                                     |
| ------------------------------------- | --------------------------------------------- |
| `01_all_tables.yml`                   | All five tables, full feature set (paid plan)  |
| `02_free_plan_geolocation_only.yml`   | Free plan, geolocation table only              |
| `03_security_focused.yml`             | Threat detection across a set of IP addresses  |
| `04_asn_and_abuse.yml`                | Network topology plus abuse contact lookups    |
| `05_user_agent_parsing.yml`           | Browser and bot detection from user-agent strings |
| `06_edge_cases.yml`                   | IPv6 addresses, bogon IPs, and invalid IP handling |
| `07_postgresql.yml`                   | Syncing to a PostgreSQL destination instead of SQLite |
| `08_auto_detect_caller_ip.yml`        | Auto-detecting and looking up the machine's own public IP |

## Running an Example

```bash
export IPGEOLOCATION_API_KEY="your-key"
cloudquery sync example/02_free_plan_geolocation_only.yml
```

## Building From Source

```bash
git clone https://github.com/IPGeolocation/cq-source-ipgeolocation
cd cq-source-ipgeolocation
go build -o cq-source-ipgeolocation
```

## Running Tests

The plugin ships with 34 tests that run without requiring a live API key:

```bash
go test ./... -v
```
