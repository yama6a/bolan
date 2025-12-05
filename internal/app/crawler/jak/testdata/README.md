# JAK Medlemsbank Crawler Data

## Overview

JAK Medlemsbank is an ethical/cooperative bank with a unique "sparlånesystem" (savings-loan system).
They only offer **2 binding periods**: 3 månader and 12 månader.

JAK is different from traditional banks:
- Member-owned cooperative
- Customers must save alongside their loan
- "Interest-free" model where interest is offset by savings requirements
- Requires membership to get a mortgage

## Data Source

- **Rates Page**: `https://www.jak.se/snittranta/`

## Data Format

JAK uses static HTML tables. Both list rates and average rates are on the same page.

### Table Structure

Two tables exist on the page:

1. **"List- och snittränta 3 månader"** - 3 month rates
2. **"List- och snittränta 12 månader"** - 12 month (1 year) rates

Each table has the same structure:

| Column | Description |
|--------|-------------|
| Tidsperiod | Month in "YYYY MM" format (e.g., "2025 11") |
| Listränta | List rate in Swedish format (e.g., "3,58 %") |
| Snittränta | Average rate in Swedish format (e.g., "3,24 %") |
| Sparkrav | Savings requirement (always "Enligt sparlånesystem") |

### List Rate Logic

The **first row** of each table contains the current list rate.
JAK publishes the same rate for all customers - there's no negotiation.

### Average Rate Logic

All rows contain historical average rates (snittränta).
Missing values are shown as "-".

## Terms Available

JAK offers only 2 binding periods:

| Term | Payload Key | Notes |
|------|-------------|-------|
| 3 månader | 3 månader | Variable rate, changes quarterly (Jan 2, Apr 2, Jul 2, Oct 2) |
| 12 månader | 12 månader | 1 year fixed rate |

## Required Headers

- `User-Agent`: Standard browser user agent

No authentication required.

## Refresh Command

```bash
curl -s 'https://www.jak.se/snittranta/' \
  -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36' \
  > internal/app/crawler/jak/testdata/jak_rates.html
```

## Notes

- JAK is an ethical bank founded in 1965
- Members must save alongside their loan (sparlånesystem)
- The "list rate" and "average rate" are typically very close because there's no negotiation
- Rate changes for 3-month term occur on fixed quarterly dates
- Only 2 terms available (compared to 8-10 terms at most banks)
- Max LTV is 85% (standard bolånetak)
- Data quality note: Some rows have malformed data like "3,58 % %" (double percent sign)
