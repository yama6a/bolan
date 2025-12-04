## Skandiabanken

Skandiabanken embeds rate data as JSON within JavaScript code in their HTML pages. The data is stored in a `SKB.pageContent` variable that can be extracted using regex.

### List Rates

**Minimal working request:**

```bash
curl -s 'https://www.skandia.se/lana/bolan/bolanerantor/' \
  -H 'User-Agent: Mozilla/5.0'
```

**Data format:** JSON embedded in HTML within `SKB.pageContent` JavaScript variable

**JSON structure:**

The page content contains nested structures:
- `sectionContent2[]` - Array of sections
- Each section has a `contentLink.expanded` object
- Find the TableBlock with `header.headerText` containing "Listräntor"
- The `columns[]` array contains:
  - Column with `cellHeader` "Bindningstid" - term cells
  - Column with `cellHeader` "Listränta" - rate cells
  - Column with `cellHeader` "Senast ändrad" - change date cells

**Cell format:** HTML with entities (e.g., `<p>3 m&aring;n</p>`, `<p>3,45 %</p>`)

**Terms Available:** 3 mån, 1 år, 2 år, 3 år, 5 år

### Average Rates

**Minimal working request:**

```bash
curl -s 'https://www.skandia.se/lana/bolan/bolanerantor/snittrantor/' \
  -H 'User-Agent: Mozilla/5.0'
```

**Data format:** JSON embedded in HTML within `SKB.pageContent` JavaScript variable

**JSON structure:**

Two places to find snitträntor:
1. **TableBlock** in `sectionContent1` or `sectionContent2` with `header.headerText` like "Snitträntor november 2025"
2. **AccordionBlock** containing historical months - each item is a TableBlock with similar structure

**Table structure:** Same as list rates - columns with Bindningstid and Snittränta

**Month parsing:** Extract from header text (e.g., "Snitträntor november 2025" → November 2025)

**Note:** Cell content requires HTML entity decoding (`&aring;` → `å`)

---

