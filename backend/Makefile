MIGRATIONS_DIR := migrations
MIGRATE        := migrate
DATABASE_URL   ?= $(shell cd .. && grep -m1 "^DATABASE_URL=" .env | cut -d= -f2-)

.PHONY: migrate-up migrate-down migrate-version migrate-create

## Apply all pending migrations.
migrate-up:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

## Roll back the most recent migration.
migrate-down:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

## Print the current migration version.
migrate-version:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" version

## Scaffold a new migration pair: make migrate-create NAME=description
migrate-create:
	@test -n "$(NAME)" || (echo "Usage: make migrate-create NAME=<description>" && exit 1)
	$(MIGRATE) -path $(MIGRATIONS_DIR) create -ext sql -dir $(MIGRATIONS_DIR) -seq $(NAME)
