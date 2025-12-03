# Swedish Mortgage Banks - Research

## Currently Implemented Crawlers
- [x] Danske Bank
- [x] SEB
- [x] ICA Banken
- [x] Nordea

## Complete Bank List (from Konsumenternas.se - 21 banks)

The following banks are listed on the official Konsumenternas.se comparison (updated 2025-11-28):

| # | Bank | Website | Status | Notes |
|---|------|---------|--------|-------|
| 1 | Avanza Bank | avanza.se | To Add | Superbolånet product, variable rate only (2.01-2.25%) |
| 2 | Bluestep/Enity Bank Group | bluestep.se | To Add | Specialty lender, 3 mån/3 år/5 år terms, high rate range (4.45-9.30%) |
| 3 | Danske Bank | danskebank.se | **Done** | Full term range |
| 4 | Handelsbanken | handelsbanken.se | **Done** | Big Four, full term range |
| 5 | Hypoteket | hypoteket.com | To Add | Digital-first, data currently missing on Konsumenternas |
| 6 | ICA Banken | icabanken.se | **Done** | Full term range |
| 7 | Ikano Bank | ikanobank.se | To Add | Uses Borgo, full term range |
| 8 | JAK Medlemsbank | jak.se | To Add | Ethical bank, only 3 mån and 1 år terms |
| 9 | Landshypotek | landshypotek.se | To Add | Also via Avanza/Bolån+, terms up to 5 år |
| 10 | Länsförsäkringar (LF) | lansforsakringar.se | To Add | Major player, full term range |
| 11 | Marginalen Bank | marginalen.se | To Add | Specialty lender, complex rate structure (4.46-10.56%) |
| 12 | Nordax Bank/NOBA Bank Group | nordax.se | To Add | Specialty lender, 3 mån/3 år/5 år terms (4.45-9.94%) |
| 13 | Nordea | nordea.se | **Done** | Big Four |
| 14 | Nordnet | nordnet.se | To Add | Variable rate only (1.80-3.31%) |
| 15 | SBAB | sbab.se | To Add | State-owned, highest satisfaction, full term range |
| 16 | SEB | seb.se | **Done** | Big Four |
| 17 | Skandiabanken | skandiabanken.se | To Add | Insurance bank, terms up to 5 år |
| 18 | Stabelo | stabelo.se | To Add | Also via Avanza/Bolån+ and Nordnet, data currently missing |
| 19 | Svea Bank | svea.com | To Add | Specialty lender, variable rate only (from 5.65%) |
| 20 | Swedbank | swedbank.se | To Add | Big Four, largest market share (~25%), full term range |
| 21 | Ålandsbanken | alandsbanken.se | To Add | Finnish bank in Sweden, full term range |

## Banks by Category

### Big Four (~71% market share)
| Bank | Market Share | Status |
|------|--------------|--------|
| Swedbank | ~25% | To Add |
| Handelsbanken | ~24% | **Done** |
| SEB | ~15% | **Done** |
| Nordea | ~14% | **Done** |

### Major Banks & Mortgage Institutions
| Bank | Notes | Status |
|------|-------|--------|
| SBAB | ~8.5% share, state-owned, highest customer satisfaction | To Add |
| Länsförsäkringar Bank | Insurance company's bank, high satisfaction | To Add |
| Skandiabanken | Insurance/pension company's bank | To Add |
| Danske Bank | - | **Done** |
| Landshypotek Bank | Agricultural/rural property focus | To Add |
| Ålandsbanken | Finnish bank operating in Sweden | To Add |
| ICA Banken | Uses Borgo | **Done** |
| Ikano Bank | Uses Borgo | To Add |

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
| Bluestep/Enity Bank Group | Largest specialty lender in Nordics | To Add |
| Marginalen Bank | Higher approval rates | To Add |
| Nordax Bank/NOBA Bank Group | Non-prime lending | To Add |
| Svea Bank | Non-prime lending | To Add |
| JAK Medlemsbank | Ethical/member-owned, interest-free model | To Add |

## Priority Order for Implementation

### High Priority (Major market presence)
1. **Swedbank** - Largest market share (~25%)
2. **Handelsbanken** - Second largest (~24%)
3. **SBAB** - Third largest, state-owned, great data quality
4. **Länsförsäkringar Bank** - Major player, high satisfaction

