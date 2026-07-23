COMPOSE ?= docker compose

.PHONY: redis-up redis-down redis-logs redis-cli run test

redis-up:
	$(COMPOSE) up -d redis

redis-down:
	$(COMPOSE) down

redis-logs:
	$(COMPOSE) logs -f redis

redis-cli:
	$(COMPOSE) exec redis redis-cli

run:
	go run ./cmd/marketagregator

test:
	go test ./...
