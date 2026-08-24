.PHONY: compose-up compose-down build run test vet migrate-down

# 开发环境数据库（PG + Redis）
compose-up:
	docker compose -f deploy/docker-compose.dev.yml up -d

compose-down:
	docker compose -f deploy/docker-compose.dev.yml down

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

test:
	go test ./...

vet:
	go vet ./...

# 手工回滚全部迁移（测试可逆脚本用；正常勿跑）
migrate-down:
	psql "$$DATABASE_URL" -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;" >/dev/null 2>&1 || true
	@echo "手工回滚请参考 0001_init.down.sql / 0002_seed.down.sql 顺序执行"
