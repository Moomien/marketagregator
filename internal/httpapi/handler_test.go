package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agregator/internal/product"
	"agregator/internal/search"
)

type fakeMarketplace struct {
	products []product.Product
	err      error
}

func (m fakeMarketplace) Search(_ context.Context, _ string) ([]product.Product, error) {
	return m.products, m.err
}

func newHandler(marketplace fakeMarketplace) *Handler {
	service := search.New(slog.Default(), nil, marketplace)
	return New(slog.Default(), service, time.Second)
}

func TestSearchRequiresQuery(t *testing.T) {
	recorder := httptest.NewRecorder()
	newHandler(fakeMarketplace{}).Search(recorder, httptest.NewRequest(http.MethodGet, "/search", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestSearchReturnsProducts(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := newHandler(fakeMarketplace{products: []product.Product{{ProductID: "42", DiscountPriceKopecks: 1_000}}})
	handler.Search(recorder, httptest.NewRequest(http.MethodGet, "/search?query=phone", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q", contentType)
	}
}

func TestSearchHidesInternalError(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := newHandler(fakeMarketplace{err: errors.New("upstream secret")})
	handler.Search(recorder, httptest.NewRequest(http.MethodGet, "/search?query=phone", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if recorder.Body.String() == "upstream secret\n" {
		t.Fatal("internal error leaked to response")
	}
}
