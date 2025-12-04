# Crawler Data Sources

This document describes the data sources and HTTP requests for each bank crawler.

> **Last validated:** 2025-12-03

## Overview

| Bank          | Data Sources | Rate Types     | Auth Required           | Min Headers            |
|---------------|--------------|----------------|-------------------------|------------------------|
| SEB           | 2 JSON APIs  | List + Average | Yes (API key + Referer) | X-API-Key, Referer     |
| Nordea        | 2 HTML pages | List + Average | No                      | User-Agent             |
| ICA Banken    | 1 HTML page  | List + Average | No                      | User-Agent, Sec-Ch-Ua* |
| Danske Bank   | 1 HTML page  | List + Average | No                      | User-Agent             |
| Handelsbanken | 2 JSON APIs  | List + Average | No                      | User-Agent             |
| SBAB          | 2 JSON APIs  | List + Average | No                      | User-Agent             |
| Swedbank      | 2 HTML pages | List + Average | No                      | User-Agent             |
| Stabelo       | 1 HTML page (Remix JSON) + 1 PDF | List + Average | No               | User-Agent             |

\* ICA Banken requires matching `User-Agent` and `Sec-Ch-Ua` headers (Chrome version must match in both)

---

## SEB

SEB uses a JSON API that requires an API key extracted from their JavaScript bundle.

### Step 1: Fetch the HTML page to find JS bundle filename

```bash
curl -s 'https://pricing-portal-web-public.clouda.sebgroup.com/mortgage/averageratehistoric' \
  -H 'User-Agent: Mozilla/5.0'
```

Extract the JS filename using regex: `main\.[a-zA-Z0-9]+\.js`

Example match: `main.0022927c80a4eb07.js`

### Step 2: Fetch the JS bundle to extract API key

```bash
curl -s 'https://pricing-portal-web-public.clouda.sebgroup.com/main.0022927c80a4eb07.js' \
  -H 'User-Agent: Mozilla/5.0'
```

Extract the API key using regex: `x-api-key":"(.*?)"`

Example match: `AIzaSyACwKNIkAVff9Eh_lfX8yhAPiBRiawuYbU`

### Step 3: Fetch List Rates API

**Required headers:**

- `X-API-Key` - extracted from JS bundle (required, 401 without)
- `Referer` - must be `https://pricing-portal-web-public.clouda.sebgroup.com/` (required, 403 without)
- `Origin` - optional (works without)

```bash
curl -s 'https://pricing-portal-api-public.clouda.sebgroup.com/public/mortgage/listrate/current' \
  -H 'User-Agent: Mozilla/5.0' \
  -H 'X-API-Key: AIzaSyACwKNIkAVff9Eh_lfX8yhAPiBRiawuYbU' \
  -H 'Referer: https://pricing-portal-web-public.clouda.sebgroup.com/'
```

**Error responses:**

- Missing X-API-Key: `{"code":401,"message":"UNAUTHENTICATED: Method doesn't allow unregistered callers..."}`
- Missing Referer: `{"message":"PERMISSION_DENIED: Referer blocked.","code":403}`

**Response format (JSON array):**

```json
[
  {
    "adjustmentTerm": "3mo",
    "change": -0.20,
    "startDate": "2025-09-25T04:00:00Z",
    "value": 3.84
  },
  {
    "adjustmentTerm": "1yr",
    "change": -0.20,
    "startDate": "2025-07-10T04:00:00Z",
    "value": 3.44
  }
]
```

**Fields:**

- `adjustmentTerm`: Term identifier (3mo, 1yr, 2yr, 3yr, 5yr, 7yr, 10yr)
- `change`: Rate change from previous
- `startDate`: ISO 8601 date when rate became effective
- `value`: Current nominal rate (percentage)

### Step 4: Fetch Average Rates API

