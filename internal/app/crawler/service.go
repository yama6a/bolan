package crawler

import (
	"sync"

	"github.com/yama6a/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
)

type Store interface {
	UpsertInterestSet(set model.InterestSet) error
	GetInterestSets() ([]model.InterestSet, error)
}

type SiteCrawler interface {
	Crawl(result chan<- model.InterestSet)
}

type Service struct {
	store    Store
	crawlers []SiteCrawler
	logger   *zap.Logger
}

func NewService(store Store, crawlers []SiteCrawler, logger *zap.Logger) *Service {
	return &Service{
		store:    store,
		crawlers: crawlers,
		logger:   logger,
	}
}

func (s *Service) Crawl() {
	var wg sync.WaitGroup
	objChan := make(chan model.InterestSet)

	for _, c := range s.crawlers {
		wg.Add(1)
		go func(c SiteCrawler) {
			defer wg.Done()
			c.Crawl(objChan)
		}(c)
	}

	go s.recv(objChan)

	wg.Wait()
	s.logger.Info("all crawlers finished, closing channels")

	interestSets, err := s.store.GetInterestSets()
	if err != nil {
		s.logger.Error("failed to get interestSets", zap.Error(err))
		return
	}

	// Build summary by bank and type.
	summary := make(map[model.Bank]map[model.Type]uint)
	for _, is := range interestSets {
		if _, ok := summary[is.Bank]; !ok {
			summary[is.Bank] = make(map[model.Type]uint)
		}
		summary[is.Bank][is.Type]++
	}

	// Log summary per bank.
	for bank, types := range summary {
		s.logger.Info("crawl results",
			zap.String("bank", string(bank)),
			zap.Uint("listRates", types[model.TypeListRate]),
			zap.Uint("avgRates", types[model.TypeAverageRate]),
		)
	}

	close(objChan)
}

func (s *Service) recv(c <-chan model.InterestSet) {
	s.logger.Info("starting crawler receiver")

	for set := range c {
		if err := s.store.UpsertInterestSet(set); err != nil {
			s.logger.Error("failed to upsert interestSet", zap.Any("interestSet", set), zap.Error(err))
		}
	}
}
