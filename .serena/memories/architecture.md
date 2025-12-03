# Bolan Crawler Architecture

## Dependency Injection Pattern

All shared dependencies are instantiated as singletons in `cmd/crawler/main.go` and injected into crawlers.

### HTTP Client
- Interface: `httpclient.Fetcher` in `internal/pkg/httpclient/client.go`
- Implementation: `httpclient.Client`
- Mock: `httpclientmock.FetcherMock` (generated via moq)

### Instantiation in main.go
```go
httpClient := httpclient.NewClient(30 * time.Second)
crawlers := []crawler.SiteCrawler{
    crawler.NewBankNameCrawler(httpClient, logger),
}
```

### Crawler Structure
Each crawler accepts dependencies via constructor:
```go
type BankCrawler struct {
    httpClient httpclient.Fetcher
    logger     *zap.Logger
}
```

## Mock Generation
Run moq to generate mocks:
```bash
moq -out internal/pkg/httpclient/httpclientmock/fetcher_mock.go -pkg httpclientmock ./internal/pkg/httpclient Fetcher
```

## Key Directories
- `internal/pkg/httpclient/` - HTTP client interface and implementation
- `internal/pkg/httpclient/httpclientmock/` - Generated mocks
- `internal/pkg/utils/` - HTML parsing utilities (ParseTable, ParseTerm, etc.)
- `internal/pkg/model/` - Data models (InterestSet, Term, Bank)
