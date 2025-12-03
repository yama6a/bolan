# CLAUDE.md - Bolan Crawler Project

## Tech Stack
- **Language**: Go 1.21+
- **HTTP**: Standard `net/http` (no external HTTP clients)
- **HTML Parsing**: `golang.org/x/net/html` tokenizer
- **Logging**: `go.uber.org/zap` (structured logging)
- **Linting**: golangci-lint with strict config (`.golangci.yaml`)

## Project Structure
```
cmd/crawler/                        # Main entrypoint (instantiates singletons)
internal/app/crawler/               # Crawler implementations (one file per bank)
internal/pkg/http/                  # HTTP client interface + implementation
internal/pkg/http/httpmock/         # Generated mocks (via moq)
internal/pkg/model/                 # Data models (InterestSet, Term, Bank, Type)
internal/pkg/utils/                 # HTML parsing, string normalization
internal/pkg/store/                 # Data storage interface
```

## Commands
```bash
make build    # Build binaries
make test     # Run tests
make lint     # Run golangci-lint
make fumpt    # Format with gofumpt
make ci       # Run all checks (lint, vet, test, vuln)
```

## Workflow

**IMPORTANT: Always run `make ci` after each work step and fix all issues before proceeding.**

This ensures:
- Code compiles correctly
- All tests pass
- Linting rules are satisfied
- No security vulnerabilities

If `make ci` fails, fix all reported issues before continuing to the next task.

## Adding a New Bank Crawler

### 1. Create crawler file
Create `internal/app/crawler/{bank_name}.go`. Follow existing pattern:
```go
package crawler

type BankNameCrawler struct {
    httpClient http.Client  // Dependency injected
    logger     *zap.Logger
}

var _ SiteCrawler = &BankNameCrawler{}  // Interface compliance check

func NewBankNameCrawler(httpClient http.Client, logger *zap.Logger) *BankNameCrawler {
    return &BankNameCrawler{httpClient: httpClient, logger: logger}
}

func (c *BankNameCrawler) Crawl(channel chan<- model.InterestSet) {
    // Fetch HTML/JSON using injected client, parse, send results to channel
    html, err := c.httpClient.Fetch(url, nil)
}
```

### 2. Register in main.go
Add to `cmd/crawler/main.go`:
```go
// httpClient is already instantiated as singleton in main()
crawlers := []crawler.SiteCrawler{
    crawler.NewBankNameCrawler(httpClient, logger.Named("bank-name-crawler")),
}
```

### 3. Prefer HTTP over Playwright
Always try basic `net/http` first via the injected `http.Client`:
```go
html, err := c.httpClient.Fetch(url, nil)
```
Only use Playwright when planning/exploring the banks' websites. If you cannot fetch the required data with HTTP, document the reason in `crawler-plan.md`.

## HTML Table Parsing Pattern
Use `utils.FindTokenizedTableByTextBeforeTable()` to locate tables:
```go
tokenizer, err := utils.FindTokenizedTableByTextBeforeTable(rawHTML, "Aktuella bolåneräntor")
if err != nil {
    return nil, fmt.Errorf("failed to find table: %w", err)
}

table, err := utils.ParseTable(tokenizer)
// table.Header = []string{"Bindningstid", "Ränta", ...}
// table.Rows = [][]string{{"3 mån", "3,45 %", ...}, ...}
```

## JSON API Pattern
For banks with JSON APIs:
```go
type RatesResponse struct {
    Rates []struct {
        Period string  `json:"period"`
        Rate   float64 `json:"rate"`
    } `json:"rates"`
}

body, err := c.httpClient.Fetch(apiURL, nil)
var resp RatesResponse
if err := json.Unmarshal([]byte(body), &resp); err != nil { ... }
```

## Dependency Injection

### Singletons
All shared dependencies (including nested dependencies) are instantiated in `cmd/crawler/main.go`:
```go
import (
    gohttp "net/http"
    "github.com/yama6a/bolan-compare/internal/pkg/http"
)

httpTimeout := 30 * time.Second
baseHTTPClient := &gohttp.Client{
    Timeout: httpTimeout,
    CheckRedirect: func(_ *gohttp.Request, _ []*gohttp.Request) error {
        return gohttp.ErrUseLastResponse
    },
}
httpClient := http.NewClient(baseHTTPClient, httpTimeout)
```
**Rule**: Packages must not instantiate their own dependencies internally. All `New*()` constructors receive dependencies as parameters.

