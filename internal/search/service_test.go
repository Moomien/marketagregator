package search

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"agregator/internal/product"
)

type fakeMarketplace struct {
	products []product.Product
	err      error
	calls    int
}

func (m *fakeMarketplace) Search(context.Context, string) ([]product.Product, error) {
	m.calls++
	return m.products, m.err
}

type fakeCache struct {
	products []product.Product
	getErr   error
	setErr   error
	setCalls int
}

func (c *fakeCache) Get(context.Context, string) ([]product.Product, error) {
	return c.products, c.getErr
}

func (c *fakeCache) Set(_ context.Context, _ string, products []product.Product) error {
	c.products = products
	c.setCalls++
	return c.setErr
}

func TestSearchUsesCache(t *testing.T) {
	cached := []product.Product{{ProductID: "cached", DiscountPriceKopecks: 500}}
	cache := &fakeCache{products: cached}
	marketplace := &fakeMarketplace{}
	service := New(slog.Default(), cache, marketplace)

	products, err := service.Search(context.Background(), " Phone ")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(products) != 1 || products[0].ProductID != "cached" {
		t.Fatalf("Search() products = %#v", products)
	}
	if marketplace.calls != 0 {
		t.Fatalf("marketplace calls = %d, want 0", marketplace.calls)
	}
}

func TestSearchSortsAndCachesPartialResult(t *testing.T) {
	cache := &fakeCache{getErr: errors.New("cache miss")}
	first := &fakeMarketplace{products: []product.Product{{ProductID: "expensive", DiscountPriceKopecks: 2_000}}}
	second := &fakeMarketplace{products: []product.Product{{ProductID: "cheap", DiscountPriceKopecks: 1_000}}}
	failed := &fakeMarketplace{err: errors.New("unavailable")}
	service := New(slog.Default(), cache, first, second, failed)

	products, err := service.Search(context.Background(), "phone")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if products[0].ProductID != "cheap" || products[1].ProductID != "expensive" {
		t.Fatalf("Search() did not sort products: %#v", products)
	}
	if cache.setCalls != 1 {
		t.Fatalf("cache Set calls = %d, want 1", cache.setCalls)
	}
}

func TestSearchReturnsErrorWhenAllMarketplacesFail(t *testing.T) {
	service := New(slog.Default(), nil,
		&fakeMarketplace{err: errors.New("ozon unavailable")},
		&fakeMarketplace{err: errors.New("wb unavailable")},
	)

	if _, err := service.Search(context.Background(), "phone"); err == nil {
		t.Fatal("Search() error = nil, want error")
	}
}
