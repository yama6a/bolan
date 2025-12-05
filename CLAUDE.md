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
internal/app/crawler/               # Crawler base package (SiteCrawler interface, testing helpers)
internal/app/crawler/{bank}/        # Each bank has its own package
  ├── {bank}.go                     # Crawler implementation
  ├── {bank}_test.go                # Tests
  ├── testdata/                     # Golden files (HTML/JSON/XLSX)
  │   ├── {bank}_list_rates.*       # List rate test data
  │   ├── {bank}_avg_rates.*        # Average rate test data
  │   └── README.md                 # Bank-specific documentation
internal/pkg/http/                  # HTTP client interface + implementation
internal/pkg/http/httpmock/         # Generated mocks (via moq)
internal/pkg/model/                 # Data models (InterestSet, Term, Bank, Type)
internal/pkg/utils/                 # HTML parsing, string normalization
internal/pkg/store/                 # Data storage interface
```

### Current Bank Crawlers

Each bank crawler is in its own package:

- `alandsbanken/` - Ålandsbanken
- `bluestep/` - Bluestep
- `danskebank/` - Danske Bank
- `handelsbanken/` - Handelsbanken
- `icabanken/` - ICA Banken
- `ikanobank/` - Ikano Bank
- `nordea/` - Nordea
- `sbab/` - SBAB
- `seb/` - SEB
- `skandia/` - Skandia
- `stabelo/` - Stabelo
- `swedbank/` - Swedbank

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

When adding a new bank crawler, the following files must be created or modified:

### Files to Create

1. `internal/app/crawler/{bank_name}/{bank_name}.go` - Crawler implementation
2. `internal/app/crawler/{bank_name}/{bank_name}_test.go` - Tests
3. `internal/app/crawler/{bank_name}/testdata/` - Directory for golden files
    - `{bank_name}_list_rates.*` - List rate test data (HTML/JSON/XLSX)
    - `{bank_name}_avg_rates.*` - Average rate test data (HTML/JSON/XLSX)
    - `README.md` - Bank-specific documentation (URLs, headers, data formats)

### Files to Modify

4. `cmd/crawler/main.go` - Import and register the new crawler package
5. `crawler-plan.md` - Update bank status to "Done"

### Implementation Pattern

- Create package: `package {bankname}` (lowercase, no underscores)
- Import crawler package: `import "github.com/yama6a/bolan-compare/internal/app/crawler"`
- Inject `http.Client` and `*zap.Logger` via constructor
- Add interface compliance check: `var _ crawler.SiteCrawler = &BankNameCrawler{}`
- Use `//nolint:revive // Bank name prefix is intentional for clarity` if type name "stutters"
- Prefer HTTP over Playwright - only use Playwright for exploration
- If HTTP doesn't work, document the reason in `crawler-plan.md`

### Testing Pattern

- Use golden files (real HTML/JSON/XLSX from bank websites) for deterministic tests
- **NEVER create fake test data** - only use actual data downloaded from the bank's website
- Import test helpers: `import crawlertest "github.com/yama6a/bolan-compare/internal/app/crawler"`
- Load golden files: `crawlertest.LoadGoldenFile(t, "testdata/filename.json")`
- Mock HTTP client using `httpmock.ClientMock`
- Test extraction methods, parsing functions, and edge cases
- Use shared assertions: `crawlertest.AssertBankName()`, `crawlertest.CountRatesByType()`

### Example Structure

```
internal/app/crawler/examplebank/
├── examplebank.go           # Implementation
├── examplebank_test.go      # Tests
└── testdata/
    ├── README.md            # Bank-specific docs
    ├── examplebank_list_rates.json
    └── examplebank_avg_rates.html
```

## HTML Table Parsing

### Locating Tables

Three utility functions for locating tables in HTML:

- `utils.FindTokenizedTableByTextBeforeTable(html, text)` - Find first table after text appears
- `utils.FindTokenizedNthTableByTextBeforeTable(html, text, skip)` - Find Nth table after text (0-indexed)
- `utils.FindTokenizedTableByTextInCaption(html, text)` - Find table by `<caption>` element content

### Parsing Tables

Use `utils.ParseTable(tokenizer)` after locating a table. Returns a `utils.Table` struct:

- `Table.Header` - First row as `[]string`
- `Table.Rows` - Remaining rows as `[][]string`

### Term and Rate Parsing

- `utils.ParseTerm(str)` - Parses Swedish terms like "3 mån", "3 månader", "1 år" to `model.Term`
- Rate parsing is bank-specific (handle Swedish decimal comma, percent signs, dashes for empty values)

## JSON API Pattern

For banks with JSON APIs, use `c.httpClient.Fetch()` and `json.Unmarshal()`.

## Dependency Injection

All dependencies are instantiated in `cmd/crawler/main.go` and injected via constructors.
Packages must not instantiate their own dependencies internally.

## Code Style

### Naming Conventions

- Crawler type: `{BankName}Crawler`
- Constructor: `New{BankName}Crawler`
- URL const: `{bank}ListRatesURL` / `{bank}AvgRatesURL`
- Bank const: `{bank}BankName` of type `model.Bank`

### Error Handling

Log and continue on parse errors (don't fail entire crawl). Use `c.logger.Warn()` and `continue`.

### Interface Compliance

Always add compile-time interface checks: `var _ SiteCrawler = &BankNameCrawler{}`

## Robustness Requirements

### Never Hardcode Terms

- **Always read terms from source** using `utils.ParseTerm()` - never hardcode term lists
- Terms should be extracted dynamically from table headers, XLSX headers, or JSON keys
- This makes crawlers resilient to banks adding/removing terms

### Dynamic Link Discovery

- When a page links to downloadable files (XLSX, PDF), search for the link dynamically
- Don't hardcode filenames - they may change (e.g., `rates-2506.xlsx` → `rates-2507.xlsx`)
- Example: Use regex like `href="([^"]*\.xlsx)"` to find any XLSX link on a page
- Verify there's only one link of that type, or handle multiple appropriately

### Dynamic Header/Sheet Discovery

- For XLSX files, don't hardcode sheet names or header row numbers
- Search for sheets by keywords (e.g., "ränteändring", "historisk")
- Search for header rows by identifying marker text (e.g., "Datum", "Bindningstid")

## Don'ts

- Don't use external HTTP client libraries (use standard `net/http`)
- Don't panic on parse errors (log and continue)
- Don't hardcode rate values (always fetch from source)
- Don't hardcode terms - always read from source
- Don't hardcode filenames for downloadable files - discover them dynamically
- Don't skip `make lint` - CI will fail
- Don't commit without running `make ci`
- Don't create fake test data - only use real golden files from bank websites

## Research Resources

- See `crawler-plan.md` for implementation details per bank
- Konsumenternas.se lists all 21 Swedish mortgage banks
- Banks must publish snitträntor (average rates) by Swedish FSA requirement
