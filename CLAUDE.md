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

When adding a new bank crawler, the following files must be created or modified:

### Files to Create
1. `internal/app/crawler/{bank_name}.go` - Crawler implementation
2. `internal/app/crawler/{bank_name}_test.go` - Tests
3. `internal/app/crawler/testdata/{bank_name}.html` (or `.json`) - Golden file(s)

### Files to Modify
4. `cmd/crawler/main.go` - Register the new crawler
5. `crawler-plan.md` - Update bank status to "Done"
6. `internal/app/crawler/_crawler-data-sources.md` - Document URLs, headers, and data formats

### Implementation Pattern
- Inject `http.Client` and `*zap.Logger` via constructor
- Add interface compliance check: `var _ SiteCrawler = &BankNameCrawler{}`
- Prefer HTTP over Playwright - only use Playwright for exploration
- If HTTP doesn't work, document the reason in `crawler-plan.md`

### Testing Pattern
- Use golden files (real HTML/JSON from bank websites) for deterministic tests
- Mock HTTP client using `httpmock.ClientMock`
- Test extraction methods, parsing functions, and edge cases

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
