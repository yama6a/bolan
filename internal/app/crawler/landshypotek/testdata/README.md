# Landshypotek Crawler Test Data

## Data Sources

- **Rates Page**: https://www.landshypotek.se/lana/bolanerantor/
- **Bank**: Landshypotek Bank (rural/agricultural property specialist)

## Test Files

### `landshypotek_rates.html`

**Source**: https://www.landshypotek.se/lana/bolanerantor/

**Content**: HTML page containing both list rates and average rates
- List rates shown for two LTV tiers (60% and 75%)
- Recent month average rates (expandable section)
- Historical average rates for 12 months (expandable section)

**Captured**: 2025-12-04

**Refresh command**:
```bash
curl -s 'https://www.landshypotek.se/lana/bolanerantor/' \
  -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 13_5_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36' \
  -H 'Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8' \
  -H 'Accept-Language: sv-SE,sv;q=0.9,en;q=0.8' \
  -o internal/app/crawler/landshypotek/testdata/landshypotek_rates.html
```

## Data Format

**Note**: This crawler extracts 5 different rate types from the page, all available in the static HTML (not JavaScript-loaded).

### 1. Discounted Rates (Erbjudna bolåneräntor)

The page shows **discounted rates** (after rabatt) for two LTV (belåningsgrad) tiers:
- **60% or lower**: Best discounted rates (TypeRatioDiscounted, 0-60% LTV)
- **60-75%**: Higher discounted rates (TypeRatioDiscounted, 60-75% LTV)

**Table Structure**: 3 columns
- Bindningstid (term)
- Ränta (rate)
- Effektiv ränta (effective rate)

**Example (60% LTV tier)**:
```
Bindningstid | Ränta   | Effektiv ränta
3 mån        | 2,54 %  | 2,57 %
1 år         | 2,69 %  | 2,72 %
2 år         | 2,90 %  | 2,94 %
3 år         | 3,00 %  | 3,04 %
4 år         | 3,15 %  | 3,20 %
5 år         | 3,25 %  | 3,30 %
```

### 2. List Rates (Listräntor)

Expandable section "Listräntor för bolån" shows **list rates before discount** (TypeListRate).

**Table Structure**: 3 columns (same as discounted rates)

**Example**:
```
Bindningstid | Ränta   | Effektiv ränta
3 mån        | 3,04 %  | 3,07 %
1 år         | 3,19 %  | 3,23 %
2 år         | 3,40 %  | 3,44 %
3 år         | 3,50 %  | 3,55 %
4 år         | 3,65 %  | 3,70 %
5 år         | 3,75 %  | 3,80 %
```

### 3. Current Month Average Rates (Snitträntor senaste månaden)

Expandable section "Snitträntor för bolån senaste månaden" shows current month average rates (TypeAverageRate).

**Table Structure**: 2 columns
- Bindningstid
- Snittränta

**Header**: Contains `<h4>` tag with month name (e.g., "Oktober")

**Note**: Some terms may show "n/a" if fewer than 5 loans were issued with that term.

**Example**:
```
Oktober
Bindningstid | Snittränta
3 mån        | 2,55 %
1 år         | 2,71 %
```

### 4. Historical Average Rates (Historisk snittränta)

Expandable section "Historisk snittränta för bolån" shows 12 months of historical data (TypeAverageRate).

**Table Structure**: 8 columns
- År (year)
- Månad (month)
- 3 mån
- 1 år
- 2 år
- 3 år
- 4 år
- 5 år

**Example**:
```
År   | Månad     | 3 mån  | 1 år   | 2 år   | 3 år   | 4 år | 5 år
2025 | Oktober   | 2,55 % | 2,71 % | 2,76 % | 2,89 % | n/a  | 3,11 %
2025 | September | 2,75 % | 2,71 % | 2,84 % | 2,91 % | 2,98%| 3,13 %
```

**Month Format**: Full Swedish month name (e.g., "Oktober", "September")

**Missing Data**: Shown as "n/a" when fewer than 5 loans with that term

## Terms Available

All rate types use the same 6 terms:
- **3 mån, 1 år, 2 år, 3 år, 4 år, 5 år**

Note:
- Landshypotek has a stricter 75% max LTV compared to most banks (85%)
- Average rates may show "n/a" for some terms when fewer than 5 loans were issued

## Rate Format

- Swedish decimal format with comma separator: "2,54 %"
- Percentage sign included in HTML
- Missing values shown as "n/a"

## Implementation Notes

- Page requires standard HTTP headers (User-Agent, Accept, Accept-Language)
- All rates are in the initial HTML response (not JavaScript-loaded)
- Expandable sections use HTML entities (e.g., `&#xE4;` for ä, `&#xF6;` for ö)
- Historical table uses `<th scope="row">` for year column instead of `<td>`

## Extracted Rate Types

The crawler extracts **5 rate types** totaling ~80+ interest sets:
1. **Discounted rates 60% LTV** (6 rates) - TypeRatioDiscounted, ratio 0-60%
2. **Discounted rates 75% LTV** (6 rates) - TypeRatioDiscounted, ratio 60-75%
3. **List rates** (6 rates) - TypeListRate, before discount
4. **Current month average rates** (~2-6 rates) - TypeAverageRate, some terms may be n/a
5. **Historical average rates** (~60-72 rates) - TypeAverageRate, 12 months × 5-6 terms

## Parsing Strategy

1. **Discounted rates**: Use `FindTokenizedTableByTextInCaption` with "belåningsgrad 60" and "belåningsgrad 75"
2. **List rates**: Search for HTML entity "Listr&#xE4;ntor f&#xF6;r bol&#xE5;n", then find table by caption
3. **Current month avg**: Search for "Snittr&#xE4;ntor", extract month from `<h4>` tag, find table by "Bindningstid"
4. **Historical avg**: Search for "Historisk snittr&#xE4;nta", find table by "bindningstid" text in paragraph before table