### Interfaces with Mocks
Dependencies expose interfaces for testability. Generate mocks with:
```bash
go generate ./internal/pkg/http/...
# Or manually: moq -out internal/pkg/http/httpmock/client_mock.go -pkg httpmock . Client
```

Use mocks in tests:
```go
mockClient := &httpmock.ClientMock{
    FetchFunc: func(url string, headers map[string]string) (string, error) {
        return `<html>...</html>`, nil
    },
}
crawler := NewBankNameCrawler(mockClient, logger)
```

## Fixing Broken Crawlers

### Diagnosis Steps
1. Run crawler, check error logs
2. Fetch URL manually: `curl -v "https://bank.example.com/rates"`
3. If 404/changed: Use browser DevTools to find new URL
4. If empty response: Check if JS-rendered (view-source vs rendered)
5. If blocked: Check User-Agent requirements

### Common Fixes
- **URL changed**: Update const, verify with curl
- **Table moved**: Update search string in `FindTokenizedTableByTextBeforeTable()`
- **Date format changed**: Update regex pattern
- **API endpoint changed**: Use DevTools Network tab to find new endpoint

### Example: SEB URL change
```go
// OLD: sebAvgURL = "https://seb.se/bolan/snittrantor"
// NEW: sebAvgURL = "https://seb.se/privat/lan/bolan/snittrantor"
const sebAvgURL = "https://seb.se/privat/lan/bolan/snittrantor"
```

## Data Model

### InterestSet Fields
```go
model.InterestSet{
    Bank:          "Nordea",              // Bank name constant
    Type:          model.TypeListRate,    // TypeListRate or TypeAverageRate
    Term:          model.Term3months,     // Term3months, Term1year, ..., Term10years
    NominalRate:   3.45,                  // Rate as float32 (3.45 = 3.45%)
    ChangedOn:     &time.Time{},          // When rate changed (list rates only)
    LastCrawledAt: time.Now().UTC(),      // Always set to crawl time
    AverageReferenceMonth: &model.AvgMonth{Year: 2025, Month: 11}, // Avg rates only
}
```

### Term Parsing
Use `utils.ParseTerm()` - handles Swedish formats:
```go
term, err := utils.ParseTerm("3 mån")  // Returns model.Term3months
term, err := utils.ParseTerm("1 år")   // Returns model.Term1year
```

## Code Style

### Naming
- Crawler type: `{BankName}Crawler` (e.g., `NordeaCrawler`)
- Constructor: `New{BankName}Crawler`
- URL const: `{bank}URL` or `{bank}ListRatesURL`/`{bank}AvgRatesURL`
- Bank const: `{bank}BankName` of type `model.Bank`

### Interface Compliance
Always add compile-time interface checks for types that implement interfaces:
```go
var _ SiteCrawler = &NordeaCrawler{}  // Ensures NordeaCrawler implements SiteCrawler
var _ Client = &client{}              // Ensures client implements Client
```
This catches interface mismatches at compile time rather than runtime.

### Interface Location
Define interfaces next to their implementations, not where they are injected. For example:
- `Store` interface lives in `internal/pkg/store/` alongside `MemoryStore`
- `Client` interface lives in `internal/pkg/http/` alongside `client`
- `SiteCrawler` interface lives in `internal/app/crawler/` alongside crawler implementations

This follows the Go proverb "accept interfaces, return structs" and keeps related code together.

### Error Handling
Log and continue on parse errors (don't fail entire crawl):
```go
rate, err := parseRate(row[1])
if err != nil {
    c.logger.Warn("failed to parse rate", zap.String("rate", row[1]), zap.Error(err))
    continue  // Skip this row, continue with others
}
```

### Formatting
Run `make fumpt` before committing. The linter is strict - don't add `//nolint` without good reason.

## Don'ts
- Don't use external HTTP client libraries (use standard `net/http`)
- Don't panic on parse errors (log and continue)
- Don't hardcode rate values (always fetch from source)
- Don't skip `make lint` - CI will fail
- Don't commit without running `make ci`

## Research Resources
- See `crawler-plan.md` for implementation details per bank
- Konsumenternas.se lists all 21 Swedish mortgage banks
- Banks must publish snitträntor (average rates) by Swedish FSA requirement
