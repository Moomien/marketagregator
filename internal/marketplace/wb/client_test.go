package wb

import "testing"

func TestParseProducts(t *testing.T) {
	body := []byte(`{"products":[{"id":123456,"name":"Phone","rating":4.8,"feedbacks":12,"sizes":[{"price":{"product":123400,"basic":150000}}]}]}`)

	products, err := parseProducts(body)
	if err != nil {
		t.Fatalf("parseProducts() error = %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("parseProducts() returned %d products, want 1", len(products))
	}
	got := products[0]
	if got.ProductID != "123456" {
		t.Fatalf("ProductID = %q", got.ProductID)
	}
	if got.DiscountPriceKopecks != 123_400 || got.BasePriceKopecks != 150_000 {
		t.Fatalf("unexpected prices: %#v", got)
	}
}

func TestParseProductsRejectsMissingPrice(t *testing.T) {
	body := []byte(`{"products":[{"id":123456,"sizes":[]}]}`)
	if _, err := parseProducts(body); err == nil {
		t.Fatal("parseProducts() error = nil, want missing price error")
	}
}
