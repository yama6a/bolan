# Hypoteket Crawler Data

## Overview

Hypoteket is a digital-first mortgage provider ("bolånegivare") that offers non-negotiable rates ("förhandlingsfri
ränta"). They have a maximum LTV of 65%.

## Data Source

- **Rates Page**: `https://hypoteket.com/borantor`
- **Payload API**: `https://hypoteket.com/borantor/_payload.json`

## Data Format

Hypoteket uses Nuxt.js and serves rate data in a JSON payload. The payload uses a reference-based serialization format
where objects contain numeric indices that point to values in the array.

### Payload Structure

The payload is a JSON array where:

- Index 0: Metadata
- Index 1: Array descriptors
- Index 2: Data object with keys like `"interest-rates"`, `"$a58oYYQdHx"` (historical rates)
- Remaining indices: Actual values referenced by the data objects

### List Rates

Located via `data["interest-rates"]` → `["Reactive", arrayIndex]` → array of rate entry indices.

Each rate entry contains:

- `interestTerm`: Index to term name (e.g., "threeMonth", "oneYear")
- `rate`: Index to numeric rate value
- `effectiveInterestRate`: Index to effective rate
- `validFrom`: Index to ISO date string (e.g., "2025-11-10T00:00:00.000Z")

### Average Rates (Snitträntor)

Historical average rates are stored as objects with `monthPeriod` field.

Each entry contains:

- `monthPeriod`: Index to "YYYY-MM" string
- `threeMonth`, `oneYear`, `twoYear`, `threeYear`, `fiveYear`: Indices to rate values or "-" for missing

## Terms Available

Hypoteket offers 5 binding periods:

| Term       | Payload Key  |
|------------|--------------|
| Rörlig/3m  | threeMonth   |
| 1 år       | oneYear      |
| 2 år       | twoYear      |
| 3 år       | threeYear    |
| 5 år       | fiveYear     |

## Required Headers

- `User-Agent`: Standard browser user agent

No authentication required.

## Refresh Command

```bash
curl -s 'https://hypoteket.com/borantor/_payload.json' \
  -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36' \
  > internal/app/crawler/hypoteket/testdata/hypoteket_rates.json
```

## Notes

- Hypoteket is a digital-only bank with no physical branches
- They offer "förhandlingsfri ränta" (non-negotiable rates) - the listed rate is the rate you get
- Max LTV is 65% (stricter than the standard 85% bolånetak)
- Founded in 2018
- Average rates are published monthly with data going back 12 months
- Some months may have "-" for certain terms if insufficient data
