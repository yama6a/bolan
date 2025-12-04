# Swedish Mortgage Banks - Research

## Currently Implemented Crawlers
- [x] Danske Bank
- [x] SEB
- [x] ICA Banken
- [x] Nordea
- [x] SBAB
- [x] Handelsbanken
- [x] Swedbank
- [x] Bluestep
- [x] Skandiabanken
- [x] Stabelo
- [x] Ikano Bank
- [x] Ålandsbanken

## Complete Bank List (from Konsumenternas.se - 21 banks)

The following banks are listed on the official Konsumenternas.se comparison (updated 2025-11-28):

| # | Bank | Website | Status | Notes |
|---|------|---------|--------|-------|
| 1 | Avanza Bank | avanza.se | To Add | Superbolånet product, variable rate only (2.01-2.25%) |
| 2 | Bluestep/Enity Bank Group | bluestep.se | **Done** | Specialty lender, 3 mån/3 år/5 år terms, high rate range (4.45-9.30%) |
| 3 | Danske Bank | danskebank.se | **Done** | Full term range |
| 4 | Handelsbanken | handelsbanken.se | **Done** | Big Four, full term range |
| 5 | Hypoteket | hypoteket.com | To Add | Digital-first, data currently missing on Konsumenternas |
| 6 | ICA Banken | icabanken.se | **Done** | Full term range |
| 7 | Ikano Bank | ikanobank.se | **Done** | Uses Borgo, full term range |
| 8 | JAK Medlemsbank | jak.se | To Add | Ethical bank, only 3 mån and 1 år terms |
| 9 | Landshypotek | landshypotek.se | To Add | Also via Avanza/Bolån+, terms up to 5 år |
| 10 | Länsförsäkringar (LF) | lansforsakringar.se | To Add | Major player, full term range |
| 11 | Marginalen Bank | marginalen.se | To Add | Specialty lender, complex rate structure (4.46-10.56%) |
| 12 | Nordax Bank/NOBA Bank Group | nordax.se | To Add | Specialty lender, 3 mån/3 år/5 år terms (4.45-9.94%) |
| 13 | Nordea | nordea.se | **Done** | Big Four |
| 14 | Nordnet | nordnet.se | To Add | Variable rate only (1.80-3.31%) |
| 15 | SBAB | sbab.se | **Done** | State-owned, highest satisfaction, full term range |
| 16 | SEB | seb.se | **Done** | Big Four |
| 17 | Skandiabanken | skandiabanken.se | **Done** | Insurance bank, terms up to 5 år |
| 18 | Stabelo | stabelo.se | To Add | Also via Avanza/Bolån+ and Nordnet, data currently missing |
| 19 | Svea Bank | svea.com | To Add | Specialty lender, variable rate only (from 5.65%) |
| 20 | Swedbank | swedbank.se | **Done** | Big Four, largest market share (~25%), full term range |
| 21 | Ålandsbanken | alandsbanken.se | **Done** | Finnish bank in Sweden, full term range |

## Banks by Category

### Big Four (~71% market share)
| Bank | Market Share | Status |
|------|--------------|--------|
| Swedbank | ~25% | **Done** |
| Handelsbanken | ~24% | **Done** |
| SEB | ~15% | **Done** |
| Nordea | ~14% | **Done** |

### Major Banks & Mortgage Institutions
| Bank | Notes | Status |
|------|-------|--------|
| SBAB | ~8.5% share, state-owned, highest customer satisfaction | **Done** |
| Länsförsäkringar Bank | Insurance company's bank, high satisfaction | To Add |
| Skandiabanken | Insurance/pension company's bank | **Done** |
| Danske Bank | - | **Done** |
| Landshypotek Bank | Agricultural/rural property focus | To Add |
| Ålandsbanken | Finnish bank operating in Sweden | To Add |
| ICA Banken | Uses Borgo | **Done** |
| Ikano Bank | Uses Borgo | **Done** |

### Digital/Fintech Mortgage Providers
| Bank | Notes | Status |
|------|-------|--------|
| Hypoteket | Digital-first, negotiation-free rates | To Add |
| Stabelo | Fintech, via Avanza/Nordnet | To Add |
| Avanza Bank | Superbolånet, variable only | To Add |
| Nordnet | Variable only, uses other providers | To Add |

