# Cost and quota methodology

Agent Doctor never treats missing billing data as zero.

| Precision | Meaning |
| --- | --- |
| **exact** | Authenticated billing source reported the charged micro-units |
| **estimated** | Captured usage × versioned model price catalog |
| **unavailable** | Usage, price, currency, or billing evidence is missing |

Estimated cost uses integer micro-units:

```text
input × input-price-per-million / 1,000,000
+ output × output-price-per-million / 1,000,000
+ cached × cached-price-per-million / 1,000,000
```

Every record retains currency, provenance, price version, exchange-rate version,
and precision. Exact and estimated values are displayed separately. Conversion
is rejected when a compatible versioned rate is absent. Subscription quota
forecasts require a complete observation window and become unavailable when a
plan changes or the reset point is unknown.