```bash
curl -s 'https://pricing-portal-api-public.clouda.sebgroup.com/public/mortgage/averagerate/historic' \
  -H 'User-Agent: Mozilla/5.0' \
  -H 'X-API-Key: AIzaSyACwKNIkAVff9Eh_lfX8yhAPiBRiawuYbU' \
  -H 'Referer: https://pricing-portal-web-public.clouda.sebgroup.com/'
```

**Response format (JSON array):**

```json
[
  {
    "period": 2510,
    "rates": {
      "1yr": 2.8376548023,
      "2yr": 2.8392814692,
      "3mo": 2.6455364444,
      "3yr": 2.8989635094,
      "5yr": 3.0733379816,
      "7yr": 3.3381629981,
      "tot": 2.6563842455,
      "10yr": 3.2971524591
    }
  }
]
```

**Fields:**

- `period`: Year and month in YYMM format (e.g., 2510 = October 2025)
- `rates`: Map of term to average rate
    - `tot` is total/aggregate and is skipped by the crawler

---

## Nordea

Nordea uses two separate HTML pages - one for list rates and one for average rates.

### List Rates

**Minimal working request:**

```bash
curl -s 'https://www.nordea.se/privat/produkter/bolan/listrantor.html' \
  -H 'User-Agent: Mozilla/5.0'
```

**Table identifier:** Search for text "Listräntor för bolån" before the table

**Table structure:**
| Bindningstid | Ränta | Ändring | Senast ändrad |
|--------------|-------|---------|---------------|
| 3 mån | 3,33 % | -0,20 | 2025-10-06 |
| 1 år | 3,44 % | -0,20 | 2025-07-10 |

**Data formats:**

- Term: Swedish format (3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 8 år)
- Rate: Swedish decimal format with comma (3,33 %)
- Date: YYYY-MM-DD

**Note:** Nordea has 7 standard terms (includes 8 år, does NOT have 7 år or 10 år). The page also shows 16 år and 18 år
which are not parsed.

### Average Rates

**Minimal working request:**

```bash
curl -s 'https://www.nordea.se/privat/produkter/bolan/snittrantor.html' \
  -H 'User-Agent: Mozilla/5.0'
```

**Table identifier:** Search for text "respektive månad" before the table

**Month format:** YYYYMM (e.g., 202511 = November 2025)

---

## ICA Banken

ICA Banken uses a single HTML page containing both list rates and average rates.

### Fetch Page

ICA Banken has bot protection that checks for matching `User-Agent` and `Sec-Ch-Ua` headers. The Chrome version in both
headers must match.

**Required headers:**

- `User-Agent` - Chrome browser user agent with version number
- `Sec-Ch-Ua` - Client hints header with **matching** Chrome version
- `Accept`, `Accept-Language`, `Connection`, `Cache-Control` - Standard browser headers

```bash
curl -s 'https://www.icabanken.se/lana/bolan/bolanerantor/' \
  -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 13_5_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.847 Safari/537.36' \
  -H 'Sec-Ch-Ua: "Chromium";v="125", "Brave";v="125", "Not_A Brand";v="99"' \
  -H 'Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9,application/json' \
  -H 'Accept-Language: en-US,en;q=0.9,de-DE;q=0.8,de;q=0.7' \
  -H 'Connection: keep-alive' \
  -H 'Cache-Control: no-cache'
```

**Key:** The `Sec-Ch-Ua` version (125) must match the Chrome version in `User-Agent` (Chrome/125.x.x.x).

### List Rates Table

**Table identifier:** Search for text "Aktuella bolåneräntor" before the table

**Table structure:**
| Bindningstid | Ränta | Senast ändrad |
|--------------|-------|---------------|
| 3 mån | 3,33 % | 2025-10-06 |
| 1 år | 3,44 % | 2025-07-10 |

**Data formats:**

- Term: Swedish format (3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år, 10 år)
- Rate: Swedish decimal format with comma (3,33 %)
- Date: YYYY-MM-DD

### Average Rates Table

