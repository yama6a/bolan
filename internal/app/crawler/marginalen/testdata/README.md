# Marginalen Bank - Test Data

## Bank Information

- **Bank**: Marginalen Bank
- **Type**: Specialty/non-prime lender
- **Website**: https://www.marginalen.se
- **Rate Range**: 4,41 % - 10,50 %
- **Binding Times**: 3 månader - 3 år
- **Max LTV**: 85%
- **Special Notes**: Accepts customers with betalningsanmärkningar (payment remarks), individual credit assessment

## Data Sources

### Average Rates (Snitträntor)
- **API URL**: https://www.marginalen.se/api/episerver/v3.0/content?contentUrl=%2Fprivat%2Fbanktjanster%2Flan%2Fflytta-eller-utoka-bolan%2Fgenomsnittlig-bolaneranta%2F&matchExact=true&expand=*
- **Web Page URL**: https://www.marginalen.se/privat/banktjanster/lan/flytta-eller-utoka-bolan/genomsnittlig-bolaneranta/ (for reference)
- **Method**: HTTP GET to Episerver Content Delivery API
- **Authentication**: None
- **Format**: JSON response with HTML embedded in the `body` field
- **CMS**: Episerver (now Optimizely) with Vue.js frontend

### List Rates (Listräntor)
**Not Available** - Marginalen Bank does not publish specific list rates per term. They only publish a rate range (4,41 % - 10,50 %) because rates are set individually based on credit assessment.

## Data Formats

### API Response Structure

The Episerver API returns JSON with nested structure:

```json
[
  {
    "mainContentArea": [
      {
        "mainContentArea": [
          {
            "body": "<!DOCTYPE html>..."
          }
        ]
      }
    ]
  }
]
```

The HTML is embedded in `[0].mainContentArea[0].mainContentArea[0].body`.

### Average Rates Table

The embedded HTML contains a single table with the following structure:

```html
<table>
  <tbody>
    <tr>
      <th><strong>Månad</strong></th>
      <th><strong>3 Mån</strong></th>
      <th><strong>6 Mån</strong></th>
      <th><strong>1 år</strong></th>
      <th><strong>2 år</strong></th>
      <th><strong>3 år</strong></th>
    </tr>
    <tr>
      <td><strong>202412</strong></td>
      <td>5,92 %</td>
      <td>-</td>
      <td>5,31 %</td>
      <td>-</td>
      <td>-</td>
    </tr>
    ...
  </tbody>
</table>
```

#### Column Details:
- **Månad**: Period in YYYYMM format (e.g., "202412" for December 2024)
- **3 Mån**: 3 months variable rate
- **6 Mån**: 6 months fixed rate
- **1 år**: 1 year fixed rate
- **2 år**: 2 years fixed rate
- **3 år**: 3 years fixed rate

#### Notes:
- Missing values are shown as "-"
- Rates include Swedish decimal format with comma (e.g., "5,92 %")
- Period format is YYYYMM (no separator)
- Table is found after "Genomsnittlig bolåneränta" heading

## Test Data Files

### marginalen_api_response.json
- **Description**: API response from Episerver Content Delivery API
- **Date Retrieved**: 2025-12-06
- **Terms Available**: 3 Mån, 6 Mån, 1 år, 2 år, 3 år
- **Months Covered**: 12 months (202412 to 202511)
- **Format**: JSON with HTML embedded in nested `body` field

### marginalen_avg_rates.html
- **Description**: Extracted HTML content from API response (for reference)
- **Date Retrieved**: 2025-12-06
- **Note**: This is the HTML extracted from the API's JSON response

### marginalen_bolan.html
- **Description**: Main bolån page (for reference)
- **Date Retrieved**: 2025-12-06
- **Contains**: Rate range information, terms of service, general information
- **Note**: Does not contain specific list rates per term

## Refreshing Test Data

Marginalen uses Episerver (Optimizely) CMS with a Content Delivery API. The Vue.js frontend fetches content from this API, but you can fetch the same data directly via HTTP.

### Updating Test Data

To update the golden files with fresh data:

1. **Average Rates API Response**:
   ```bash
   curl -s 'https://www.marginalen.se/api/episerver/v3.0/content?contentUrl=%2Fprivat%2Fbanktjanster%2Flan%2Fflytta-eller-utoka-bolan%2Fgenomsnittlig-bolaneranta%2F&matchExact=true&expand=*' \
     -H 'User-Agent: Mozilla/5.0' \
     -H 'Accept: application/json' \
     > internal/app/crawler/marginalen/testdata/marginalen_api_response.json
   ```

2. **Extract HTML (optional, for reference)**:
   ```bash
   # Use jq to extract the HTML body from JSON:
   jq -r '.[0].mainContentArea[0].mainContentArea[0].body' \
     internal/app/crawler/marginalen/testdata/marginalen_api_response.json \
     > internal/app/crawler/marginalen/testdata/marginalen_avg_rates.html
   ```

3. **Main Bolån Page** (for reference):
   ```bash
   curl -s 'https://www.marginalen.se/privat/banktjanster/lan/bolan/' \
     -H 'User-Agent: Mozilla/5.0' \
     > internal/app/crawler/marginalen/testdata/marginalen_bolan.html
   ```

## Implementation Notes

1. **No List Rates**: Marginalen only publishes a rate range, not specific list rates per term. Implementation only extracts average rates.

2. **Episerver CMS API**: Marginalen uses Episerver (now Optimizely) CMS with a Content Delivery API. The Vue.js frontend is just a presentation layer - the actual data is fetched from the API at `/api/episerver/v3.0/content`. The crawler uses this API directly, avoiding the need for JavaScript execution.

3. **Nested JSON Structure**: The API returns JSON with the HTML content nested in `[0].mainContentArea[0].mainContentArea[0].body`. The crawler extracts this HTML and then parses the table using standard HTML parsing.

4. **Period Format**: Unlike most banks that use Swedish month names or YYYY-MM format, Marginalen uses YYYYMM format (e.g., "202412").

5. **Missing Values**: Many entries have "-" indicating no loans were issued for that term in that month (fewer than 5 loans required for average calculation per FSA rules).

6. **Individual Pricing**: Marginalen specializes in non-prime lending with individual credit assessment, hence the wide rate range and lack of published list rates.

## HTTP Request Example

**API Request** (this is what the crawler uses):

```
GET /api/episerver/v3.0/content?contentUrl=%2Fprivat%2Fbanktjanster%2Flan%2Fflytta-eller-utoka-bolan%2Fgenomsnittlig-bolaneranta%2F&matchExact=true&expand=* HTTP/1.1
Host: www.marginalen.se
User-Agent: Mozilla/5.0
Accept: application/json
Connection: keep-alive
```

The response contains JSON with the HTML table embedded in the nested `body` field.

**Web Page Request** (for reference - returns Vue.js shell):

```
GET /privat/banktjanster/lan/flytta-eller-utoka-bolan/genomsnittlig-bolaneranta/ HTTP/1.1
Host: www.marginalen.se
User-Agent: Mozilla/5.0
Accept: text/html
Connection: keep-alive
```

This returns only the Vue.js shell with `<div id="app">`, not the actual data. The Vue.js app then fetches data from the API endpoint above.

No authentication or API keys required for either endpoint.
