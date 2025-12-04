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