**Table identifier:** Search for text "Snitträntor för bolån" before the table

**Table structure:**
| Månad | 3 mån | 1 år | 2 år | 3 år | 4 år | 5 år | 7 år | 10 år |
|-------|-------|------|------|------|------|------|------|-------|
| 2025 11 | 2,65 | 2,84 | 2,84 | 2,90 | 2,99 | 3,07 | 3,34 | 3,30 |
| 2025 10 | 2,68 | 2,92 | 2,93 | 3,00 | 3,10 | 3,19 | - | 3,57 |

**Data formats:**

- Month: "YYYY MM" format with space (e.g., "2025 11" = November 2025)
- Rate: Swedish decimal format with comma
- Missing data indicated by "-" or "-*"

---

## Danske Bank

Danske Bank uses a single HTML page containing both list rates and average rates.

### Fetch Page

**Minimal working request:**

```bash
curl -s 'https://danskebank.se/privat/produkter/bolan/relaterat/aktuella-bolanerantor' \
  -H 'User-Agent: Mozilla/5.0'
```

### List Rates Table

**Table identifier:** Search for text "Läs mer om listräntor" before the table, then skip 1 table (first is empty style
table)

**Table structure (4 columns):**
| Bindningstid | Ändrad | Förändring | Listränta |
|--------------|--------|------------|-----------|
| 3 mån | 2025-10-06 | -0,20 | 3.33% |

**Data formats:**

- Term: Swedish format
- Date: YYYY-MM-DD
- Rate: Decimal with dot (3.33%)

### Average Rates Table

**Table identifier:** Search for text "Historiska snitträntor" before the table

**Table structure:**
| Period | 3 mån | 1 år | 2 år | 3 år | ...
|--------|-------|------|------|------|
| Augusti 2021 | 1,23 | 1,44 | 1,66 | ... |

**Data formats:**

- Month: Swedish month name with year (e.g., "Augusti 2021", "Feb 1955")
- Rate: Swedish decimal format with comma

**Note:** Danske Bank has inconsistent HTML table formatting where some rows are split across multiple `<tr>` elements.
The crawler handles this by detecting rows with only a month name and merging with the following row.

---

## Handelsbanken

Handelsbanken uses clean JSON APIs for both list rates and average rates. No authentication required.

### List Rates API

**Minimal working request:**

```bash
curl -s 'https://www.handelsbanken.se/tron/slana/slan/service/mortgagerates/v1/interestrates' \
  -H 'User-Agent: Mozilla/5.0'
```

**Response format (JSON):**

```json
{
  "interestRates": [
    {
      "effectiveRateValue": {
        "value": "3,91",
        "valueRaw": 3.91
      },
      "periodBasisType": "3",
      "rateValue": {
        "value": "3,84",
        "valueRaw": 3.84
      },
      "term": "3"
    },
    {
      "effectiveRateValue": {
        "value": "3,50",
        "valueRaw": 3.50
      },
      "periodBasisType": "4",
      "rateValue": {
        "value": "3,44",
        "valueRaw": 3.44
      },
      "term": "1"
    }
  ]
}
```

**Fields:**

- `periodBasisType`: "3" = months, "4" = years
- `term`: Number of months or years (depending on periodBasisType)
- `rateValue.valueRaw`: Nominal rate (percentage)
- `effectiveRateValue.valueRaw`: Effective rate (percentage)

**Terms Available:** 3 mån, 1 år, 2 år, 3 år, 5 år, 8 år, 10 år (note: 8 år instead of 7 år)

### Average Rates API

```bash
curl -s 'https://www.handelsbanken.se/tron/slana/slan/service/mortgagerates/v1/averagerates' \
  -H 'User-Agent: Mozilla/5.0'
```

**Response format (JSON):**