### Specialty/Non-Prime Lenders
| Bank | Notes | Status |
|------|-------|--------|
| Bluestep/Enity Bank Group | Largest specialty lender in Nordics | **Done** |
| Marginalen Bank | Higher approval rates | To Add |
| Nordax Bank/NOBA Bank Group | Non-prime lending | To Add |
| Svea Bank | Non-prime lending | To Add |
| JAK Medlemsbank | Ethical/member-owned, interest-free model | To Add |

## Priority Order for Implementation

### High Priority (Major market presence)
1. ~~**Swedbank** - Largest market share (~25%)~~ **Done**
2. ~~**Handelsbanken** - Second largest (~24%)~~ **Done**
3. ~~**SBAB** - Third largest, state-owned, great data quality~~ **Done**
4. **Länsförsäkringar Bank** - Major player, high satisfaction

### Medium Priority (Significant presence)
5. ~~**Skandiabanken** - Major insurance bank~~ **Done**
6. **Landshypotek Bank** - Niche but significant
7. **Ålandsbanken** - Listed on all comparison sites
8. ~~**Ikano Bank** - Uses Borgo~~ **Done**
9. **Hypoteket** - Growing fintech player
10. ~~**Stabelo** - Growing fintech player~~ **Done**

### Lower Priority (Limited products or specialty)
11. **Avanza Bank** - Variable rate only
12. **Nordnet** - Variable rate only
13. **JAK Medlemsbank** - Limited terms
14. ~~**Bluestep Bank** - Specialty lender~~ **Done**
15. **Marginalen Bank** - Non-prime
16. **Nordax Bank** - Non-prime
17. **Svea Bank** - Non-prime, variable only

## Data Sources

