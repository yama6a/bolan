# Svea Bank Crawler Data

## Overview

Svea Bank is a specialty/non-prime lender that publishes both **list rates (listräntor)** and **average rates (snitträntor)**.
Svea only offers **variable rate (rörlig ränta)** mortgages - no fixed-rate options.

## Data Sources

- **List Rates Page**: `https://www.svea.com/sv-se/privat/låna/bolån`
- **Average Rates Page**: `https://www.svea.com/sv-se/privat/låna/bolån/snitträntor`

## Data Format

### List Rates

The list rate is displayed in the page header as "Bolån från X,XX %".
This is the starting rate for new mortgage customers.

### Average Rates

Svea uses a simple static HTML table with 12 months of historical average rates.

#### Table Structure

| Column | Description |
|--------|-------------|
| Månad för utbetalning | Month in "MonthName YYYY" format (e.g., "November 2025") |
| Räntesats | Rate in Swedish format (e.g., "6,10 %") |

### Important Notes

1. **Variable Rate Only**: All rates are for rörlig ränta (3-month term)
2. **Non-Prime Lender**: Accepts customers with betalningsanmärkningar (payment remarks)
3. **Higher Rates**: Rates typically range 5.5% - 8.5% (higher than traditional banks)

## Terms Available

| Term | Notes |
|------|-------|
| 3 månader (rörlig) | Only variable rate mortgages offered |

## Required Headers

- `User-Agent`: Standard browser user agent

No authentication required.

## Refresh Commands

```bash
# List rates page
curl -s 'https://www.svea.com/sv-se/privat/låna/bolån' \
  -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36' \
  > internal/app/crawler/svea/testdata/svea_list_rates.html

# Average rates page
curl -s 'https://www.svea.com/sv-se/privat/låna/bolån/snitträntor' \
  -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36' \
  > internal/app/crawler/svea/testdata/svea_avg_rates.html
```

## Notes

- Svea Bank (formerly Svea Ekonomi) is a Swedish financial services company
- They specialize in lending to customers who may not qualify at traditional banks
- Max LTV is 85% (standard bolånetak)
- Month format uses full Swedish month names (e.g., "November", "Oktober")
