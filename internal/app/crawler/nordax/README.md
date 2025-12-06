## Nordax Bank

Nordax Bank is a specialty/non-prime lender (NOBA Bank Group) that only publishes average rates (snitträntor). No list rates are publicly available.

### Average Rates

**Minimal working request:**

```bash
curl -s 'https://www.nordax.se/lana/bolan/genomsnittsrantor' \
  -H 'User-Agent: Mozilla/5.0'
```

**Data format:** Next.js server-rendered page with `__NEXT_DATA__` JSON embedded in HTML

**JSON extraction:** The page contains a `<script id="__NEXT_DATA__" type="application/json">` tag with the complete page data structure.

**JSON path to table data:**
```
props.pageProps.page.content[0].expandableContent[0].body[0]
```

**Table structure:** JSON table with header row and data rows

Header row:
```json
["Datum", "3 månaders", "36 månaders", "60 månaders"]
```

Data rows (example):
```json
["2025-11", "4,66%", "4,81%", "4,72%"]
```

**Data formats:**

- Month: "YYYY-MM" format (e.g., "2025-11" = November 2025)
- Rate: Swedish decimal format with comma and percent sign (e.g., "4,66%")
- Term: Swedish format with number of months (e.g., "3 månaders", "36 månaders", "60 månaders")

**Terms Available:** 3 månaders, 36 månaders, 60 månaders

**Special Notes:**

- Only average rates available (no list rates published)
- Data is embedded in Next.js JSON, not visible in DOM
- The table is located within an expandable content section
- Empty cells contain empty string `""` - these are skipped

**Table Type Validation:**

The crawler verifies that `body[0]._type === "table"` before parsing.

---
