package cache

import (
	"context"
	"testing"

	"agregator/internal/product"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func TestRedisSetAndGet(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := &Redis{client: client}
	t.Cleanup(func() { _ = store.Close() })

	want := []product.Product{{
		ProductID:            "42",
		ProductName:          "Phone",
		DiscountPriceKopecks: 123_400,
		BasePriceKopecks:     150_000,
	}}
	if err := store.Set(context.Background(), "phone", want); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := store.Get(context.Background(), "phone")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("Get() = %#v, want %#v", got, want)
	}
}
