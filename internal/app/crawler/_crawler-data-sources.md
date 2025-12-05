# Crawler Data Sources

This document describes general information about crawler data sources and HTTP requests.

> **Last validated:** 2025-12-03

## Overview

| Bank          | Package         | Data Sources                     | Rate Types     | Auth Required           | Min Headers            |
|---------------|-----------------|----------------------------------|----------------|-------------------------|------------------------|
| SEB           | `seb`           | 2 JSON APIs                      | List + Average | Yes (API key + Referer) | X-API-Key, Referer     |
| Nordea        | `nordea`        | 2 HTML pages                     | List + Average | No                      | User-Agent             |
| ICA Banken    | `icabanken`     | 1 HTML page                      | List + Average | No                      | User-Agent, Sec-Ch-Ua* |
| Danske Bank   | `danskebank`    | 1 HTML page                      | List + Average | No                      | User-Agent             |
| Handelsbanken | `handelsbanken` | 2 JSON APIs                      | List + Average | No                      | User-Agent             |
| SBAB          | `sbab`          | 2 JSON APIs                      | List + Average | No                      | User-Agent             |
| Skandiabanken | `skandia`       | 2 HTML+JSON                      | List + Average | No                      | User-Agent             |
| Swedbank      | `swedbank`      | 2 HTML pages                     | List + Average | No                      | User-Agent             |
| Stabelo       | `stabelo`       | 1 HTML page (Remix JSON) + 1 PDF | List + Average | No                      | User-Agent             |
| Bluestep      | `bluestep`      | 2 HTML pages                     | List + Average | No                      | User-Agent             |
| Ikano Bank    | `ikanobank`     | 1 JSON API + 1 HTML page         | List + Average | No                      | User-Agent             |
| Ålandsbanken  | `alandsbanken`  | 1 HTML page                      | List + Average | No                      | User-Agent             |
| Nordnet       | `nordnet`       | 1 JSON API                       | List only      | No                      | User-Agent             |

\* ICA Banken requires matching `User-Agent` and `Sec-Ch-Ua` headers (Chrome version must match in both)

## Crawler-Specific Documentation

Each crawler has its own package with detailed documentation:

- **SEB**: See `internal/app/crawler/seb/README.md`
- **Nordea**: See `internal/app/crawler/nordea/README.md`
- **ICA Banken**: See `internal/app/crawler/icabanken/README.md`
- **Danske Bank**: See `internal/app/crawler/danskebank/README.md`
- **Handelsbanken**: See `internal/app/crawler/handelsbanken/README.md`
- **SBAB**: See `internal/app/crawler/sbab/README.md`
- **Skandiabanken**: See `internal/app/crawler/skandia/README.md`
- **Swedbank**: See `internal/app/crawler/swedbank/README.md`
- **Stabelo**: See `internal/app/crawler/stabelo/README.md`
- **Bluestep**: See `internal/app/crawler/bluestep/README.md`
- **Ikano Bank**: See `internal/app/crawler/ikanobank/README.md`
- **Ålandsbanken**: See `internal/app/crawler/alandsbanken/README.md`
- **Nordnet**: See `internal/app/crawler/nordnet/testdata/README.md`

---

## Common Headers

The Go HTTP client uses these headers (with randomization):

```
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 13_5_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.3.847 Safari/537.36
Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9,application/json
Accept-Language: en-US,en;q=0.9,de-DE;q=0.8,de;q=0.7
Connection: keep-alive
Cache-Control: no-cache
Sec-Ch-Ua: "Chromium";v="125", "Brave";v="125", "Not_A Brand";v="99"
```

**Randomized values:**

- Chrome version: 120-144
- Mac OS version: 13.x.x - 15.x.x
- Minor/patch versions for Chrome and OS

---

## Term Mappings

All banks use Swedish terms that are parsed to internal term codes:

| Swedish      | Internal Code | Duration |
|--------------|---------------|----------|
| 3 mån / 3mo  | 3months       | 3 months |
| 1 år / 1yr   | 1year         | 1 year   |
| 2 år / 2yr   | 2years        | 2 years  |
| 3 år / 3yr   | 3years        | 3 years  |
| 4 år / 4yr   | 4years        | 4 years  |
| 5 år / 5yr   | 5years        | 5 years  |
| 6 år / 6yr   | 6years        | 6 years  |
| 7 år / 7yr   | 7years        | 7 years  |
| 8 år / 8yr   | 8years        | 8 years  |
| 9 år / 9yr   | 9years        | 9 years  |
| 10 år / 10yr | 10years       | 10 years |

---

## Rate Types

### List Rates (Listräntor)

List rates are the advertised interest rates published by banks. These represent the starting point for negotiations and
change periodically (typically weekly or monthly).

### Average Rates (Snitträntor)

Average rates (also called "genomsnittsräntor") are historical monthly averages of actual rates granted to customers.
Swedish banks are required by the Financial Supervisory Authority (Finansinspektionen) to publish these rates monthly.

Average rates are always lower than list rates because:

1. Customers negotiate discounts from list rates
2. Banks offer volume and LTV-based discounts
3. Special promotions and relationship pricing

---

## Testing

Each crawler has its own test suite with golden files stored in the crawler's `testdata/` directory.

Run all crawler tests:

```bash
go test ./internal/app/crawler/...
```

Run tests for a specific crawler:

```bash
go test ./internal/app/crawler/seb/
```

### Golden Files

Golden files are real HTML/JSON/XLSX/PDF files downloaded from bank websites. They enable deterministic testing without
making live HTTP requests.

**Important:**

- Always use real data from bank websites - never create fake test data
- Golden files should be refreshed periodically to ensure crawlers handle current website structures
- See each crawler's README.md for specific refresh commands

---

## Adding a New Crawler

When adding a new bank crawler, follow these steps:

### 1. Create Package Structure

```bash
mkdir -p internal/app/crawler/{bankname}/testdata
```

### 2. Create Implementation Files

- `internal/app/crawler/{bankname}/{bankname}.go` - Crawler implementation
- `internal/app/crawler/{bankname}/{bankname}_test.go` - Tests
- `internal/app/crawler/{bankname}/testdata/` - Golden files
- `internal/app/crawler/{bankname}/README.md` - Documentation

### 3. Implementation Pattern

```go
package bankname

import (
	"github.com/yama6a/bolan-compare/internal/pkg/http"
	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

const (
	bankNameURL            = "https://..."
	bankName    model.Bank = "Bank Name"
)

type BankNameCrawler struct {
	httpClient http.Client
	logger     *zap.Logger
}

func NewBankNameCrawler(httpClient http.Client, logger *zap.Logger) *BankNameCrawler {
	return &BankNameCrawler{httpClient: httpClient, logger: logger}
}

func (c *BankNameCrawler) Crawl(channel chan<- model.InterestSet) {
	// Implementation
}

// Interface compliance check
var _ crawler.SiteCrawler = &BankNameCrawler{}
```

### 4. Register Crawler

Update `cmd/crawler/main.go`:

```go
import "github.com/yama6a/bolan-compare/internal/app/crawler/bankname"

crawlers := []crawler.SiteCrawler{
// ...
bankname.NewBankNameCrawler(httpClient, logger.Named("bankname-crawler")),
}
```

### 5. Update Documentation

- Update `crawler-plan.md` with implementation status
- Update this file's overview table
- Create detailed `README.md` in the crawler's package directory

---

## Architecture Notes

### Dependency Injection

All dependencies are instantiated in `cmd/crawler/main.go` and injected via constructors. Packages must not instantiate
their own dependencies internally.

### Error Handling

Crawlers should log errors and continue processing. Do not fail the entire crawl due to parse errors for individual
rates.

Use `c.logger.Warn()` for expected errors (e.g., parsing failures) and `c.logger.Error()` for unexpected errors (e.g.,
network failures).

### Robustness

Crawlers should be resilient to website changes:

- Never hardcode rate values or term lists
- Dynamically discover downloadable file links (XLSX, PDF)
- Use table identifiers based on surrounding text, not structure
- Handle missing/empty values gracefully

See `CLAUDE.md` for detailed coding guidelines.
