## ICA Banken

ICA Banken uses a single HTML page containing both list rates and average rates.

### Fetch Page

ICA Banken has bot protection that checks for matching `User-Agent` and `Sec-Ch-Ua` headers. The Chrome version in both
headers must match.

**Required headers:**

- `User-Agent` - Chrome browser user agent with version number
- `Sec-Ch-Ua` - Client hints header with **matching** Chrome version
- `Accept`, `Accept-Language`, `Connection`, `Cache-Control` - Standard browser headers

```bash
curl -s 'https://www.icabanken.se/lana/bolan/bolanerantor/' \
  -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 13_5_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.847 Safari/537.36' \
  -H 'Sec-Ch-Ua: "Chromium";v="125", "Brave";v="125", "Not_A Brand";v="99"' \
  -H 'Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9,application/json' \
  -H 'Accept-Language: en-US,en;q=0.9,de-DE;q=0.8,de;q=0.7' \
  -H 'Connection: keep-alive' \
  -H 'Cache-Control: no-cache'
```

**Key:** The `Sec-Ch-Ua` version (125) must match the Chrome version in `User-Agent` (Chrome/125.x.x.x).

### List Rates Table

**Table identifier:** Search for text "Aktuella bolåneräntor" before the table

**Table structure:**

| Bindningstid | Ränta  | Senast ändrad |
|--------------|--------|---------------|
| 3 mån        | 3,33 % | 2025-10-06    |
| 1 år         | 3,44 % | 2025-07-10    |

**Data formats:**

- Term: Swedish format (3 mån, 1 år, 2 år, 3 år, 4 år, 5 år, 7 år, 10 år)
- Rate: Swedish decimal format with comma (3,33 %)
- Date: YYYY-MM-DD

### Average Rates Table

**Table identifier:** Search for text "Snitträntor för bolån" before the table

**Table structure:**

| Månad   | 3 mån | 1 år | 2 år | 3 år | 4 år | 5 år | 7 år | 10 år |
|---------|-------|------|------|------|------|------|------|-------|
| 2025 11 | 2,65  | 2,84 | 2,84 | 2,90 | 2,99 | 3,07 | 3,34 | 3,30  |
| 2025 10 | 2,68  | 2,92 | 2,93 | 3,00 | 3,10 | 3,19 | -    | 3,57  |

**Data formats:**

- Month: "YYYY MM" format with space (e.g., "2025 11" = November 2025)
- Rate: Swedish decimal format with comma
- Missing data indicated by "-" or "-*"

---

