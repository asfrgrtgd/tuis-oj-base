MIGRATE ?= migrate
# DB_URL 解決: 1) 環境変数 POSTGRES_URL があればそれを使用 2) .env に POSTGRES_URL があればそれを読む
DB_URL ?= $(shell if [ -n "$$POSTGRES_URL" ]; then echo $$POSTGRES_URL; elif [ -f .env ]; then . ./.env && echo $$POSTGRES_URL; fi)

.PHONY: migrate-up migrate-down migrate-force

# Always use api image as migrate runner (migrate binary + /migrations bind mount)
migrate-up:
	@docker compose run --rm \
		-e POSTGRES_URL="$(DB_URL)" \
		--entrypoint migrate \
		api -path=/migrations -database "$(DB_URL)" up

migrate-down:
	@docker compose run --rm \
		-e POSTGRES_URL="$(DB_URL)" \
		--entrypoint migrate \
		api -path=/migrations -database "$(DB_URL)" down 1

migrate-force:
	@docker compose run --rm \
		-e POSTGRES_URL="$(DB_URL)" \
		--entrypoint migrate \
		api -path=/migrations -database "$(DB_URL)" force