```json
{
  "averageRatePeriods": [
    {
      "period": "202412",
      "rates": [
        {
          "periodBasisType": "3",
          "rateValue": {
            "value": "3,52",
            "valueRaw": 3.52
          },
          "term": "3"
        },
        {
          "periodBasisType": "4",
          "rateValue": {
            "value": "3,13",
            "valueRaw": 3.13
          },
          "term": "1"
        }
      ]
    }
  ]
}
```

**Fields:**

- `period`: Year and month in YYYYMM format (e.g., 202412 = December 2024)
- `rates`: Array of rates per term (same structure as list rates)

---

## SBAB

SBAB uses clean JSON APIs for both list rates and average rates. No authentication required.

### List Rates API

**Minimal working request:**

```bash
curl -s 'https://www.sbab.se/api/interest-mortgage-service/api/external/v1/interest' \
  -H 'User-Agent: Mozilla/5.0'
```

**Response format (JSON):**

```json
{
  "listInterests": [
    {
      "period": "P_3_MONTHS",
      "interestRate": "3.05",
      "validFrom": "2025-09-29"
    },
    {
      "period": "P_1_YEAR",
      "interestRate": "3.17",
      "validFrom": "2025-07-04"
    }
  ]
}
```

**Fields:**

- `period`: Term identifier (P_3_MONTHS, P_1_YEAR, P_2_YEARS, P_3_YEARS, P_4_YEARS, P_5_YEARS, P_7_YEARS, P_10_YEARS)
- `interestRate`: Nominal rate as string (percentage)
- `validFrom`: ISO 8601 date when rate became effective

**Terms Available:** 3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år, 10 år

### Average Rates API

```bash
curl -s 'https://www.sbab.se/api/historical-average-interest-rate-service/interest-rate/average-interest-rate-last-twelve-months-by-period' \
  -H 'User-Agent: Mozilla/5.0'
```

**Response format (JSON):**

```json
{
  "average_interest_rate_last_twelve_months": [
    {
      "period": "2025-11-30",
      "three_months": 2.69,
      "one_year": 2.85,
      "two_years": 2.93,
      "three_years": 3.02,
      "four_years": 3.05,
      "five_years": 3.19,
      "seven_years": 3.52,
      "ten_years": 3.67
    }
  ]
}
```

**Fields:**

- `period`: Date in YYYY-MM-DD format (last day of month)
- `three_months`, `one_year`, etc.: Average rate for each term (nullable - some months may have null values)

**Note:** SBAB uses snake_case for JSON field names in the average rates API.

---

## Swedbank

Swedbank uses two separate HTML pages - one for list rates and one for historic average rates.

### List Rates

**Minimal working request:**

```bash
curl -s 'https://www.swedbank.se/privat/boende-och-bolan/bolanerantor.html' \
  -H 'User-Agent: Mozilla/5.0'
```

**Table identifier:** Search for text "Aktuella bolåneräntor – listpris" before the table

**Table structure:**
| Bindningstid | Ränta |
|--------------|-------|
| 3 mån | 3,05 % |
| 1 år | 3,17 % |

**Data formats:**

- Term: Swedish format (3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år, 10 år)
- Rate: Swedish decimal format with comma (3,05 %)
- Date extracted from header: "senast ändrad 25 september 2025"

### Historic Average Rates

**Minimal working request:**

```bash
curl -s 'https://www.swedbank.se/privat/boende-och-bolan/bolanerantor/historiska-genomsnittsrantor.html' \
  -H 'User-Agent: Mozilla/5.0'
```

**Table identifier:** Find table by caption containing "Våra historiska genomsnittsräntor"

**Table structure (transposed - months as rows, terms as columns):**
| Bindningstid | 3 månader | 1 år | 2 år | 3 år | 4 år | 5 år | 7 år | 10 år | Banklån* |
|--------------|-----------|------|------|------|------|------|------|-------|----------|
| nov. 2025 | 2,65 | 2,84 | 2,84 | 2,90 | 2,99 | 3,07 | 3,34 | 3,30 | 4,88 |
| okt. 2025 | 2,68 | 2,92 | 2,93 | 3,00 | 3,10 | 3,19 | 3,50 | 3,57 | 4,98 |

