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
		echo "✓ pitchlake-db is running and accessible via pitchlake-network"; \
	else \
		echo "✗ pitchlake-db is not running. Please start your pitchlake-db container first."; \
	fi

# Local database commands (only if you want to use local db instead of pitchlake-db)
db-up-local:
	docker compose --profile local-db up db -d

db-down-local:
	docker compose --profile local-db down

db-logs-local:
	docker compose --profile local-db logs -f db

# Migration commands (using pitchlake-db via network)
migrate-up:
	@echo "Running database migrations on pitchlake-db..."
	@if ! docker ps --format "table {{.Names}}" | grep -q "pitchlake-db"; then \
		echo "Error: pitchlake-db is not running. Please start your pitchlake-db container first."; \
		exit 1; \
	fi; \
	echo "Checking if migrations are needed..."; \
	if docker exec pitchlake-db psql -U pitchlake_user -d pitchlake -c "\dt" 2>/dev/null | grep -q "events"; then \
		echo "✓ events table already exists"; \
	else \
		echo "Creating events table..."; \
		docker exec -i pitchlake-db psql -U pitchlake_user -d pitchlake < db/migrations/000001_create_events_table.up.sql; \
	fi; \
	if docker exec pitchlake-db psql -U pitchlake_user -d pitchlake -c "\dt" 2>/dev/null | grep -q "starknet_blocks"; then \
		echo "✓ starknet_blocks table already exists"; \
	else \
		echo "Creating starknet_blocks table..."; \
		docker exec -i pitchlake-db psql -U pitchlake_user -d pitchlake < db/migrations/000002_create_starknet_blocks_table.up.sql; \
	fi; \
	if docker exec pitchlake-db psql -U pitchlake_user -d pitchlake -c "\dt" 2>/dev/null | grep -q "vault_registry"; then \
		echo "✓ vault_registry table already exists"; \
	else \
		echo "Creating vault_registry table..."; \
		docker exec -i pitchlake-db psql -U pitchlake_user -d pitchlake < db/migrations/000003_vault_registry.up.sql; \
	fi; \
	echo "✓ All migrations completed!"

migrate-down:
	@echo "Rolling back database migrations on pitchlake-db..."
	@if ! docker ps --format "table {{.Names}}" | grep -q "pitchlake-db"; then \
		echo "Error: pitchlake-db is not running. Please start your pitchlake-db container first."; \
		exit 1; \
	fi; \
	echo "⚠️  WARNING: This will drop all tables and data!"; \
	read -p "Are you sure you want to continue? (y/N): " confirm && [ "$$confirm" = "y" ] || exit 1; \
	if docker exec pitchlake-db psql -U pitchlake_user -d pitchlake -c "\dt" 2>/dev/null | grep -q "vault_registry"; then \
		echo "Dropping vault_registry table..."; \
		docker exec -i pitchlake-db psql -U pitchlake_user -d pitchlake < db/migrations/000003_create_vault_registry.down.sql; \
	fi; \
	if docker exec pitchlake-db psql -U pitchlake_user -d pitchlake -c "\dt" 2>/dev/null | grep -q "starknet_blocks"; then \
		echo "Dropping starknet_blocks table..."; \
		docker exec -i pitchlake-db psql -U pitchlake_user -d pitchlake < db/migrations/000002_create_starknet_blocks_table.down.sql; \
	fi; \
	if docker exec pitchlake-db psql -U pitchlake_user -d pitchlake -c "\dt" 2>/dev/null | grep -q "events"; then \
		echo "Dropping events table..."; \
		docker exec -i pitchlake-db psql -U pitchlake_user -d pitchlake < db/migrations/000001_create_events_table.down.sql; \
	fi; \
	echo "✓ All migrations rolled back!"

migrate-status:
	@echo "Checking migration status on pitchlake-db..."
	@if ! docker ps --format "table {{.Names}}" | grep -q "pitchlake-db"; then \
		echo "Error: pitchlake-db is not running. Please start your pitchlake-db container first."; \
		exit 1; \
	fi; \
	echo "Checking tables..."; \
	docker exec pitchlake-db psql -U pitchlake_user -d pitchlake -c "\dt" 2>/dev/null || echo "Could not connect to database"

# Development setup
dev: check-db migrate-up docker-build
	@echo ""
	@echo "🚀 Development environment ready!"
	@echo "📊 Database: pitchlake-db (connected via pitchlake-network)"
	@echo "🔧 Docker image: Built successfully"
	@echo "📋 Next steps:"
	@echo "   • Run 'make docker-up' to start the application"
	@echo "   • Or run 'make docker-up-detached' to run in background"
	@echo "   • Use 'make docker-logs' to view application logs"
	@echo "   • Use 'make migrate-status' to check database tables"
