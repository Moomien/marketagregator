package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"agregator/internal/product"

	"github.com/go-redis/redis/v8"
)

const defaultAddress = "localhost:6379"

type Redis struct {
	client *redis.Client
}

func NewFromEnv() *Redis {
	address := os.Getenv("REDIS_ADDR")
	if address == "" {
		address = defaultAddress
	}

	return &Redis{client: redis.NewClient(&redis.Options{
		Addr:     address,
		Password: os.Getenv("REDIS_PASSWORD"),
	})}
}

func (r *Redis) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *Redis) Get(ctx context.Context, query string) ([]product.Product, error) {
	value, err := r.client.Get(ctx, query).Result()
	if err != nil {
		return nil, err
	}

	var products []product.Product
	if err := json.Unmarshal([]byte(value), &products); err != nil {
		return nil, fmt.Errorf("decode cached products: %w", err)
	}
	return products, nil
}

func (r *Redis) Set(ctx context.Context, query string, products []product.Product) error {
	value, err := json.Marshal(products)
	if err != nil {
		return fmt.Errorf("encode products for cache: %w", err)
	}
	return r.client.Set(ctx, query, value, time.Hour).Err()
}

func (r *Redis) Close() error {
	return r.client.Close()
}
