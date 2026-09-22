# FAQ

## Do I need a paid IPGeolocation.io plan?

Not for the geolocation table. The free Developer plan allows 1,000 requests a day and covers `ipgeolocation_ip_geolocation` with its base fields. The security, abuse, ASN and user agent endpoints need a paid plan, as do the optional modules `include_security`, `include_abuse`, `include_geo_accuracy`, `include_dma_code` and `include_hostname`.

## Which CloudQuery destinations work with this plugin?

All of them. Source plugins do not care where rows go, so PostgreSQL, SQLite, BigQuery, Snowflake, S3, ClickHouse, DuckDB and the rest all work. Two of the bundled examples cover SQLite and PostgreSQL.

## Does it support IPv6?

Yes. Put IPv6 addresses in `ips` exactly as you would IPv4. Coverage varies by prefix, and some IPv6 blocks return less location detail than the equivalent IPv4 space.

## Can I look up a domain name instead of an IP?

No. Resolve the name to an address first and pass that. The plugin sends whatever you put in `ips` straight to the API.

## What happens if one IP in the list fails?

It is logged as a warning and skipped, and the sync continues with the rest. A malformed address, a private range or an unsupported prefix will not abort the run. Check the logs at debug level to see which entries were dropped and why.

## How do I look up my own public IP?

Leave `ips` out. The geolocation, security and abuse tables then query whatever public address the machine running the sync egresses from. `example/08_auto_detect_caller_ip.yml` shows this.

## How many credits does a sync cost?

Per lookup: 1 for geolocation, 2 for security, 1 for abuse, 1 for ASN, 1 for a user agent string. Enabling `include_security` and `include_abuse` raises a geolocation lookup from 1 credit to as many as 4. The geo accuracy, DMA code and hostname modules add no credits.

Multiply by the number of entries in the relevant input list. Ten IPs against the geolocation and security tables with security enrichment turned on is 10 requests at up to 3 credits plus 10 requests at 2 credits.

## Why are my security columns NULL rather than false?

Because nobody asked the API. NULL means the module was not requested, while false means the check ran and found nothing. Keeping them distinct stops an unfetched IP from looking like a clean one. Set `include_security: true` to populate them.

## Should I use include_security or the separate security table?

Use `include_security` when you want one wide row per IP and plan to filter on location and threat data together. Use `ipgeolocation_ip_security` when you want a narrow, dedicated threat table, or when you are syncing security data on a different schedule from geolocation.

Doing both costs credits twice for the same information.

## Why is the ASN table missing peers and routes?

They are opt in. Add the modules you want to `asn_include`, or `"*"` for all of them. The default keeps rows compact, because large transit networks return route lists in the tens of thousands.

## Can I run this on a schedule?

Yes. `cloudquery sync` is a single command, so cron, systemd timers, GitHub Actions, Airflow or any other scheduler will do. Size the interval against your daily credit allowance.

## Does the plugin cache results?

No. Every sync makes fresh API calls. If you want to reduce spend, sync less often or trim `ips`. The data changes slowly for most IPs, so daily is enough for many uses.

## How do I keep my API key out of version control?

Reference an environment variable in the config with `${IPGEOLOCATION_API_KEY}` and export it before running the sync. CloudQuery expands it at load time and the literal key never appears in a file.

## Is the plugin published to the CloudQuery Hub?

Build it and serve it locally over gRPC for now, as the bundled examples do. Once it is on the Hub you can swap `registry: grpc` and `path: "localhost:7777"` for the registry path and a version, and drop the separate serve step.

## Which Go version do I need to build it?

Go 1.22.7 or later, matching the `go` directive in `go.mod`.

## Do the tests need an API key?

No. The suite runs against a stub HTTP server, so it works offline and spends no credits. Run `go test ./... -v`.
