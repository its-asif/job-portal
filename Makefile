-include .env
export

MIGRATE_IMAGE ?= migrate/migrate:v4.18.3
MIGRATIONS_DIR ?= $(CURDIR)/db/migrations
MIGRATE_NETWORK ?= host

.PHONY: migrate-up migrate-down

migrate-up:
	@if [ -z "$(DB_URL)" ]; then echo "DB_URL is not set"; exit 1; fi
	docker run --rm \
		--network "$(MIGRATE_NETWORK)" \
		-v "$(MIGRATIONS_DIR):/migrations" \
		$(MIGRATE_IMAGE) \
		-path=/migrations -database "$(DB_URL)" up

migrate-down:
	@if [ -z "$(DB_URL)" ]; then echo "DB_URL is not set"; exit 1; fi
	docker run --rm \
		--network "$(MIGRATE_NETWORK)" \
		-v "$(MIGRATIONS_DIR):/migrations" \
		$(MIGRATE_IMAGE) \
		-path=/migrations -database "$(DB_URL)" down 1
