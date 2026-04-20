FRONT_END_BINARY=frontApp
BROKER_BINARY=brokerApp
COMPOSE := docker compose

## up: starts all containers in the background without forcing build
up:
	@echo "Starting Docker images..."
	$(COMPOSE) up -d
	@echo "Docker images started!"

## up_build: stops docker-compose (if running), builds all projects and starts docker compose
up_build:
	@echo "Stopping docker images (if running...)"
	$(COMPOSE) down
	@echo "Building (when required) and starting docker images..."
	$(COMPOSE) up --build -d
	@echo "Docker images built and started!"

## down: stop docker compose
down:
	@echo "Stopping docker compose..."
	$(COMPOSE) down
	@echo "Done!"

## build_broker: builds the broker-service image (compiles inside Docker)
build_broker:
	@echo "Building broker-service Docker image (binary is built inside the Dockerfile)..."
	$(COMPOSE) build broker-service
	@echo "Done!"

## build: builds all docker images (compiles inside Dockerfiles)
build:
	@echo "Building all Docker images (binaries are built inside Dockerfiles)..."
	$(COMPOSE) build
	@echo "Done!"

## build_front: builds the front end binary into bin/ folder
build_front:
	@echo "Building front end binary..."
	cd ../front-end && env CGO_ENABLED=0 go build -o ../bin/${FRONT_END_BINARY} ./cmd/web
	@echo "Done!"

## start: starts the front end
start: build_front
	@echo "Starting front end"
	@echo "Running binary from front-end folder to ensure runtime assets are found..."
	cd ../front-end && ../bin/${FRONT_END_BINARY} &

## stop: stop the front end
stop:
	@echo "Stopping front end..."
	@-pkill -SIGTERM -f "../bin/${FRONT_END_BINARY}"
	@echo "Stopped front end!"