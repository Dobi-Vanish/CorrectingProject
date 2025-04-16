SHELL=cmd.exe
REWARD_BINARY=rewardApp
PROJECT_ROOT=..
BUILD_DIR=$(PROJECT_ROOT)\build

## up: starts all containers in the background without forcing build
up:
	@echo Starting Docker images...
	docker-compose up -d
	@echo Docker images started!

## up_build: stops docker-compose (if running), builds all projects and starts docker compose
up_build: build_reward
	@echo Stopping docker images (if running...)
	docker-compose down
	@echo Building (when required) and starting docker images...
	docker-compose up --build -d
	@echo Docker images built and started!

## down: stop docker compose
down:
	@echo Stopping docker compose...
	docker-compose down
	@echo Done!

## build_reward: builds the reward binary as a linux executable
build_reward:
	@echo Building reward binary...
	@if not exist "$(BUILD_DIR)" mkdir "$(BUILD_DIR)"
	set GOOS=linux&& set GOARCH=amd64&& set CGO_ENABLED=0&& \
	cd "$(PROJECT_ROOT)" && go build -o "$(REWARD_BINARY)" ./cmd/app
	@echo Done!