# Nordnet Crawler

## Overview

Nordnet is a Nordic investment platform that offers mortgages via Stabelo. Their mortgage rates are displayed on their
website using data from a Contentful CMS API.

## Data Source

- **API URL**:
  `https://api.prod.nntech.io/cms/v1/contentful-cache/spaces/main_se/environments/master/entries?include=5&sys.id=36p8FGv6CCUfUIiXPjPBJy`
- **Data Format**: JSON (Contentful CMS response)
- **Authentication**: None required
- **User-Agent**: Standard browser User-Agent recommended

## Rate Structure

Nordnet offers mortgages with rates based on LTV (Loan-to-Value) ratio:

| LTV Tier | Description                       |
|----------|-----------------------------------|
| <75%     | Best rates                        |
| 75-80%   | Medium rates                      |
| 80-85%   | Highest rates (used as list rate) |

## Terms Available

- 3 mån (3 months variable)
- 1 år (1 year)
- 2 år (2 years)
- 3 år (3 years)
- 5 år (5 years)
- 10 år (10 years)

## Rate Format

Rates are displayed in Swedish decimal format with effective rate in parentheses:

- Example: `2,54 (2,57)` where 2.54% is the nominal rate and 2.57% is the effective rate

## Implementation Notes

1. **List Rates Only**: Nordnet does not publish average rates (snitträntor) since they use Stabelo as their mortgage
   provider.

2. **LTV-Based Rates**: For list rates, we use the highest LTV tier (80-85%) which represents the standard/worst-case
   rate that most customers would be quoted.

3. **Contentful CMS**: The rate data is served via Contentful CMS API. The table data is found in an entry with
   `contentType.sys.id = "componentTable"` and `internalName` containing "Räntor" and "Tabell".

## Refresh Golden File

```bash
curl -sL 'https://api.prod.nntech.io/cms/v1/contentful-cache/spaces/main_se/environments/master/entries?include=5&sys.id=36p8FGv6CCUfUIiXPjPBJy' \
  -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36' \
  > testdata/nordnet_rates.json
```

## Last Updated

Golden file last updated: 2025-12-04
