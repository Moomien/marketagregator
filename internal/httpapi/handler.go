package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"agregator/internal/search"
)

type Handler struct {
	search  *search.Service
	timeout time.Duration
}

func New(searchService *search.Service, timeout time.Duration) *Handler {
	return &Handler{search: searchService, timeout: timeout}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		http.Error(w, "query parameter is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	products, err := h.search.Search(ctx, query)
	if err != nil {
		if errors.Is(err, search.ErrProductsNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(products); err != nil {
		http.Error(w, "encode response", http.StatusInternalServerError)
	}
}
