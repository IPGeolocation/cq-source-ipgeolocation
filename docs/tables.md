# Tables

| Table                           | Endpoint         | Description                                                                        | Credits per Request | Plan   |
| -------------------------------- | ---------------- | ------------------------------------------------------------------------------------ | -------------------- | ------ |
| `ipgeolocation_ip_geolocation`   | `/v3/ipgeo`       | Location, ASN, company, network, currency, and timezone data (plus optional security/abuse fields) | 1–4 per IP            | Free+  |
| `ipgeolocation_ip_security`      | `/v3/security`    | Threat score, and VPN, proxy, Tor, relay, bot, and spam detection with provider names | 2 per IP              | Paid   |
| `ipgeolocation_abuse_contact`    | `/v3/abuse`       | Abuse team name, emails, phone numbers, and postal address per network block         | 1 per IP              | Paid   |
| `ipgeolocation_asn`              | `/v3/asn`         | ASN name, organization, routes, peers, upstreams, downstreams, and allocation status | 1 per IP or ASN        | Paid   |
| `ipgeolocation_user_agent`       | `/v3/user-agent`  | Browser, device, engine, and operating system detection from user-agent strings      | 1 per user-agent       | Paid   |

## Notes

- The **Credits** column reflects IPGeolocation.io API usage credits consumed per request, not CloudQuery sync cost.
- Tables tagged **Paid** require an IPGeolocation.io plan above Free. Attempting to sync them without the appropriate plan will result in an API error.
- Use the `tables` field in your source spec to select exactly which tables to sync, or use `["*"]` to sync all tables your plan supports.
- Set `include_security: true` or `include_abuse: true` in the config spec to enrich rows in `ipgeolocation_ip_geolocation` with fields from the security and abuse endpoints, instead of syncing those as separate tables.