**Data formats:**

- Month: Abbreviated Swedish month with period + year (e.g., "nov. 2025", "okt. 2025")
- Rate: Swedish decimal format with comma
- Missing data: "-" or empty
- Banklån* row is skipped (not a standard term)

**Month abbreviations:**

| Abbreviation | Full Name | Month |
|--------------|-----------|-------|
| jan. | januari | January |
| feb. | februari | February |
| mar. | mars | March |
| apr. | april | April |
| maj | maj | May |
| jun. | juni | June |
| jul. | juli | July |
| aug. | augusti | August |
| sep. | september | September |
| okt. | oktober | October |
| nov. | november | November |
| dec. | december | December |

---

## Stabelo

Stabelo uses a Remix.js application that embeds rate data as JSON in the HTML page.

### Rate Model

Stabelo's rates are **not simple list rates**. They vary based on three factors:
1. **Loan Amount** - Volume discounts for larger loans
2. **LTV Ratio** - Risk-based pricing by loan-to-value percentage
3. **Green Loan (Grönt bolån)** - 0.10% discount for energy-efficient properties (EPC class A/B)

For comparison purposes, we define the **"list rate"** as the **worst-case rate** (highest rate):
- Loan amount: ≤500k SEK (smallest tier = no volume discount)
- LTV: >80% (uses base tier = highest rate)
- No green loan discount

### Fetch Page

**Minimal working request:**

```bash
curl -s 'https://api.stabelo.se/rate-table/' \
  -H 'User-Agent: Mozilla/5.0'
```

The rate data is embedded in a `<script>` tag as `window.__remixContext = {...}`.

### Extract Rate Data

**Option 1: Extract from __remixContext (initial page load)**

```bash
curl -s 'https://api.stabelo.se/rate-table/' \
  -H 'User-Agent: Mozilla/5.0' \
  | grep -oP 'window.__remixContext\s*=\s*\K\{.*?\}(?=;\s*<\/script>)'
```

**Option 2: Use regex to extract the JSON array directly**

The rate data is in the path:
`loaderData["routes/_index"].rateTable.interest_rate_items`

### JSON Response Structure

The rate table contains 864 entries (all combinations of term × loan amount × LTV × green loan).

**Single entry format (list rate - worst case):**

```json
{
  "interest_rate": {
    "bps": 333,
    "display": "3,33 %"
  },
  "product_configuration": {
    "rate_fixation": "3M",
    "product_amount": {
      "value": 0
    }
  }
}
```

**Entry with LTV tier (better rate):**

```json
{
  "interest_rate": {
    "bps": 275,
    "display": "2,75 %"
  },
  "product_configuration": {
    "rate_fixation": "3M",
    "ltv": 60,
    "product_amount": {
      "value": 10000000
    }
  }
}
```

**Entry with green loan discount:**

```json
{
  "interest_rate": {
    "bps": 265,
    "display": "2,65 %"
  },
  "product_configuration": {
    "rate_fixation": "3M",
    "ltv": 60,
    "epc_classification": "B",
    "product_amount": {
      "value": 10000000
    }
  }
}
```

### Fields

- `interest_rate.bps`: Rate in basis points (333 = 3.33%)
- `interest_rate.display`: Swedish formatted display string
- `product_configuration.rate_fixation`: Term identifier (3M, 1Y, 2Y, 3Y, 5Y, 10Y)
- `product_configuration.product_amount.value`: Loan amount threshold in SEK (0 = base tier ≤500k)
- `product_configuration.ltv`: LTV percentage threshold (60, 65, 70, 75, 80) - **absent means >80%**
- `product_configuration.epc_classification`: Green loan tier ("B") - **absent means no green discount**

### List Rate Extraction Logic

