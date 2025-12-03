package main

import (
	"fmt"
	gohttp "net/http"
	"time"

	"github.com/yama6a/bolan-compare/internal/app/crawler"
	"github.com/yama6a/bolan-compare/internal/pkg/http"
	"github.com/yama6a/bolan-compare/internal/pkg/store"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	loggerConfig.DisableStacktrace = true
	logger, err := loggerConfig.Build()
	noErr(err)

	// Initialize shared HTTP client singleton.
	httpTimeout := 30 * time.Second
	baseHTTPClient := &gohttp.Client{
		Timeout: httpTimeout,
		CheckRedirect: func(_ *gohttp.Request, _ []*gohttp.Request) error {
			return gohttp.ErrUseLastResponse
		},
	}
	httpClient := http.NewClient(baseHTTPClient, httpTimeout)

	crawlers := []crawler.SiteCrawler{
		crawler.NewDanskeBankCrawler(httpClient, logger.Named("danske-bank-crawler")),
		crawler.NewSebBankCrawler(httpClient, logger.Named("seb-crawler")),
		crawler.NewICABankenCrawler(httpClient, logger.Named("ica-banken-crawler")),
		crawler.NewNordeaCrawler(httpClient, logger.Named("nordea-crawler")),
		crawler.NewHandelsbankenCrawler(httpClient, logger.Named("handelsbanken-crawler")),
		crawler.NewSBABCrawler(httpClient, logger.Named("sbab-crawler")),
		crawler.NewSwedbankCrawler(httpClient, logger.Named("swedbank-crawler")),
	}

	pgStore := store.NewMemoryStore(nil, logger.Named("Store"))
	svc := crawler.NewService(pgStore, crawlers, logger.Named("Crawler Svc"))

	svc.Crawl()
}

func noErr(err error) {
	if err != nil {
		fmt.Printf("failed to initialize something important: %v\n", err)
		panic(err)
	}
}
