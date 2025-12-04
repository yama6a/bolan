## Ikano Bank

Ikano Bank uses Borgo (same mortgage provider as ICA Banken) and has a JSON API for list rates and an HTML table for average rates.

### List Rates API

**Minimal working request:**

```bash
curl -s 'https://ikanobank.se/api/interesttable/gettabledata' \
  -H 'User-Agent: Mozilla/5.0'
```

**Response format (JSON):**

```json
{
  "success": true,
  "listData": [
    {
      "rateFixationPeriod": "3 mån",
      "listPriceInterestRate": "3.4800",
      "effectiveInterestRate": "3.5400"
    },
    {
      "rateFixationPeriod": "1 år",
      "listPriceInterestRate": "3.0800",
      "effectiveInterestRate": "3.1200"
    }
  ]
}
```

**Fields:**

- `success`: Boolean indicating if API call succeeded
- `listData`: Array of rate objects
- `rateFixationPeriod`: Term in Swedish format (3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år, 10 år)
- `listPriceInterestRate`: Nominal rate as string (decimal with dot, e.g., "3.4800")
- `effectiveInterestRate`: Effective rate as string

**Terms Available:** 3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år, 10 år

### Average Rates

**Minimal working request:**

```bash
curl -s 'https://ikanobank.se/bolan/bolanerantor' \
  -H 'User-Agent: Mozilla/5.0'
```

**Table identifier:** Search for text "Snitträntor för bolån" before the table

**Table structure:**
| Månad | 3 mån | 1 år | 2 år | 3 år | 4 år | 5 år | 7 år | 10 år |
|-------|-------|------|------|------|------|------|------|-------|
| 2025 01 | 3,61 % | 3,04 % | 2,97 % | 2,98 % | 3,04 % | 3,05 % | 3,27 % | 3,48 % |
| 2024 12 | 3,71 % | 3,04 % | 2,90 % | 2,95 % | 2,97 % | 2,99 % | 3,22 % | 3,44 % |

**Data formats:**

- Month: "YYYY MM" format with space (e.g., "2025 01" = January 2025)
- Rate: Swedish decimal format with comma and percent sign (3,61 %)
- Missing data indicated by "-"

**Note:** Ikano Bank uses Borgo as the credit provider (same as ICA Banken), but has a different API endpoint structure.

---