Filter for entries where:
1. `product_configuration.ltv` is **absent** (not present in JSON) → worst LTV tier (>80%)
2. `product_configuration.epc_classification` is **absent** (not present in JSON) → no green discount
3. `product_configuration.product_amount.value` is `0` → smallest loan tier (≤500k)

This gives 6 entries (one per rate fixation term).

### Loan Amount Tiers (Volume Discounts)

| Threshold (SEK) | Description |
|-----------------|-------------|
| 0 | Base tier (≤500k) - **highest rate** |
| 500,000 | >500k |
| 600,000 | >600k |
| 700,000 | >700k |
| 800,000 | >800k |
| 900,000 | >900k |
| 1,000,000 | >1M |
| 1,500,000 | >1.5M |
| 2,000,000 | >2M |
| 3,500,000 | >3.5M |
| 4,500,000 | >4.5M |
| 10,000,000 | >10M - **lowest rate** |

### LTV Tiers (Risk Pricing)

| LTV Field | Meaning | Rate Impact |
|-----------|---------|-------------|
| (absent) | >80% LTV | **Highest rate** |
| 80 | 75-80% LTV | Better rate |
| 75 | 70-75% LTV | Better rate |
| 70 | 65-70% LTV | Better rate |
| 65 | 60-65% LTV | Better rate |
| 60 | ≤60% LTV | **Lowest rate** |

### Rate Fixation Terms

| Value | Model Term |
|-------|------------|
| 3M | 3 months |
| 1Y | 1 year |
| 2Y | 2 years |
| 3Y | 3 years |
| 5Y | 5 years |
| 10Y | 10 years |

**Note:** Stabelo does NOT offer 4, 7, or 8 year terms.

### Average Rates (PDF)

Stabelo publishes historical average rates in a PDF document. The PDF URL is dynamic and contains the current month in Swedish.

**Step 1: Find the PDF URL from the main page**

```bash
curl -s 'https://www.stabelo.se/bolanerantor' \
  -H 'User-Agent: Mozilla/5.0' \
  | grep -oP 'href="[^"]*[Gg]enomsnitts[^"]*\.pdf"' \
  | head -1 \
  | sed 's/href="//;s/"$//'
```

Example output: `documents/StabeloGenomsnittsräntorOktober2025.pdf`

**Step 2: Fetch the PDF**

```bash
# Extract PDF path and download
PDF_PATH=$(curl -s 'https://www.stabelo.se/bolanerantor' \
  -H 'User-Agent: Mozilla/5.0' \
  | grep -oP 'href="\K[^"]*[Gg]enomsnitts[^"]*\.pdf(?=")' \
  | head -1)

curl -sL "https://www.stabelo.se/$PDF_PATH" \
  -H 'User-Agent: Mozilla/5.0' \
  -o stabelo_avg_rates.pdf
```

**Note:** The URL contains Swedish characters (ä in "Genomsnittsräntor"). The server accepts both:
- URL-encoded: `StabeloGenomsnittsra%CC%88ntor...` (combining diaeresis)
- Direct UTF-8: `StabeloGenomsnittsräntor...`

### PDF Structure

The PDF contains a table with historical average rates from November 2017 onwards.

**Columns:**
| Column | Description |
|--------|-------------|
| Bindningstid | Month and year (e.g., "oktober 2025") |
| 3 mån | 3-month rate |
| 1 år | 1-year rate |
| 2 år | 2-year rate |
| 3 år | 3-year rate |
| 5 år | 5-year rate |
| 10 år | 10-year rate |

**Data formats:**
- Month: Swedish lowercase month name + year (e.g., "oktober 2025", "september 2025")
- Rate: Swedish decimal format with comma (e.g., "2,61%")
- Missing data: "-"

**Swedish month names:**
| Swedish | English |
|---------|---------|
| januari | January |
| februari | February |
| mars | March |
| april | April |
| maj | May |
| juni | June |
| juli | July |
| augusti | August |
| september | September |
| oktober | October |
| november | November |
| december | December |

