package search

import (
	"context"
	"errors"
	"fmt"
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
}

func New(cache Cache, marketplaces ...Marketplace) *Service {
	return &Service{cache: cache, marketplaces: marketplaces}
}

func (s *Service) Search(ctx context.Context, query string) ([]product.Product, error) {
	query = normalizeQuery(query)
	if s.cache != nil {
		products, err := s.cache.Get(ctx, query)
		if err == nil {
			return products, nil
		}
	}

	products, errs := s.fetch(ctx, query)
	if len(products) == 0 {
		if len(errs) == len(s.marketplaces) {
			return nil, fmt.Errorf("search marketplaces: %w", errors.Join(errs...))
		}
		return nil, ErrProductsNotFound
	}

	sort.Slice(products, func(i, j int) bool {
		return products[i].DiscountPriceKopecks < products[j].DiscountPriceKopecks
	})

	if s.cache != nil {
		_ = s.cache.Set(ctx, query, products)
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
