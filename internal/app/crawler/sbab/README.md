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

