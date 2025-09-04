ifeq ($(VM_DEBUG),true)
    GO_TAGS = -tags 'vm_debug,no_coverage'
    VM_TARGET = debug
else
    GO_TAGS = -tags 'no_coverage'
    VM_TARGET = all
endif

build:
	go build $(GO_TAGS) -a -ldflags="-X main.Version=$(shell git describe --tags)" -buildmode=plugin -o myplugin.so plugin/myplugin.go

# Docker commands
docker-build:
	docker compose build

docker-up:
	docker compose up

docker-up-detached:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

docker-restart:
	docker compose restart

docker-clean:
	docker compose down -v --remove-orphans
	docker system prune -f

# Database commands
check-db:
	@if docker ps --format "table {{.Names}}" | grep -q "pitchlake-db"; then \
		echo "✓ pitchlake-db is already running"; \
	else \
		echo "✗ pitchlake-db is not running"; \
	fi

db-up:
	@if docker ps --format "table {{.Names}}" | grep -q "pitchlake-db"; then \
		echo "Using existing pitchlake-db container"; \
	else \
		echo "Starting new pitchlake-db container..."; \
		docker compose up db -d; \
		echo "Waiting for database to be ready..."; \
		sleep 5; \
	fi

db-down:
	@if docker ps --format "table {{.Names}}" | grep -q "pitchlake-db"; then \
		echo "Stopping pitchlake-db..."; \
		docker compose stop db; \
	else \
		echo "pitchlake-db is not running"; \
	fi

db-logs:
	@if docker ps --format "table {{.Names}}" | grep -q "pitchlake-db"; then \
		docker compose logs -f db; \
	else \
		echo "pitchlake-db is not running"; \
	fi

# Migration commands
migrate-up:
	@echo "Running database migrations..."
	@if [ -f .env ]; then \
		export $$(cat .env | xargs); \
	fi; \
	if docker ps --format "table {{.Names}}" | grep -q "pitchlake-db"; then \
		echo "Using existing pitchlake-db container for migrations"; \
		DB_CONTAINER="pitchlake-db"; \
	elif docker ps --format "table {{.Names}}" | grep -q "juno-indexer-db"; then \
		echo "Using juno-indexer-db container for migrations"; \
		DB_CONTAINER="juno-indexer-db"; \
	else \
		echo "No database container found. Please start a database first."; \
		exit 1; \
	fi; \
	if [ -z "$$DB_URL" ]; then \
		echo "Error: DB_URL environment variable is not set"; \
		echo "Please set DB_URL in your .env file (e.g., postgres://user:password@localhost:5433/database)"; \
		exit 1; \
	fi; \
	if [ "$$DB_CONTAINER" = "pitchlake-db" ]; then \
		DB_CONNECTION="postgres://pitchlake_user:pitchlake_password@localhost:5432/pitchlake"; \
	else \
		DB_CONNECTION="$$DB_URL"; \
	fi; \
	docker exec -i $$DB_CONTAINER psql "$$DB_CONNECTION" < db/migrations/000001_create_events_table.up.sql; \
	docker exec -i $$DB_CONTAINER psql "$$DB_CONNECTION" < db/migrations/000002_create_starknet_blocks_table.up.sql; \
	docker exec -i $$DB_CONTAINER psql "$$DB_CONNECTION" < db/migrations/000003_vault_registry.up.sql; \
	echo "✓ All migrations completed!"

migrate-down:
	@echo "Rolling back database migrations..."
	@if [ -f .env ]; then \
		export $$(cat .env | xargs); \
	fi; \
	if docker ps --format "table {{.Names}}" | grep -q "pitchlake-db"; then \
		echo "Using existing pitchlake-db container for rollback"; \
		DB_CONTAINER="pitchlake-db"; \
	elif docker ps --format "table {{.Names}}" | grep -q "juno-indexer-db"; then \
		echo "Using juno-indexer-db container for rollback"; \
		DB_CONTAINER="juno-indexer-db"; \
	else \
		echo "No database container found. Please start a database first."; \
		exit 1; \
	fi; \
	if [ -z "$$DB_URL" ]; then \
		echo "Error: DB_URL environment variable is not set"; \
		echo "Please set DB_URL in your .env file (e.g., postgres://user:password@localhost:5433/database)"; \
		exit 1; \
	fi; \
	if [ "$$DB_CONTAINER" = "pitchlake-db" ]; then \
		DB_CONNECTION="postgres://pitchlake_user:pitchlake_password@localhost:5432/pitchlake"; \
	else \
		DB_CONNECTION="$$DB_URL"; \
	fi; \
	docker exec -i $$DB_CONTAINER psql "$$DB_CONNECTION" < db/migrations/000003_create_vault_registry.down.sql; \
	docker exec -i $$DB_CONTAINER psql "$$DB_CONNECTION" < db/migrations/000002_create_starknet_blocks_table.down.sql; \
	docker exec -i $$DB_CONTAINER psql "$$DB_CONNECTION" < db/migrations/000001_create_events_table.down.sql; \
	echo "✓ All migrations rolled back!"

migrate-status:
	@echo "Checking migration status..."
	@if [ -f .env ]; then \
		export $$(cat .env | xargs); \
	fi; \
	if docker ps --format "table {{.Names}}" | grep -q "pitchlake-db"; then \
		echo "Using existing pitchlake-db container"; \
		DB_CONTAINER="pitchlake-db"; \
	elif docker ps --format "table {{.Names}}" | grep -q "juno-indexer-db"; then \
		echo "Using juno-indexer-db container"; \
		DB_CONTAINER="juno-indexer-db"; \
	else \
		echo "No database container found. Please start a database first."; \
		exit 1; \
	fi; \
	if [ -z "$$DB_URL" ]; then \
		echo "Error: DB_URL environment variable is not set"; \
		echo "Please set DB_URL in your .env file (e.g., postgres://user:password@localhost:5433/database)"; \
		exit 1; \
	fi; \
	echo "Checking tables..."; \
	if [ "$$DB_CONTAINER" = "pitchlake-db" ]; then \
		DB_CONNECTION="postgres://pitchlake_user:pitchlake_password@localhost:5432/pitchlake"; \
	else \
		DB_CONNECTION="$$DB_URL"; \
	fi; \
	docker exec $$DB_CONTAINER psql "$$DB_CONNECTION" -c "\dt" 2>/dev/null || echo "Could not connect to database"

# Development setup
dev: check-db migrate-up
	@echo ""
	@echo "🚀 Development environment ready!"
	@echo "📊 Database: $(shell if docker ps --format 'table {{.Names}}' | grep -q 'pitchlake-db'; then echo 'pitchlake-db (existing)'; else echo 'juno-indexer-db (local)'; fi)"
	@echo "📋 Next steps:"
	@echo "   • Run 'make docker-build' to build the Docker image"
	@echo "   • Then run 'make docker-up' to start the application"
	@echo "   • Or run 'make docker-up-detached' to run in background"
	@echo "   • Use 'make docker-logs' to view application logs"
