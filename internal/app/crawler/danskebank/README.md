## Danske Bank

Danske Bank uses a single HTML page containing both list rates and average rates.

### Fetch Page

**Minimal working request:**

```bash
curl -s 'https://danskebank.se/privat/produkter/bolan/relaterat/aktuella-bolanerantor' \
  -H 'User-Agent: Mozilla/5.0'
```

### List Rates Table

**Table identifier:** Search for text "Läs mer om listräntor" before the table, then skip 1 table (first is empty style
table)

**Table structure (4 columns):**
| Bindningstid | Ändrad | Förändring | Listränta |
|--------------|--------|------------|-----------|
| 3 mån | 2025-10-06 | -0,20 | 3.33% |

**Data formats:**

- Term: Swedish format
- Date: YYYY-MM-DD
- Rate: Decimal with dot (3.33%)

### Average Rates Table

**Table identifier:** Search for text "Historiska snitträntor" before the table

**Table structure:**
| Period | 3 mån | 1 år | 2 år | 3 år | ...
|--------|-------|------|------|------|
| Augusti 2021 | 1,23 | 1,44 | 1,66 | ... |

**Data formats:**

- Month: Swedish month name with year (e.g., "Augusti 2021", "Feb 1955")
- Rate: Swedish decimal format with comma

**Note:** Danske Bank has inconsistent HTML table formatting where some rows are split across multiple `<tr>` elements.
The crawler handles this by detecting rows with only a month name and merging with the following row.

---

