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

docker-restart-network:
	docker compose down --remove-orphans
	docker compose up -d

# Development setup
dev: docker-restart-network
	@echo "🔧 Building Docker image..."
	@make docker-build
	@echo "📊 Setting up infrastructure..."
	@make check-db migrate-up
	@echo ""
	@echo "🚀 Development environment ready!"
	@echo "📊 Database: pitchlake-db (connected via pitchlake-network)"
	@echo "🔧 Docker image: Built successfully"
	@echo "📋 Next steps:"
	@echo "   • Run 'make docker-up' to start the application"
	@echo "   • Or run 'make docker-up-detached' to run in background"
	@echo "   • Use 'make docker-logs' to view application logs"
	@echo "   • Use 'make -f Makefile.infra migrate-status' to check database tables"


# Network commands
network-create:
	@echo "Creating pitchlake-network..."
	@if docker network ls | grep -q "pitchlake-network"; then \
		echo "✓ pitchlake-network already exists"; \
	else \
		docker network create pitchlake-network; \
		echo "✓ pitchlake-network created"; \
	fi

network-remove:
	@echo "⚠️  WARNING: This will remove the pitchlake-network!"
	@read -p "Are you sure you want to continue? (y/N): " confirm && [ "$$confirm" = "y" ] || exit 1; \
	if docker network ls | grep -q "pitchlake-network"; then \
		docker network rm pitchlake-network; \
		echo "✓ pitchlake-network removed"; \
	else \
		echo "✗ pitchlake-network does not exist"; \
	fi

network-status:
	@echo "Checking pitchlake-network status..."
	@if docker network ls | grep -q "pitchlake-network"; then \
		echo "✓ pitchlake-network exists"; \
		docker network inspect pitchlake-network --format "{{.IPAM.Config}}" 2>/dev/null || echo "Could not inspect network"; \
	else \
		echo "✗ pitchlake-network does not exist"; \
	fi

# Database commands
check-db:
	@if docker ps --format "table {{.Names}}" | grep -q "pitchlake-db"; then \
		echo "✓ pitchlake-db is running and accessible via pitchlake-network"; \
	else \
		echo "✗ pitchlake-db is not running. Please start your pitchlake-db container first."; \
	fi

# External database commands (affects external pitchlake-db)
db-create:
	@echo "Creating pitchlake-db container..."
	@if docker ps --format "table {{.Names}}" | grep -q "pitchlake-db"; then \
		echo "✓ pitchlake-db is already running"; \
	else \
		docker run -d \
			--name pitchlake-db \
			--network pitchlake-network \
			-e POSTGRES_DB=pitchlake \
			-e POSTGRES_USER=pitchlake_user \
			-e POSTGRES_PASSWORD=pitchlake_password \
			-p 5433:5432 \
			-v pitchlake_data:/var/lib/postgresql/data \
			postgres:15-alpine; \
		echo "✓ pitchlake-db created and started"; \
		echo "Waiting for database to be ready..."; \
		sleep 5; \
	fi

db-remove:
	@echo "⚠️  WARNING: This will remove the pitchlake-db container and all data!"
	@read -p "Are you sure you want to continue? (y/N): " confirm && [ "$$confirm" = "y" ] || exit 1; \
	if docker ps --format "table {{.Names}}" | grep -q "pitchlake-db"; then \
		docker stop pitchlake-db; \
		docker rm pitchlake-db; \
		docker volume rm pitchlake_data 2>/dev/null || echo "Volume already removed"; \
		echo "✓ pitchlake-db removed"; \
	else \
		echo "✗ pitchlake-db is not running"; \
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

# Vault management commands
add-vault:
	@echo "Adding vault to registry..."
	@if [ -z "$(VAULT_ADDRESS)" ] || [ -z "$(DEPLOYED_AT)" ]; then \
		echo "Usage: make add-vault VAULT_ADDRESS=0x... DEPLOYED_AT=0x..."; \
		echo "Example: make add-vault VAULT_ADDRESS=0x123... DEPLOYED_AT=0x456..."; \
		exit 1; \
	fi
	@echo "Vault Address: $(VAULT_ADDRESS)"
	@echo "Deployed At: $(DEPLOYED_AT)"
	@echo "⚠️  This will add the vault to the registry and start indexing from deployment block"
	@read -p "Are you sure you want to continue? (y/N): " confirm && [ "$$confirm" = "y" ] || exit 1
	@docker exec pitchlake-db psql -U pitchlake_user -d pitchlake -c "INSERT INTO vault_registry (vault_address, deployed_at, last_block_indexed, last_block_processed) VALUES ('$(VAULT_ADDRESS)', '$(DEPLOYED_AT)', NULL, NULL);" || echo "Failed to add vault. Make sure the container is running."

list-vaults:
	@echo "Listing all vaults in registry..."
	@docker exec pitchlake-db psql -U pitchlake_user -d pitchlake -c "SELECT vault_address, deployed_at, last_block_indexed, last_block_processed FROM vault_registry ORDER BY id;" || echo "Failed to list vaults. Make sure the container is running."
list-events:
	@docker exec pitchlake-db psql -U pitchlake_user -d pitchlake -c "SELECT * from events"
list-blocks:
	@docker exec pitchlake-db psql -U pitchlake_user -d pitchlake -c "SELECT * from starknet_blocks"

# Infrastructure help
help-infra:
	@echo "Infrastructure Commands:"
	@echo ""
	@echo "Network Commands:"
	@echo "  network-create    - Create pitchlake-network"
	@echo "  network-remove    - Remove pitchlake-network (with confirmation)"
	@echo "  network-status    - Check pitchlake-network status"
	@echo ""
	@echo "External Database Commands:"
	@echo "  db-create         - Create pitchlake-db container"
	@echo "  db-remove         - Remove pitchlake-db container and data (with confirmation)"
	@echo "  db-restart        - Restart pitchlake-db container"
	@echo "  db-logs           - View pitchlake-db logs"
	@echo "  check-db          - Check if pitchlake-db is running"
	@echo ""
	@echo "Local Database Commands:"
	@echo "  db-up-local       - Start local database (alternative to pitchlake-db)"
	@echo "  db-down-local     - Stop local database"
	@echo "  db-logs-local     - View local database logs"
	@echo ""
	@echo "Migration Commands:"
	@echo "  migrate-up        - Run database migrations"
	@echo "  migrate-down      - Roll back database migrations (with confirmation)"
	@echo "  migrate-status    - Check migration status"
	@echo ""
	@echo "Vault Management Commands:"
	@echo "  add-vault         - Add a new vault to the registry"
	@echo "  list-vaults       - List all vaults in the registry"
	@echo ""
	@echo "Help:"
	@echo "  help-infra        - Show this help message"