package search

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"

	"agregator/internal/product"
)

var ErrProductsNotFound = errors.New("products not found")

type Marketplace interface {
	Search(ctx context.Context, query string) ([]product.Product, error)
}

type Cache interface {
	Get(ctx context.Context, query string) ([]product.Product, error)
	Set(ctx context.Context, query string, products []product.Product) error
}

type Service struct {
	cache        Cache
	marketplaces []Marketplace
	logger       *slog.Logger
}

func New(logger *slog.Logger, cache Cache, marketplaces ...Marketplace) *Service {
	return &Service{logger: logger, cache: cache, marketplaces: marketplaces}
}

func (s *Service) Search(ctx context.Context, query string) ([]product.Product, error) {
	query = normalizeQuery(query)
	if s.cache != nil {
		products, err := s.cache.Get(ctx, query)
		if err == nil {
			s.logger.Debug("cache hit", "query", query, "products", len(products))
			return products, nil
		}
		s.logger.Debug("cache unavailable", "query", query, "error", err)
	}

	products, errs := s.fetch(ctx, query)
	if len(products) == 0 {
		if len(errs) == len(s.marketplaces) {
			return nil, fmt.Errorf("search marketplaces: %w", errors.Join(errs...))
		}
		return nil, ErrProductsNotFound
	}
	for _, err := range errs {
		s.logger.Warn("marketplace search failed", "query", query, "error", err)
	}

	sort.Slice(products, func(i, j int) bool {
		return products[i].DiscountPriceKopecks < products[j].DiscountPriceKopecks
	})

	if s.cache != nil {
		if err := s.cache.Set(ctx, query, products); err != nil {
			s.logger.Warn("save search result to cache", "query", query, "error", err)
		}
	}
	return products, nil
}

func (s *Service) fetch(ctx context.Context, query string) ([]product.Product, []error) {
	type result struct {
		products []product.Product
		err      error
	}

	results := make(chan result, len(s.marketplaces))
	var wg sync.WaitGroup
	for _, marketplace := range s.marketplaces {
		wg.Add(1)
		go func(m Marketplace) {
			defer wg.Done()
			products, err := m.Search(ctx, query)
			results <- result{products: products, err: err}
		}(marketplace)
	}
	wg.Wait()
	close(results)

	var products []product.Product
	var errs []error
	for result := range results {
		products = append(products, result.products...)
		if result.err != nil {
			errs = append(errs, result.err)
		}
	}
	return products, errs
}

func normalizeQuery(query string) string {
	return strings.ToLower(strings.Join(strings.Fields(query), " "))
}
