package main

import (
	"github.com/yama6a/bolan-compare/internal/app/crawler"
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

	crawlers := []crawler.SiteCrawler{
		// crawler.NewDummyCrawler(logger.Named("DummyCrawler")),
		crawler.NewDanskeBankCrawler(logger.Named("danske-bank-crawler")),
		crawler.NewSebBankCrawler(logger.Named("seb-crawler")),
		crawler.NewICABankenCrawler(logger.Named("ica-banken-crawler")),
	}

	pgStore := store.NewMemoryStore(nil, logger.Named("Store"))
	svc := crawler.NewService(pgStore, crawlers, logger.Named("Crawler Svc"))

	svc.Crawl()
}

func noErr(err error) {
	if err != nil {
		panic("failed to initialize something important: " + err.Error())
	}
}