### Medium Priority (Significant presence)
5. **Skandiabanken** - Major insurance bank
6. **Landshypotek Bank** - Niche but significant
7. **Ålandsbanken** - Listed on all comparison sites
8. **Ikano Bank** - Uses Borgo
9. **Hypoteket** - Growing fintech player
10. **Stabelo** - Growing fintech player

### Lower Priority (Limited products or specialty)
11. **Avanza Bank** - Variable rate only
12. **Nordnet** - Variable rate only
13. **JAK Medlemsbank** - Limited terms
14. **Bluestep Bank** - Specialty lender
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
- **Already implemented**: 5 (Danske Bank, SEB, ICA Banken, Nordea, Handelsbanken)
- **Remaining to add**: 16

---

## Implementation Plans

### 1. Swedbank

**Status**: Ready to implement
**Difficulty**: Easy
**HTTP Method**: Basic net/http (curl works)

**URLs**:
- List Rates & Avg Rates: `https://www.swedbank.se/privat/boende-och-bolan/bolanerantor.html`

**Data Format**: Static HTML tables embedded in page

**Tables Found**:
1. "Genomsnittsränta" (Average rates) - contains snitträntor per bindningstid
2. "Aktuella bolåneräntor" (List rates) - contains listräntor per bindningstid

**Terms Available**: 3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år, 10 år, Banklån

**Date Format**: "senast ändrad 25 september 2025" in header text

**Implementation Notes**:
- Use `utils.FindTokenizedTableByTextBeforeTable()` to find tables
- Parse date from header text using regex
- Similar pattern to existing Nordea/ICA crawlers

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

**Status**: Ready to implement
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

**Status**: Ready to implement
**Difficulty**: Easy
**HTTP Method**: Basic net/http (curl works)

**URLs**:
- Rates Page: `https://www.alandsbanken.se/bolan/bolanerantor`

**Data Format**: Static HTML tables

**Tables Found**:
1. "Aktuella bolåneräntor" - List rates with Bindningstid, Ränta, Senast ändrad
2. "Snitträntor" - Average rates by month

**Terms Available**: 3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år, 10 år

**Implementation Notes**:
- Standard HTML table parsing
- Finnish bank operating in Sweden
- Requires 1M SEK depot for rate discount

---

### 8. Ikano Bank

**Status**: Ready to implement
**Difficulty**: Easy
**HTTP Method**: Basic net/http (curl works)

**URLs**:
- Rates Page: `https://www.ikanobank.se/privat/bolan/bolaneranta/`

**Data Format**: Static HTML tables

**Tables Found**:
1. List rates table with Bindningstid, Listränta, Senast ändrad
2. Average rates table with monthly snitträntor

**Terms Available**: 3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år, 10 år

**Implementation Notes**:
- Uses Borgo (same as ICA Banken) - similar table structure
- Standard HTML parsing
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
**Difficulty**: Easy
**HTTP Method**: Basic net/http (JSON API)

**URLs**:
- Rates Page: `https://www.stabelo.se/bolan/`
- Rate Table API: `https://www.stabelo.se/api/rate-table`

**Data Format**: JSON API

**JSON Structure**:
```json
{
  "rates": [
    {
      "term": "3_months",
      "ltv_tiers": [
        {"max_ltv": 50, "rate": 1.89},
        {"max_ltv": 60, "rate": 2.09}
      ]
    }
  ]
}
```

**Terms Available**: 3 mån, 1 år, 2 år, 3 år, 5 år, 10 år

**Implementation Notes**:
- Clean JSON API for rate data
- Rates vary by LTV tier (≤50%, 50-60%)
- Fintech lender, also available via Avanza and Nordnet
- Max LTV 60%

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

**Status**: Ready to implement
**Difficulty**: Easy
**HTTP Method**: Basic net/http (curl works)

**URLs**:
- List Rates: `https://www.bluestep.se/bolan/borantor/`
- Average Rates: `https://www.bluestep.se/bolan/borantor/genomsnittsrantor/`

**Data Format**: Static HTML tables

**Tables Found**:
1. List rates table: Bindningstid, Ränta, Senast ändrad
2. Average rates table: Månad, 3 mån, 1 år, 3 år, 5 år

**Current Rates Example**:
- 3 mån: 4.45%
- 3 år: 4.60%
- 5 år: 4.68%

**Terms Available**: 3 mån, 3 år, 5 år (limited terms)

**Implementation Notes**:
- Specialty/non-prime lender (higher rates)
- Two separate pages for list and average rates
- Standard HTML table parsing
- Offers kontantinsatslån (Hemlån) for up to 10%
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
