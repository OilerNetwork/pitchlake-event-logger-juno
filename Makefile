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
	@make -f Makefile.infra check-db migrate-up
	@echo ""
	@echo "🚀 Development environment ready!"
	@echo "📊 Database: pitchlake-db (connected via pitchlake-network)"
	@echo "🔧 Docker image: Built successfully"
	@echo "📋 Next steps:"
	@echo "   • Run 'make docker-up' to start the application"
	@echo "   • Or run 'make docker-up-detached' to run in background"
	@echo "   • Use 'make docker-logs' to view application logs"
	@echo "   • Use 'make -f Makefile.infra migrate-status' to check database tables"