### Official Comparison Sites
- [Konsumenternas.se](https://www.konsumenternas.se/konsumentstod/jamforelser/lan--betalningar/lan/jamfor-borantor/) - **21 banks** (official government-backed)
- [Finansportalen](https://www.finansportalen.se/borantor/) - 15 banks
- [Compricer](https://www.compricer.se/) - 9 banks via Lendo

### Official Sources
- [Finansinspektionen](https://www.fi.se/) - Swedish FSA mortgage reports
- [Svenska Bankföreningen](https://www.swedishbankers.se/) - Swedish Bankers' Association
- [Finance Sweden](https://www.financesweden.se/) - Industry reports

## Notes
- Banks are required by Swedish FSA to publish average interest rates (snitträntor) on their websites since 2015
- Many banks offer both "list rates" (listräntor) and negotiated rates
- Some banks (Hypoteket, SBAB, Stabelo, Danske Bank, Skandia) offer "förhandlingsfria bolån" (non-negotiable/transparent rates)
- Avanza and Nordnet act as intermediaries, offering mortgages from Landshypotek and Stabelo
- Borgo is a mortgage institution used by ICA Banken, Ikano Bank, Sparbanken Syd, Söderberg & Partners, and Ålandsbanken

## LTV (Belåningsgrad) Requirements

Maximum loan-to-value ratios for each bank. Most banks follow the standard Swedish bolånetak (85%), but some fintech/digital lenders have stricter requirements.

| Bank | Max LTV | Min Kontantinsats | Notes |
|------|---------|-------------------|-------|
| **Standard Banks (85% LTV)** ||||
| Danske Bank | 85% | 15% | Better rates at ≤75% LTV |
| SEB | 85% | 15% | Offers kontantinsatslån for up to 10% |
| ICA Banken | 85% | 15% | Requires BRF with min 10 apartments |
| Nordea | 85% | 15% | Offers kontantinsatslån for up to 10% |
| Swedbank | 85% | 15% | Standard bolånetak |
| Handelsbanken | 85% | 15% | Standard bolånetak |
| SBAB | 85% | 15% | State-owned, transparent pricing |
| Länsförsäkringar | 85% | 15% | Standard bolånetak |
| Skandiabanken | 85% | 15% | Better rates at ≤60% LTV |
| Ålandsbanken | 85% | 15% | Requires 1M SEK depot for discount |
| Ikano Bank | 85% | 15% | Via Borgo, no kontantinsatslån |
| JAK Medlemsbank | 85% | 15% | Requires membership, ethical bank |
| Bluestep | 85% | 15% | Offers kontantinsatslån (Hemlån) for up to 10% |
| Marginalen Bank | 85% | 15% | Accepts betalningsanmärkningar |
| Nordax Bank | 85% | 15% | Specialty lender, higher rates |
| Svea Bank | 85% | 15% | Accepts betalningsanmärkningar |
| **Stricter LTV Banks** ||||
| Landshypotek | 75% | 25% | Rural/agricultural property focus |
| Hypoteket | 65% | 35% | Digital-first, förhandlingsfri ränta |
| Stabelo | 60% | 40% | Fintech, via Avanza/Nordnet |
| Avanza (Superbolånet) | 60% | 40% | Via Stabelo, variable rate only |
| Nordnet | 50-60% | 40-50% | Via Stabelo, best rates at ≤50% LTV |

### Key Observations

1. **Standard bolånetak (85%)**: Most traditional banks follow the Swedish FSA's bolånetak, allowing up to 85% LTV. This has been in effect since 2010.

2. **Fintech/Digital lenders (50-65%)**: Banks like Hypoteket, Stabelo, Avanza, and Nordnet offer lower LTV limits but often compensate with better interest rates for borrowers with larger down payments.

3. **Kontantinsatslån**: Some banks (SEB, Nordea, Bluestep) offer additional loans to cover part of the kontantinsats (up to 10% of property value), effectively allowing 95% financing.

4. **Proposed changes (April 2026)**: The Swedish government has proposed raising the bolånetak from 85% to 90%, which would reduce the required kontantinsats from 15% to 10%.

5. **Rate discounts**: Most banks offer better rates at lower LTV levels (typically ≤75% or ≤60%), incentivizing larger down payments.

## Summary
- **Total banks on Konsumenternas.se**: 21
- **Already implemented**: 12 (Danske Bank, SEB, ICA Banken, Nordea, Handelsbanken, SBAB, Swedbank, Skandiabanken, Stabelo, Bluestep, Ikano Bank)
- **Remaining to add**: 9

---

## Implementation Plans

### 1. Swedbank

**Status**: ✅ Implemented
**Difficulty**: Easy
**HTTP Method**: Basic net/http (curl works)

**URLs**:
- List Rates: `https://www.swedbank.se/privat/boende-och-bolan/bolanerantor.html`
- Historic Average Rates: `https://www.swedbank.se/privat/boende-och-bolan/bolanerantor/historiska-genomsnittsrantor.html`

**Data Format**: Static HTML tables embedded in page

**Tables Found**:
1. List rates page: "Aktuella bolåneräntor – listpris" table
2. Historic page: Table with `<caption>` "Våra historiska genomsnittsräntor" (transposed: months as rows, terms as columns)

**Terms Available**: 3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år, 10 år

**Date Format**:
- List rates: "senast ändrad 25 september 2025" in header text
- Historic months: "nov. 2025", "okt. 2025" (abbreviated Swedish months with period)

**Implementation Notes**:
- List rates: Use `utils.FindTokenizedTableByTextBeforeTable()` with "Aktuella bolåneräntor – listpris"
- Historic rates: Use `utils.FindTokenizedTableByTextInCaption()` to find table by caption
- Historic table is transposed (months as rows, terms as columns)
- Abbreviations like "nov.", "okt.", "sep." are handled by extended Swedish month parser

---

### 2. Handelsbanken

**Status**: Ready to implement
**Difficulty**: Easy
**HTTP Method**: Basic net/http (curl works with JSON APIs)

**URLs**:
- List Rates API: `https://www.handelsbanken.se/tron/slana/slan/service/mortgagerates/v1/interestrates`
- Avg Rates API: `https://www.handelsbanken.se/tron/slana/slan/service/mortgagerates/v1/averagerates`

**Data Format**: JSON

**List Rates JSON Structure**:
```json
{
  "interestRates": [
    {
      "effectiveRateValue": {"value": "3,91", "valueRaw": 3.91},
      "periodBasisType": "3",
      "rateValue": {"value": "3,84", "valueRaw": 3.84},
      "term": "3"
    }
  ]
}
```

**Avg Rates JSON Structure**:
```json
{
  "averageRatePeriods": [
    {
      "period": "202412",
      "rates": [
        {"periodBasisType": "3", "rateValue": {"value": "2,87", "valueRaw": 2.87}}
      ]
    }
  ]
}
```

**Terms Available**: 3 mån (term="3"), 1-10 år

**Implementation Notes**:
- Simple JSON parsing, no HTML needed
- `periodBasisType` maps to term (3=3mån, 12=1år, 24=2år, etc.)
- `period` format is "YYYYMM" for average rates
- Use `valueRaw` for numeric parsing

---

### 3. SBAB

**Status**: Ready to implement
**Difficulty**: Easy
**HTTP Method**: Basic net/http (curl works with JSON APIs)

**URLs**:
- List Rates API: `https://www.sbab.se/api/interest-mortgage-service/api/external/v1/interest`
- Avg Rates API: `https://www.sbab.se/api/historical-average-interest-rate-service/interest-rate/average-interest-rate-last-twelve-months-by-period`

**Data Format**: JSON

**List Rates JSON Structure**:
```json
{
  "listInterests": [
    {"period": "P_3_MONTHS", "interestRate": "3.05", "validFrom": "2025-09-29"},
    {"period": "P_1_YEAR", "interestRate": "3.17", "validFrom": "2025-07-04"}
  ]
}
```

**Avg Rates JSON Structure**:
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

**Terms Available**: 3 mån, 1-10 år

**Implementation Notes**:
- Clean JSON APIs, excellent data quality
- List rates use period format like "P_3_MONTHS", "P_1_YEAR", etc.
- Avg rates have column per term with numeric values
- `validFrom` provides change date for list rates
- Null values in avg rates mean insufficient data

---

### 4. Länsförsäkringar

**Status**: Ready to implement
**Difficulty**: Medium
**HTTP Method**: Basic net/http (requires JS rendering OR use PDF)

**URLs**:
- Rates Page: `https://www.lansforsakringar.se/stockholm/privat/bank/bolan/bolaneranta/`
- Historic Avg PDF: `http://lansforsakringar.se/osfiles/00000-bolanerantor-genomsnittliga.pdf`

**Data Format**: Static HTML tables (JS-rendered) + PDF for historic data

**Important**: URL includes regional prefix (e.g., `/stockholm/`). Rates appear same across regions.

**Tables Found**:
1. "Genomsnittlig ränta [month] [year]" - Snitträntor table
2. "Listräntor" - Contains Bindningstid, Ränta, Ändring, Datum

**Terms Available**: 3 mån, 1-10 år

**Implementation Notes**:
- HTML returned by curl is empty - data loaded via JavaScript
- **Option 1**: Use Playwright to render page, then parse HTML
- **Option 2**: Parse the PDF for historic average rates
- **Option 3**: Check if there's an API endpoint (not found in network requests)
- List rates table has 4 columns: Bindningstid, Ränta, Ändring, Datum
- Date format: "YYYY-MM-DD"

**Recommendation**: May need to use Playwright for initial render, then extract data. Consider if worth the complexity vs skipping this bank.

---

### 5. Skandiabanken

**Status**: **Done** ✅
**Difficulty**: Easy
**HTTP Method**: Basic net/http (curl works)

**URLs**:
- Rates Page: `https://www.skandia.se/lana/bolan/bolanerantor/`
- Historic Snitträntor: `https://www.skandia.se/lana/bolan/bolanerantor/snittrantor/`

**Data Format**: JSON embedded in HTML

**Data embedded in page as JSON structures that can be extracted with regex/parsing.

**Tables Found**:
1. "Snitträntor [month] [year]" - Average rates table
2. "Listräntor" - List rates with Bindningstid, Listränta, Senast ändrad

**Terms Available**: 3 mån, 1 år, 2 år, 3 år, 5 år (limited compared to others)

**Implementation Notes**:
- Data is in JSON format embedded in HTML, can be extracted with regex
- Look for patterns like `"name": "Snittränta"` and parse surrounding JSON
- Date format in table: "YYYY-MM-DD"
- Only 5 terms available (no 4 år, 7 år, 10 år)

---

### 6. Landshypotek

**Status**: Ready to implement
**Difficulty**: Easy
**HTTP Method**: Basic net/http (curl works)

**URLs**:
- Rates Page: `https://www.landshypotek.se/lana/privatlan/bolan/aktuella-bolaneranta/`

**Data Format**: Static HTML tables

**Tables Found**:
1. List rates table with Bindningstid, Listränta, Senast ändrad
2. Average rates table (snitträntor) with monthly data

**Terms Available**: 3 mån, 1 år, 2 år, 3 år, 5 år

**Implementation Notes**:
- Standard HTML table parsing with `utils.FindTokenizedTableByTextBeforeTable()`
- Date format: "YYYY-MM-DD"
- Max LTV is 75% (stricter than most banks)
- Focus on rural/agricultural properties

---

### 7. Ålandsbanken

**Status**: ✅ Implemented
**Difficulty**: Easy
**HTTP Method**: Basic net/http (curl works)

**URLs**:
- Rates Page: `https://www.alandsbanken.se/banktjanster/lana-pengar/bolan`

**Data Format**: Static HTML tables

**Tables Found**:
1. "Aktuella räntor:" - List rates with Bindningstid, Räntesats %, Senaste ränteförändring, Förändring %
2. "Genomsnittlig bolåneränta" - Average rates (only 3 mån data available)

**Terms Available**: 3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år, 10 år

**Implementation Notes**:
- Standard HTML table parsing
- Both list and average rates on same page
- Finnish bank operating in Sweden
- Requires 1M SEK depot for rate discount
- List rate date format: "YYYY.MM.DD"
- Average rate month format: "Månad YYYY" (e.g., "Oktober 2025")
- Only publishes average rates for 3 mån term

---

### 8. Ikano Bank

**Status**: ✅ Implemented
**Difficulty**: Easy
**HTTP Method**: Basic net/http (JSON API + HTML)

**URLs**:
- List Rates API: `https://ikanobank.se/api/interesttable/gettabledata`
- Average Rates Page: `https://ikanobank.se/bolan/bolanerantor`

**Note:** Use non-www URLs - the www version redirects and the HTTP client doesn't follow redirects.

**Data Format**: JSON API for list rates, HTML table for average rates

**List Rates JSON Structure**:
```json
{
  "success": true,
  "listData": [
    {
      "rateFixationPeriod": "3 mån",
      "listPriceInterestRate": "3.4800",
      "effectiveInterestRate": "3.5400"
    }
  ]
}
```

**Average Rates Table**:
- Located after text "Snitträntor för bolån"
- Columns: Månad | 3 mån | 1 år | 2 år | 3 år | 4 år | 5 år | 7 år | 10 år
- Month format: "YYYY MM" (e.g., "2025 01")
- Rate format: Swedish decimal with comma (e.g., "3,61 %")

**Terms Available**: 3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år, 10 år

**Implementation Notes**:
- Uses Borgo (same as ICA Banken)
- List rates via JSON API (discovered by inspecting JavaScript source)
- Average rates via HTML table parsing
- No kontantinsatslån available

---

### 9. Hypoteket

**Status**: Ready to implement
**Difficulty**: Easy
**HTTP Method**: Basic net/http (Nuxt.js payload JSON)

**URLs**:
- Rates Page: `https://www.hypoteket.com/bolan/vara-rantor/`
- Data API: `https://www.hypoteket.com/_payload.json` (Nuxt.js payload)

**Data Format**: JSON (Nuxt.js payload)

**JSON Structure**:
The payload contains rate data in a structured format with:
- List rates per term
- Average rates per month
- LTV tier information (max 65%)

**Terms Available**: 3 mån, 1 år, 2 år, 3 år, 5 år

**Implementation Notes**:
- Nuxt.js site - data available in `_payload.json`
- Digital-first bank with "förhandlingsfri ränta" (non-negotiable rates)
- Stricter LTV (65% max)
- Parse JSON payload for rate tables

---

### 10. Stabelo

**Status**: Ready to implement
**Difficulty**: Medium
**HTTP Method**: Basic net/http (JSON embedded in HTML via Remix.js)

**URLs**:
- Rate Table Page: `https://api.stabelo.se/rate-table/`

**Data Format**: JSON embedded in HTML (Remix.js server-rendered)

**How Stabelo Rates Work**:

Stabelo does NOT have traditional "list rates". Instead, they use a **personalized pricing model** with 864 rate entries based on:
1. **Loan Amount** - Volume discounts for larger loans (12 thresholds from 0 to 10M SEK)
2. **LTV Ratio** - Risk-based pricing (6 tiers: base, 60%, 65%, 70%, 75%, 80%)
3. **Green Loan** - 0.10% discount for EPC class A/B properties
4. **Rate Fixation** - 3M, 1Y, 2Y, 3Y, 5Y, 10Y

**List Rate Definition**:

For comparison purposes, the "list rate" is the **worst-case rate** (highest rate offered):
- Loan amount: ≤500k SEK (smallest tier = highest rate)
- LTV: >80% (uses "no LTV" base tier = highest rate)
- No green loan discount

**Rate Data Structure** (embedded in `window.__remixContext`):

```json
{
  "rateTable": {
    "interest_rate_items": [
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
          // No "ltv" field = base tier (>80% LTV)
          // No "epc_classification" = standard rate
        }
      }
    ]
  }
}
```

**List Rate Extraction Logic**:

Filter for entries where:
- `product_configuration.ltv` is **absent** (not present in JSON)
- `product_configuration.epc_classification` is **absent** (not present in JSON)
- `product_configuration.product_amount.value` is `0` (smallest loan tier)

This gives the worst-case rate for each term.

**Volume Discount Tiers** (loan amount → rate reduction for 3M term):

| Loan Amount | Rate    | Discount vs Base |
|-------------|---------|------------------|
| 0-500k      | 3.33%   | Base (list rate) |
| 600k        | 3.27%   | -0.06%           |
| 700k        | 3.21%   | -0.12%           |
| 800k        | 3.16%   | -0.17%           |
| 900k        | 3.10%   | -0.23%           |
| 1M          | 2.94%   | -0.39%           |
| 1.5M        | 2.86%   | -0.47%           |
| 2M+         | 2.75%   | -0.58%           |

**LTV Tiers** (for 2M loan, 3M term):

| LTV Range | Rate    | Premium vs Best |
|-----------|---------|-----------------|
| ≤75%      | 2.54%   | Best rate       |
| 75-80%    | 2.67%   | +0.13%          |
| >80%      | 2.75%   | +0.21%          |

**Terms Available**: 3M, 1Y, 2Y, 3Y, 5Y, 10Y

**Implementation Plan**:

```bash
# Step 1: Fetch the rate table HTML page
curl -s 'https://api.stabelo.se/rate-table/' -H 'User-Agent: Mozilla/5.0' > stabelo.html

# Step 2: Extract the Remix context JSON from the HTML
# Look for: window.__remixContext = {...}
# The data is in: __remixContext.state.loaderData["routes/_index"].rateTable.interest_rate_items

# Step 3: Parse JSON and filter for list rates
# Filter criteria:
#   - No "ltv" field in product_configuration
#   - No "epc_classification" field in product_configuration
#   - product_amount.value == 0

# Step 4: Extract rate for each rate_fixation (3M, 1Y, 2Y, 3Y, 5Y, 10Y)
```

**Go Implementation Approach**:

1. Fetch HTML from `https://api.stabelo.se/rate-table/`
2. Use regex to extract JSON from `<script>` tag containing `window.__remixContext`
3. Parse the JSON and navigate to `state.loaderData["routes/_index"].rateTable.interest_rate_items`
4. Filter entries: no `ltv`, no `epc_classification`, `product_amount.value == 0`
5. Map `rate_fixation` to `model.Term` and `interest_rate.bps/100` to rate

**Average Rates (PDF)**:

Stabelo publishes average rates in a PDF document linked from `https://www.stabelo.se/bolanerantor`.

**PDF URL Pattern**: `https://www.stabelo.se/documents/StabeloGenomsnittsräntor{Month}{Year}.pdf`
- Month: Swedish month name (januari, februari, mars, april, maj, juni, juli, augusti, september, oktober, november, december)
- Example: `StabeloGenomsnittsräntorOktober2025.pdf`

**Extraction Steps**:
```bash
# Step 1: Find PDF URL from page
PDF_PATH=$(curl -s 'https://www.stabelo.se/bolanerantor' -H 'User-Agent: Mozilla/5.0' \
  | grep -oP 'href="\K[^"]*[Gg]enomsnitts[^"]*\.pdf(?=")' | head -1)

# Step 2: Download PDF
curl -sL "https://www.stabelo.se/$PDF_PATH" -H 'User-Agent: Mozilla/5.0' -o stabelo_avg.pdf
```

**PDF Table Format**:
| Bindningstid | 3 mån | 1 år | 2 år | 3 år | 5 år | 10 år |
|--------------|-------|------|------|------|------|-------|
| oktober 2025 | 2,61% | 2,52% | 2,89% | 2,96% | 3,20% | - |

- Missing values shown as "-"
- Rates in Swedish decimal format (comma separator)
- Data goes back to November 2017
- Updated on 5th business day each month

**Notes**:
- Max LTV is 85% (not 60% as previously thought)
- The current implementation parses HTML buttons, but should be updated to extract from Remix JSON for accuracy
- Average rates require PDF parsing (Go library like `pdfcpu` or `unipdf`)

---

### 11. Avanza Bank

**Status**: Ready to implement
**Difficulty**: Medium
**HTTP Method**: Basic net/http (Multiple JSON APIs)

**URLs**:
- Superbolånet Page: `https://www.avanza.se/bolan/superbollanet.html`
- Stabelo Rates API: `https://www.avanza.se/ab/component/bolan_raknare/get_interest_rates/stabelo`
- Landshypotek Rates API: `https://www.avanza.se/ab/component/bolan_raknare/get_interest_rates/landshypotek`

**Data Format**: JSON APIs

**Stabelo API Response**:
```json
{
  "interestRates": [
    {
      "bindingPeriod": "3 mån",
      "ltv": [
        {"maxLtv": 50, "interestRate": 1.89},
        {"maxLtv": 60, "interestRate": 2.09}
      ]
    }
  ]
}
```

**Landshypotek API Response**:
```json
{
  "interestRates": [
    {
      "bindingPeriod": "3 mån",
      "ltv": [
        {"maxLtv": 60, "interestRate": 2.15},
        {"maxLtv": 75, "interestRate": 2.35}
      ]
    }
  ]
}
```

**Terms Available**:
- Stabelo: 3 mån, 1 år, 2 år, 3 år, 5 år, 10 år
- Landshypotek: 3 mån, 1 år, 2 år, 3 år, 5 år

**Implementation Notes**:
- Avanza is intermediary offering Stabelo and Landshypotek mortgages
- Two separate API calls needed
- Rates include LTV tier information
- Superbolånet is variable rate only (3 mån)

---

### 12. Nordnet

**Status**: Ready to implement
**Difficulty**: Medium
**HTTP Method**: Basic net/http (Contentful CMS API)

**URLs**:
- Rates Page: `https://www.nordnet.se/se/tjanster/lan/bolan/stabelo`
- Contentful CMS API: `https://api.prod.nntech.io/cms/v1/contentful-cache/spaces/main_se/environments/master/entries?include=5&sys.id=36p8FGv6CCUfUIiXPjPBJy`

**Data Format**: JSON (Contentful CMS)

**API Response Structure**:
The Contentful API returns entries including a `componentTable` entry with rate data:
- Entry ID for rate table: `11JscTUKd02VILjvUjwyjn`
- Table contains LTV tiers: <75%, 75-80%, 80-85%
- Rows contain terms and rates

**Terms Available**: 3 mån, 1 år, 2 år, 3 år, 5 år, 10 år

**Implementation Notes**:
- Uses Contentful CMS for content management
- Need to parse nested JSON structure to find `componentTable` entries
- Rate table is embedded in Contentful entry as JSON
- Nordnet uses Stabelo as mortgage provider
- Best rates at ≤50% LTV

---

### 13. JAK Medlemsbank

**Status**: Ready to implement
**Difficulty**: Easy
**HTTP Method**: Basic net/http (curl works)

**URLs**:
- Rates Page: `https://www.jak.se/snittranta/`

**Data Format**: Static HTML tables

**Tables Found**:
1. List rates table with current rates
2. Average rates (snitträntor) table with monthly data

**Current Rates Example**:
- 3 månader: 3.58% (list), 3.24% (avg)
- 12 månader: 2.79% (list), 2.42% (avg)

**Terms Available**: 3 mån, 12 mån (only 2 terms!)

**Implementation Notes**:
- Ethical/cooperative bank with "sparlånesystem"
- Only offers 2 binding periods (very limited)
- Standard HTML table parsing
- Requires membership to get mortgage

---

### 14. Bluestep

**Status**: ✅ Implemented
**Difficulty**: Easy
**HTTP Method**: Basic net/http (curl works)

**URLs**:
- List Rates: `https://www.bluestep.se/bolan/borantor/`
- Average Rates: `https://www.bluestep.se/bolan/borantor/genomsnittsrantor/`

**Data Format**: Static HTML tables

**Tables Found**:
1. List rates table: Terms in header row (Rörlig 3 månader, Fast 3 år, Fast 5 år), rates in second row
2. Average rates table: Månad, 3 mån, 1 år, 3 år, 5 år

**Current Rates Example**:
- 3 mån: 4.45%
- 3 år: 4.60%
- 5 år: 4.68%

**Terms Available**:
- List rates: 3 mån, 3 år, 5 år
- Average rates: 3 mån, 1 år, 3 år, 5 år

**Implementation Notes**:
- Specialty/non-prime lender (higher rates)
- Two separate pages for list and average rates
- List rates table uses `<td>` for both header and data rows (no `<th>`)
- Search for "Bolån*" text before table to locate list rates
- Average rates have standard `<th>` header row
- Month format: "YYYY MM" (e.g., "2025 11")
- Rate format: Swedish decimal with comma (e.g., "4,45%")
- Rate range: 4.45% - 9.30%

---

### 15. Marginalen Bank

**Status**: Ready to implement
**Difficulty**: Medium
**HTTP Method**: Basic net/http (curl works)

**URLs**:
- List Rates: `https://www.marginalen.se/privat/banktjanster/lan/bolan/`
- Average Rates: `https://www.marginalen.se/privat/banktjanster/lan/flytta-eller-utoka-bolan/genomsnittlig-bolaneranta/`

**Data Format**: HTML content with embedded rates

**Rate Information**:
- Rate range: 4.41% - 10.50%
- Binding times: 3 months - 3 years

**Terms Available**: 3 mån, 1 år, 2 år, 3 år

**Implementation Notes**:
- Specialty/non-prime lender
- Accepts customers with betalningsanmärkningar (payment remarks)
- Rates are displayed in content sections, not clean tables
- May need more complex HTML parsing to extract rate data
- Complex rate structure based on risk profile

---

### 16. Nordax Bank

**Status**: Ready to implement
**Difficulty**: Medium
**HTTP Method**: Basic net/http (curl works)

**URLs**:
- Rates Page: `https://www.nordax.se/lana/bolan`

**Data Format**: HTML content with embedded rates

**Rate Information**:
- Rate range: 4.45% - 9.94%
- Rates shown in "Räkneexempel bolån" table

**Terms Available**: 3 mån (rörlig), 3 år, 5 år

**Implementation Notes**:
- Specialty/non-prime lender (NOBA Bank Group)
- Limited binding periods
- Rates embedded in HTML content/tables
- May need to parse rate calculator section

---

### 17. Svea Bank

**Status**: Ready to implement (Average rates only)
**Difficulty**: Easy
**HTTP Method**: Basic net/http (curl works)

**URLs**:
- Average Rates: `https://www.svea.com/sv-se/privat/låna/bolån/snitträntor`

**Data Format**: Static HTML table

**Tables Found**:
1. Average rates table: Månad för utbetalning, Räntesats

**Important**: Only publishes average rates (snitträntor), NOT list rates!

**Terms Available**: Variable rate only (rörlig ränta)

**Implementation Notes**:
- Specialty/non-prime lender
- Only offers variable rate mortgages
- Only publishes monthly average rates, no list rates
- Simple HTML table with month and rate columns
- Accepts customers with betalningsanmärkningar

---

## Implementation Order Recommendation

Based on difficulty and market importance:

### Phase 1 (Easy - JSON APIs)
1. **Handelsbanken** - Clean JSON APIs, Big Four
2. **SBAB** - Clean JSON APIs, major player
3. **Stabelo** - Clean JSON API
4. **Avanza Bank** - JSON APIs for Stabelo/Landshypotek

### Phase 2 (Easy - HTML tables)
5. **Swedbank** - HTML tables, Big Four
6. **Skandiabanken** - JSON in HTML
7. **Landshypotek** - HTML tables
8. **Ålandsbanken** - HTML tables
9. **Ikano Bank** - HTML tables (Borgo)
10. **JAK Medlemsbank** - HTML tables (only 2 terms)
11. **Bluestep** - HTML tables

### Phase 3 (Medium - Special parsing)
12. **Hypoteket** - Nuxt.js payload JSON
13. **Nordnet** - Contentful CMS API
14. **Länsförsäkringar** - JS-rendered (may need Playwright)
15. **Marginalen Bank** - Complex HTML parsing
16. **Nordax Bank** - HTML content parsing
17. **Svea Bank** - Average rates only
