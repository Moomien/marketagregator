package ozon

import "testing"

func TestParseProducts(t *testing.T) {
	body := []byte(`{
        "widgetStates": {
            "tileGridDesktop-1": "{\"items\":[{\"sku\":\"42\",\"action\":{\"link\":\"/product/42\"},\"tileImage\":{\"items\":[{\"image\":{\"link\":\"https://image.example/42.jpg\"}}]},\"mainState\":[{\"type\":\"textAtom\",\"textAtom\":{\"text\":\"Phone\"}},{\"type\":\"priceV2\",\"priceV2\":{\"price\":[{\"textStyle\":\"PRICE\",\"text\":\"1 234 ₽\"},{\"textStyle\":\"ORIGINAL_PRICE\",\"text\":\"1 500 ₽\"}]}}]}]}"
        }
    }`)

	products, err := parseProducts(body)
	if err != nil {
		t.Fatalf("parseProducts() error = %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("parseProducts() returned %d products, want 1", len(products))
	}
	got := products[0]
	if got.Link != "https://www.ozon.ru/product/42" {
		t.Fatalf("Link = %q", got.Link)
	}
	if got.DiscountPriceKopecks != 123_400 || got.BasePriceKopecks != 150_000 {
		t.Fatalf("unexpected prices: %#v", got)
	}
}

func TestParseProductsRejectsInvalidPrice(t *testing.T) {
	body := []byte(`{"widgetStates":{"tileGridDesktop-1":"{\"items\":[{\"sku\":\"42\",\"mainState\":[{\"type\":\"priceV2\",\"priceV2\":{\"price\":[{\"textStyle\":\"PRICE\",\"text\":\"not available\"}]}}]}]}"}}`)
	if _, err := parseProducts(body); err == nil {
		t.Fatal("parseProducts() error = nil, want invalid price error")
	}
}
