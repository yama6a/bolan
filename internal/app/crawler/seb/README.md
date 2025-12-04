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

