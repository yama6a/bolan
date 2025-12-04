package main

import (
	"fmt"
	gohttp "net/http"
	"time"

	"github.com/yama6a/bolan-compare/internal/app/crawler"
	"github.com/yama6a/bolan-compare/internal/app/crawler/alandsbanken"
	"github.com/yama6a/bolan-compare/internal/app/crawler/bluestep"
	"github.com/yama6a/bolan-compare/internal/app/crawler/danskebank"
	"github.com/yama6a/bolan-compare/internal/app/crawler/handelsbanken"
	"github.com/yama6a/bolan-compare/internal/app/crawler/icabanken"
	"github.com/yama6a/bolan-compare/internal/app/crawler/ikanobank"
	"github.com/yama6a/bolan-compare/internal/app/crawler/nordea"
	"github.com/yama6a/bolan-compare/internal/app/crawler/sbab"
	"github.com/yama6a/bolan-compare/internal/app/crawler/seb"
	"github.com/yama6a/bolan-compare/internal/app/crawler/skandia"
	"github.com/yama6a/bolan-compare/internal/app/crawler/stabelo"
	"github.com/yama6a/bolan-compare/internal/app/crawler/swedbank"
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
		danskebank.NewDanskeBankCrawler(httpClient, logger.Named("danske-bank-crawler")),
		seb.NewSebBankCrawler(httpClient, logger.Named("seb-crawler")),
		icabanken.NewICABankenCrawler(httpClient, logger.Named("ica-banken-crawler")),
		nordea.NewNordeaCrawler(httpClient, logger.Named("nordea-crawler")),
		handelsbanken.NewHandelsbankenCrawler(httpClient, logger.Named("handelsbanken-crawler")),
		sbab.NewSBABCrawler(httpClient, logger.Named("sbab-crawler")),
		skandia.NewSkandiaCrawler(httpClient, logger.Named("skandia-crawler")),
		swedbank.NewSwedbankCrawler(httpClient, logger.Named("swedbank-crawler")),
		stabelo.NewStabeloCrawler(httpClient, logger.Named("stabelo-crawler")),
		bluestep.NewBluestepCrawler(httpClient, logger.Named("bluestep-crawler")),
		ikanobank.NewIkanoBankCrawler(httpClient, logger.Named("ikano-bank-crawler")),
		alandsbanken.NewAlandsbankCrawler(httpClient, logger.Named("alandsbanken-crawler")),
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
