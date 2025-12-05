# Länsförsäkringar Crawler

## Data Source

- **URL**: `https://www.lansforsakringar.se/stockholm/privat/bank/bolan/bolaneranta/`
- **Format**: Static HTML tables
- **Auth Required**: No
- **Min Headers**: User-Agent

> **Note**: Länsförsäkringar uses regional URLs (e.g., `/stockholm/`, `/goteborg/`). Rates are the same across regions.

## Rate Types

### List Rates (Listräntor)

Published as a table with columns:

- Bindningstid (binding period)
- Ränta (rate)
- Ändring (change)
- Datum (date)

**Terms Available**: 3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år, 10 år

**Date Format**: `YYYY-MM-DD`

### Average Rates (Snitträntor)

Published as a table with columns:

- Bindningstid (binding period)
- Genomsnittlig ränta [month] [year] (average rate for the specified month)

Average rates are updated on the fifth working day of each month.

**Terms Available**: 3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år (sometimes empty), 10 år

## Golden Files

- `lansforsakringar_rates.html` - Main rates page with both list and average rates

## Refresh Golden Files

```bash
curl -s 'https://www.lansforsakringar.se/stockholm/privat/bank/bolan/bolaneranta/' \
  -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36' \
  -o internal/app/crawler/lansforsakringar/testdata/lansforsakringar_rates.html
```

## Historical Average Rates PDF

Länsförsäkringar also publishes a PDF with historical average rates:
`http://lansforsakringar.se/osfiles/00000-bolanerantor-genomsnittliga.pdf`

This crawler currently only extracts the current month's average rates from the HTML page.

## Rate Format

- **Rate format**: Swedish decimal with comma (e.g., "3,84 %")
- **Date format**: ISO format (e.g., "2025-10-02")
- **Month format**: Swedish month names (e.g., "oktober 2025")

## Notes

- Länsförsäkringar is one of Sweden's largest insurance companies with a significant banking operation
- They have high customer satisfaction ratings in the bolån (mortgage) segment
- Rates vary by customer profile and engagement level
- Max LTV is 85% (standard Swedish bolånetak)
