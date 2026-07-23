package main

import (
	"context"
	"log"
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
	redisCache := connectRedis()
	if redisCache != nil {
		defer redisCache.Close()
	}

	service := search.New(redisCache, ozon.New(), wb.New())
	handler := httpapi.New(service, searchTimeout)

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

	log.Printf("server started on :%s", port)
	log.Fatal(server.ListenAndServe())
}

func connectRedis() *cache.Redis {
	redisCache := cache.NewFromEnv()
	for attempt := 1; attempt <= 4; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := redisCache.Ping(ctx)
		cancel()
		if err == nil {
			log.Println("connected to Redis")
			return redisCache
		}
		log.Printf("Redis connection attempt %d failed: %v", attempt, err)
		if attempt < 4 {
			time.Sleep(2 * time.Second)
		}
	}

	if err := redisCache.Close(); err != nil {
		log.Printf("close Redis client: %v", err)
	}
	log.Println("starting without cache")
	return nil
}
