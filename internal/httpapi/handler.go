package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"agregator/internal/search"
)

type Handler struct {
	search  *search.Service
	timeout time.Duration
	logger  *slog.Logger
}

func New(logger *slog.Logger, searchService *search.Service, timeout time.Duration) *Handler {
	return &Handler{logger: logger, search: searchService, timeout: timeout}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		h.logger.Warn("request without query")
		http.Error(w, "query parameter is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	products, err := h.search.Search(ctx, query)
	if err != nil {
		h.logger.Error("search failed", "query", query, "error", err)
		if errors.Is(err, search.ErrProductsNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(products); err != nil {
		h.logger.Error("encode search response", "error", err)
		http.Error(w, "encode response", http.StatusInternalServerError)
	}
}
