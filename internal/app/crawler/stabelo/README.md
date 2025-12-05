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

| Threshold (SEK) | Description                          |
|-----------------|--------------------------------------|
| 0               | Base tier (≤500k) - **highest rate** |
| 500,000         | >500k                                |
| 600,000         | >600k                                |
| 700,000         | >700k                                |
| 800,000         | >800k                                |
| 900,000         | >900k                                |
| 1,000,000       | >1M                                  |
| 1,500,000       | >1.5M                                |
| 2,000,000       | >2M                                  |
| 3,500,000       | >3.5M                                |
| 4,500,000       | >4.5M                                |
| 10,000,000      | >10M - **lowest rate**               |

### LTV Tiers (Risk Pricing)

| LTV Field | Meaning    | Rate Impact      |
|-----------|------------|------------------|
| (absent)  | >80% LTV   | **Highest rate** |
| 80        | 75-80% LTV | Better rate      |
| 75        | 70-75% LTV | Better rate      |
| 70        | 65-70% LTV | Better rate      |
| 65        | 60-65% LTV | Better rate      |
| 60        | ≤60% LTV   | **Lowest rate**  |

### Rate Fixation Terms

| Value | Model Term |
|-------|------------|
| 3M    | 3 months   |
| 1Y    | 1 year     |
| 2Y    | 2 years    |
| 3Y    | 3 years    |
| 5Y    | 5 years    |
| 10Y   | 10 years   |

**Note:** Stabelo does NOT offer 4, 7, or 8 year terms.

### Average Rates (PDF)

Stabelo publishes historical average rates in a PDF document. The PDF URL is dynamic and contains the current month in
Swedish.

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

| Column       | Description                           |
|--------------|---------------------------------------|
| Bindningstid | Month and year (e.g., "oktober 2025") |
| 3 mån        | 3-month rate                          |
| 1 år         | 1-year rate                           |
| 2 år         | 2-year rate                           |
| 3 år         | 3-year rate                           |
| 5 år         | 5-year rate                           |
| 10 år        | 10-year rate                          |

**Data formats:**

- Month: Swedish lowercase month name + year (e.g., "oktober 2025", "september 2025")
- Rate: Swedish decimal format with comma (e.g., "2,61%")
- Missing data: "-"

**Swedish month names:**

| Swedish   | English   |
|-----------|-----------|
| januari   | January   |
| februari  | February  |
| mars      | March     |
| april     | April     |
| maj       | May       |
| juni      | June      |
| juli      | July      |
| augusti   | August    |
| september | September |
| oktober   | October   |
| november  | November  |
| december  | December  |

**Note:** The PDF is updated on the 5th business day of each month. The filename changes monthly (e.g.,
`StabeloGenomsnittsräntorOktober2025.pdf` → `StabeloGenomsnittsräntorNovember2025.pdf`).

---

