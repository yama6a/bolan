# Crawler Data Sources

This document describes the data sources and HTTP requests for each bank crawler.

> **Last validated:** 2025-12-03

## Overview

| Bank          | Data Sources | Rate Types     | Auth Required           | Min Headers           |
|---------------|--------------|----------------|-------------------------|-----------------------|
| SEB           | 2 JSON APIs  | List + Average | Yes (API key + Referer) | X-API-Key, Referer    |
| Nordea        | 2 HTML pages | List + Average | No                      | User-Agent            |
| ICA Banken    | 1 HTML page  | List + Average | No                      | Full browser headers* |
| Danske Bank   | 1 HTML page  | List + Average | No                      | User-Agent            |
| Handelsbanken | 2 JSON APIs  | List + Average | No                      | User-Agent            |

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
      "effectiveRateValue": {"value": "3,91", "valueRaw": 3.91},
      "periodBasisType": "3",
      "rateValue": {"value": "3,84", "valueRaw": 3.84},
      "term": "3"
    },
    {
      "effectiveRateValue": {"value": "3,50", "valueRaw": 3.50},
      "periodBasisType": "4",
      "rateValue": {"value": "3,44", "valueRaw": 3.44},
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
        {"periodBasisType": "3", "rateValue": {"value": "3,52", "valueRaw": 3.52}, "term": "3"},
        {"periodBasisType": "4", "rateValue": {"value": "3,13", "valueRaw": 3.13}, "term": "1"}
      ]
    }
  ]
}
```

**Fields:**

- `period`: Year and month in YYYYMM format (e.g., 202412 = December 2024)
- `rates`: Array of rates per term (same structure as list rates)

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