**Note:** The PDF is updated on the 5th business day of each month. The filename changes monthly (e.g., `StabeloGenomsnittsräntorOktober2025.pdf` → `StabeloGenomsnittsräntorNovember2025.pdf`).

---

## Common Headers

The Go HTTP client uses these headers (with randomization):

```
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 13_5_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.3.847 Safari/537.36
Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9,application/json
Accept-Language: en-US,en;q=0.9,de-DE;q=0.8,de;q=0.7
Connection: keep-alive
Cache-Control: no-cache
Sec-Ch-Ua: "Chromium";v="125", "Brave";v="125", "Not_A Brand";v="99"
```

**Randomized values:**

- Chrome version: 120-144
- Mac OS version: 13.x.x - 15.x.x
- Minor/patch versions for Chrome and OS

---

## Term Mappings

All banks use Swedish terms that are parsed to internal term codes:

| Swedish      | Internal Code | Duration |
|--------------|---------------|----------|
| 3 mån / 3mo  | 3m            | 3 months |
| 1 år / 1yr   | 1y            | 1 year   |
| 2 år / 2yr   | 2y            | 2 years  |
| 3 år / 3yr   | 3y            | 3 years  |
| 4 år / 4yr   | 4y            | 4 years  |
| 5 år / 5yr   | 5y            | 5 years  |
| 6 år / 6yr   | 6y            | 6 years  |
| 7 år / 7yr   | 7y            | 7 years  |
| 8 år / 8yr   | 8y            | 8 years  |
| 9 år / 9yr   | 9y            | 9 years  |
| 10 år / 10yr | 10y           | 10 years |

---

## Testing

Each crawler has golden file tests in `internal/app/crawler/testdata/`:

| Bank          | Test Files                                                                  |
|---------------|-----------------------------------------------------------------------------|
| SEB           | `seb_page.html`, `seb_main.js`, `seb_list_rates.json`, `seb_avg_rates.json` |
| Nordea        | `nordea_list_rates.html`, `nordea_avg_rates.html`                           |
| ICA Banken    | `ica_banken.html`                                                           |
| Danske Bank   | `danske_bank.html`                                                          |
| Handelsbanken | `handelsbanken_list_rates.json`, `handelsbanken_avg_rates.json`             |
| SBAB          | `sbab_list_rates.json`, `sbab_avg_rates.json`                               |
| Swedbank      | `swedbank.html`, `swedbank_historic.html`                                   |
| Stabelo       | `stabelo_rate_table.html`, `stabelo_avg_rates.pdf`                          |

Run tests with:

```bash
go test ./internal/app/crawler/...
```

### Refreshing Golden Files

To update golden files with fresh data:

**SEB:**

```bash
# Get JS filename
JS_FILE=$(curl -s 'https://pricing-portal-web-public.clouda.sebgroup.com/mortgage/averageratehistoric' \
  -H 'User-Agent: Mozilla/5.0' | grep -oE 'main\.[a-zA-Z0-9]+\.js' | head -1)

# Get API key
API_KEY=$(curl -s "https://pricing-portal-web-public.clouda.sebgroup.com/$JS_FILE" \
  -H 'User-Agent: Mozilla/5.0' | grep -oE 'x-api-key":"[^"]+' | cut -d'"' -f3)

# Fetch data
curl -s 'https://pricing-portal-api-public.clouda.sebgroup.com/public/mortgage/listrate/current' \
  -H 'User-Agent: Mozilla/5.0' -H "X-API-Key: $API_KEY" \
  -H 'Referer: https://pricing-portal-web-public.clouda.sebgroup.com/' \
  > internal/app/crawler/testdata/seb_list_rates.json

curl -s 'https://pricing-portal-api-public.clouda.sebgroup.com/public/mortgage/averagerate/historic' \
  -H 'User-Agent: Mozilla/5.0' -H "X-API-Key: $API_KEY" \
  -H 'Referer: https://pricing-portal-web-public.clouda.sebgroup.com/' \
  > internal/app/crawler/testdata/seb_avg_rates.json
```

