# Avanza Testdata

## Data Sources

Avanza offers mortgages through two partners via their Bolån+ product:

- **Stabelo**: `https://www.avanza.se/_api/external-mortgage-stabelo/interest-table`
- **Landshypotek (LHB)**: `https://www.avanza.se/_api/external-mortgage-lhb/interest-table`

## Data Format

Both APIs return JSON with the same structure:

```json
{
  "rows": [
    {
      "minLoanToValue": 0.0,
      "minLoanAmount": 0,
      "interestRates": [
        {
          "bindingPeriod": "THREE_MONTHS",
          "effective": 3.17,
          "nominal": 3.12
        }
      ]
    }
  ]
}
```

### Key Fields

- `minLoanToValue`: Minimum LTV percentage threshold for this tier (0=base rate)
- `minLoanAmount`: Minimum loan amount for this tier (0=base rate)
- `bindingPeriod`: Term as enum (THREE_MONTHS, ONE_YEAR, TWO_YEARS, etc.)
- `effective`: Effective annual rate (%)
- `nominal`: Nominal annual rate (%)

### Rate Tiers

The APIs return rates for multiple LTV and loan amount combinations:
- LTV tiers: 0%, 60.01%, 65.01%, 70.01%, 75.01%, 80.01%
- Loan amount tiers: 0, 500k, 600k, 700k, 800k, 900k, 1M, 1.5M, 2M, etc.

For "list rate" purposes, we use the base tier (minLoanToValue=0, minLoanAmount=0).

### Terms Available

- **Stabelo**: 3 mån, 1 år, 2 år, 3 år, 5 år, 10 år (no 4 år)
- **Landshypotek**: 3 mån, 1 år, 2 år, 3 år, 4 år, 5 år (no 10 år)

## Refresh Commands

```bash
# Download Stabelo rates
curl -s 'https://www.avanza.se/_api/external-mortgage-stabelo/interest-table' \
  -H 'User-Agent: Mozilla/5.0' > avanza_stabelo_rates.json

# Download Landshypotek rates
curl -s 'https://www.avanza.se/_api/external-mortgage-lhb/interest-table' \
  -H 'User-Agent: Mozilla/5.0' > avanza_lhb_rates.json
```

## Golden File Naming Exception

This crawler deviates from the standard `{bank}_list_rates.*` naming convention because:

- Avanza offers mortgages through two partners (Stabelo and Landshypotek)
- Each partner has a separate API endpoint
- Each partner offers different terms:
  - Stabelo: 3 mån, 1 år, 2 år, 3 år, 5 år, 10 år (no 4 år)
  - Landshypotek: 3 mån, 1 år, 2 år, 3 år, 4 år, 5 år (no 10 år)

Therefore, we maintain separate golden files per partner:
- `avanza_stabelo_rates.json` - List rates from Stabelo API
- `avanza_lhb_rates.json` - List rates from Landshypotek API

## Notes

- Avanza is an intermediary; they don't issue mortgages directly
- Only list rates available; no average rates (snitträntor) published
- Rates are negotiation-free (förhandlingsfri ränta)
- Max LTV: 85% (recently increased from lower limits)
