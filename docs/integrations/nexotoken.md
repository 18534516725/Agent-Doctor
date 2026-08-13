# NexoToken personal usage import

Agent Doctor can import the authenticated account's own usage records to show
actual charged amount alongside local estimates. The import is opt-in and
local: Agent Doctor never receives another user's records.

## Data boundary

The version 1 response contains only:

- public model display name;
- input, output and cache Token counts;
- the authenticated user's charged amount in integer micro-units and currency;
- request timestamp; and
- an optional task correlation ID chosen by the user.

Unknown fields are discarded by Agent Doctor. Invalid version, amount,
currency, negative token counts, or out-of-range timestamps reject the whole
response. The access token is never written to diagnostic storage or logs.

## Contract

The schema is [`nexotoken-usage-v1.schema.json`](../../schemas/nexotoken-usage-v1.schema.json).
Time filters use a start-inclusive, end-exclusive interval. Exact imported
amounts are labeled **exact**; local catalog calculations remain **estimated**
and are never merged into a misleading single number.

## Privacy

Import requests are made only after an explicit user action. You can remove
imported records from Agent Doctor's local database at any time. The platform
endpoint must use the currently authenticated account only, paginate results,
and build the public DTO from an allowlist.