**Nordea:**

```bash
curl -s 'https://www.nordea.se/privat/produkter/bolan/listrantor.html' \
  -H 'User-Agent: Mozilla/5.0' > internal/app/crawler/testdata/nordea_list_rates.html

curl -s 'https://www.nordea.se/privat/produkter/bolan/snittrantor.html' \
  -H 'User-Agent: Mozilla/5.0' > internal/app/crawler/testdata/nordea_avg_rates.html
```

**Danske Bank:**

```bash
curl -s 'https://danskebank.se/privat/produkter/bolan/relaterat/aktuella-bolanerantor' \
  -H 'User-Agent: Mozilla/5.0' > internal/app/crawler/testdata/danske_bank.html
```

**ICA Banken:**

```bash
curl -s 'https://www.icabanken.se/lana/bolan/bolanerantor/' \
  -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 13_5_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.847 Safari/537.36' \
  -H 'Sec-Ch-Ua: "Chromium";v="125", "Brave";v="125", "Not_A Brand";v="99"' \
  -H 'Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9,application/json' \
  -H 'Accept-Language: en-US,en;q=0.9,de-DE;q=0.8,de;q=0.7' \
  -H 'Connection: keep-alive' \
  -H 'Cache-Control: no-cache' \
  > internal/app/crawler/testdata/ica_banken.html
```

**Handelsbanken:**

```bash
curl -s 'https://www.handelsbanken.se/tron/slana/slan/service/mortgagerates/v1/interestrates' \
  -H 'User-Agent: Mozilla/5.0' > internal/app/crawler/testdata/handelsbanken_list_rates.json

curl -s 'https://www.handelsbanken.se/tron/slana/slan/service/mortgagerates/v1/averagerates' \
  -H 'User-Agent: Mozilla/5.0' > internal/app/crawler/testdata/handelsbanken_avg_rates.json
```

**SBAB:**

```bash
curl -s 'https://www.sbab.se/api/interest-mortgage-service/api/external/v1/interest' \
  -H 'User-Agent: Mozilla/5.0' > internal/app/crawler/testdata/sbab_list_rates.json

curl -s 'https://www.sbab.se/api/historical-average-interest-rate-service/interest-rate/average-interest-rate-last-twelve-months-by-period' \
  -H 'User-Agent: Mozilla/5.0' > internal/app/crawler/testdata/sbab_avg_rates.json
```

**Swedbank:**

```bash
curl -s 'https://www.swedbank.se/privat/boende-och-bolan/bolanerantor.html' \
  -H 'User-Agent: Mozilla/5.0' > internal/app/crawler/testdata/swedbank.html

curl -s 'https://www.swedbank.se/privat/boende-och-bolan/bolanerantor/historiska-genomsnittsrantor.html' \
  -H 'User-Agent: Mozilla/5.0' > internal/app/crawler/testdata/swedbank_historic.html
```

**Stabelo:**

```bash
# List rates (Remix JSON embedded in HTML)
curl -s 'https://api.stabelo.se/rate-table/' \
  -H 'User-Agent: Mozilla/5.0' > internal/app/crawler/testdata/stabelo_rate_table.html

# Average rates (PDF - dynamic URL)
PDF_PATH=$(curl -s 'https://www.stabelo.se/bolanerantor' \
  -H 'User-Agent: Mozilla/5.0' \
  | grep -oP 'href="\K[^"]*[Gg]enomsnitts[^"]*\.pdf(?=")' \
  | head -1)

curl -sL "https://www.stabelo.se/$PDF_PATH" \
  -H 'User-Agent: Mozilla/5.0' \
  -o internal/app/crawler/testdata/stabelo_avg_rates.pdf
```
