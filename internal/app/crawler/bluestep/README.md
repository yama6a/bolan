## Bluestep

Bluestep is a specialty/non-prime lender with two separate HTML pages for list and average rates.

### List Rates

**Minimal working request:**

```bash
curl -s 'https://www.bluestep.se/bolan/borantor/' \
  -H 'User-Agent: Mozilla/5.0'
```

**Table identifier:** Search for text "Bolån*" before the table (or HTML-encoded "Bol&aring;n*")

**Table structure:** The table has an unusual format with no `<th>` tags. Terms are in the first `<tr>` row and rates in the second row.

| Rörlig 3 månader | Fast 3 år | Fast 5 år |
|------------------|-----------|-----------|
| 4,45% | 4,60% | 4,68% |

**Data formats:**

- Term: Swedish format with descriptive prefix (e.g., "Rörlig 3 månader", "Fast 3 år", "Fast 5 år")
- Rate: Swedish decimal format with comma (4,45%)
- No change date provided on list rates page

**Terms Available:** 3 mån, 3 år, 5 år

### Average Rates

**Minimal working request:**

```bash
curl -s 'https://www.bluestep.se/bolan/borantor/genomsnittsrantor/' \
  -H 'User-Agent: Mozilla/5.0'
```

**Table identifier:** Search for text "Genomsnittsräntor" before the table

**Table structure:** Standard HTML table with `<th>` header row

| Månad | 3 mån | 1 år | 3 år | 5 år |
|-------|-------|------|------|------|
| 2025 11 | 5,68% | 6,63% | 6,83% | 6,33% |
| 2025 10 | 5,98% | 6,27% | 6,75% | 5,93% |

**Data formats:**

- Month: "YYYY MM" format with space (e.g., "2025 11" = November 2025)
- Rate: Swedish decimal format with comma (5,68%)

**Terms Available:** 3 mån, 1 år, 3 år, 5 år

**Note:** Average rates include 1 år term which is NOT available in list rates.

---

