package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"agregator/internal/cache"
	"agregator/internal/httpapi"
	"agregator/internal/marketplace/ozon"
	"agregator/internal/marketplace/wb"
	"agregator/internal/search"
)

const searchTimeout = 60 * time.Second

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	redisCache := connectRedis(logger)
	if redisCache != nil {
		defer redisCache.Close()
	}

	service := search.New(logger.With("component", "search"), redisCache, ozon.New(logger), wb.New(logger))
	handler := httpapi.New(logger.With("component", "http"), service, searchTimeout)

	mux := http.NewServeMux()
	mux.HandleFunc("/search", handler.Search)
	mux.Handle("/", http.FileServer(http.Dir("web/dist")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	logger.Info("server started", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server stopped unexpectedly", "error", err)
	}
}

func connectRedis(logger *slog.Logger) *cache.Redis {
	redisCache := cache.NewFromEnv()
	for attempt := 1; attempt <= 4; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := redisCache.Ping(ctx)
		cancel()
		if err == nil {
			logger.Info("connected to Redis")
			return redisCache
		}
		logger.Warn("Redis connection attempt failed", "attempt", attempt, "error", err)
		if attempt < 4 {
			time.Sleep(2 * time.Second)
		}
	}

	if err := redisCache.Close(); err != nil {
		logger.Warn("close Redis client", "error", err)
	}
	logger.Warn("starting without cache")
	return nil
}
