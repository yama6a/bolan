# Bolan Crawler Architecture

## Project Structure (Post-Refactoring)

Each bank crawler now has its own package:
```
internal/app/crawler/               # Base package
  ├── service.go                    # SiteCrawler interface
  ├── testing_helpers.go            # Shared test utilities
  ├── _crawler-data-sources.md      # General documentation
  ├── alandsbanken/
  │   ├── alandsbanken.go
  │   ├── alandsbanken_test.go
  │   └── testdata/
  │       ├── README.md             # Bank-specific docs
  │       ├── alandsbanken.html
  │       └── ...
  ├── bluestep/
  ├── danskebank/
  ├── handelsbanken/
  ├── icabanken/
  ├── ikanobank/
  ├── nordea/
  ├── sbab/
  ├── seb/
  ├── skandia/
  ├── stabelo/
  └── swedbank/
```

## Dependency Injection Pattern

All shared dependencies are instantiated as singletons in `cmd/crawler/main.go` and injected into crawlers.

### HTTP Client
- Interface: `http.Client` in `internal/pkg/http/client.go`
- Implementation: `http.Client`
- Mock: `httpmock.ClientMock` (generated via moq)

### Instantiation in main.go
```go
httpClient := http.NewClient(30 * time.Second)
crawlers := []crawler.SiteCrawler{
    seb.NewSebBankCrawler(httpClient, logger.Named("seb-crawler")),
    nordea.NewNordeaCrawler(httpClient, logger.Named("nordea-crawler")),
    // ... etc for each bank
}
```

### Crawler Structure
Each crawler:
- Lives in its own package (e.g., `package seb`)
- Imports base crawler package: `import "github.com/yama6a/bolan-compare/internal/app/crawler"`
- Implements `crawler.SiteCrawler` interface
- Accepts dependencies via constructor

Example:
```go
package seb

import "github.com/yama6a/bolan-compare/internal/app/crawler"

var _ crawler.SiteCrawler = &SebBankCrawler{}

type SebBankCrawler struct {
    httpClient http.Client
    logger     *zap.Logger
}

func NewSebBankCrawler(httpClient http.Client, logger *zap.Logger) *SebBankCrawler {
    return &SebBankCrawler{httpClient: httpClient, logger: logger}
}
```

## Testing Structure

### Shared Test Helpers
Located in `internal/app/crawler/testing_helpers.go`:
- `LoadGoldenFile(t, filename)` - Loads test data files
- `CountRatesByType(results)` - Counts list vs average rates
- `AssertBankName(t, results, wantBank)` - Validates bank names
- `TestInvalidJSON(t, parseFunc)` - Tests JSON parsing errors

### Test Pattern
Each test file:
- Uses import alias: `import crawlertest "github.com/yama6a/bolan-compare/internal/app/crawler"`
- Loads golden files from local `testdata/` directory
- Uses `crawlertest.LoadGoldenFile(t, "testdata/filename.json")`
- Mocks HTTP with `httpmock.ClientMock`

Example:
```go
package seb

import crawlertest "github.com/yama6a/bolan-compare/internal/app/crawler"

func TestSebBankCrawler_Crawl(t *testing.T) {
    listRatesJSON := crawlertest.LoadGoldenFile(t, "testdata/seb_list_rates.json")
    // ...
}
```

## Key Directories
- `internal/app/crawler/` - Base package with interface and shared test helpers
- `internal/app/crawler/{bank}/` - Bank-specific crawler packages
- `internal/app/crawler/{bank}/testdata/` - Bank-specific test data and documentation
- `internal/pkg/http/` - HTTP client interface and implementation
- `internal/pkg/http/httpmock/` - Generated mocks
- `internal/pkg/utils/` - HTML parsing utilities (ParseTable, ParseTerm, etc.)
- `internal/pkg/model/` - Data models (InterestSet, Term, Bank)

## Adding a New Crawler

1. Create package directory: `internal/app/crawler/newbank/`
2. Create files:
   - `newbank.go` - Implementation
   - `newbank_test.go` - Tests
   - `testdata/README.md` - Bank-specific documentation
   - `testdata/newbank_list_rates.*` - Test data
   - `testdata/newbank_avg_rates.*` - Test data
3. Import in `cmd/crawler/main.go`:
   ```go
   import "github.com/yama6a/bolan-compare/internal/app/crawler/newbank"
   ```
4. Register in crawlers slice:
   ```go
   crawlers := []crawler.SiteCrawler{
       // ...
       newbank.NewNewBankCrawler(httpClient, logger.Named("newbank-crawler")),
   }
   ```
