# Redis TUI Manager Makefile
SHELL := /bin/bash
APP_NAME := redis-tui
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: all build install clean test bench test-cover test-cover-check test-cross-device lint run start release snapshot demo screenshots \
	docker-up docker-down docker-seed docker-up-all docker-down-all docker-seed-all \
	docker-up-standalone docker-up-standalone-stack docker-up-cluster docker-up-cluster-stack \
	docker-down-standalone docker-down-standalone-stack docker-down-cluster docker-down-cluster-stack \
	docker-seed-standalone docker-seed-standalone-stack docker-seed-cluster docker-seed-cluster-stack

all: build

## Build the application
build:
	go build $(LDFLAGS) -o bin/$(APP_NAME) ./

## Install to GOPATH/bin
install:
	go install $(LDFLAGS) ./

## Clean build artifacts
clean:
	rm -rf bin/
	rm -rf dist/

## Run tests
test:
	go test -v -race ./...

## Run performance benchmarks (hot paths: key scans, preview fetches, rendering)
bench:
	go test -run XXX -bench . -benchmem -benchtime=5x -count=3 ./internal/redis/ ./internal/ui/

## Run tests with coverage
test-cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## Run tests and fail if any function is below 100% coverage
test-cover-check:
	@go test -v -race -coverprofile=coverage.out ./... && \
		set -euo pipefail; \
		FAILED=0; \
		while IFS= read -r line; do \
			func=$$(echo "$$line" | awk '{print $$2}'); \
			pct=$$(echo "$$line" | awk '{print $$NF}' | tr -d '%'); \
			if [[ "$$func" == "(statements)" ]]; then \
				continue; \
			fi; \
			if (( $$(echo "$$pct < 100.0" | bc -l) )); then \
				location=$$(echo "$$line" | awk '{print $$1}'); \
				echo "FAIL: Function $$func at $$location coverage is $${pct}%, required 100%"; \
				FAILED=1; \
			fi; \
		done < <(go tool cover -func=coverage.out); \
		if [[ $$FAILED -eq 1 ]]; then \
			exit 1; \
		fi; \
		echo "All functions at 100% coverage"

## Reproduce issue #71 on Ubuntu 26.04 (/tmp tmpfs vs dest disk) and verify the copy fallback.
test-cross-device:
	docker build --load -f scripts/Dockerfile.cross-device -t redis-tui-cross-device scripts/
	docker run --rm \
		--tmpfs /tmp:exec,mode=1777,size=512m \
		-v "$(CURDIR)":/src \
		-w /src \
		-e CROSS_DEVICE_SRC=/tmp \
		-e CROSS_DEVICE_DEST=/opt/dest \
		-e REQUIRE_CROSS_DEVICE=1 \
		redis-tui-cross-device \
		sh scripts/test-cross-device-update.sh

## Run linter
lint:
	go vet ./...

## Format code
fmt:
	go fmt ./...

## Build and run the application
run: build
	./bin/$(APP_NAME)

## Alias for run
start: run

## Run the application in debug mode
debug-server:
	@mkdir -p tmp || true
	@-pkill -f "dlv.*38697" || true
	go build -gcflags="all=-N -l" -o tmp/$(APP_NAME)-debug ./
	go run github.com/go-delve/delve/cmd/dlv@latest exec ./tmp/$(APP_NAME)-debug --headless --listen=127.0.0.1:38697 --api-version=2
	@printf "\033[?1049l\033[?25h"
	@stty sane
	@reset

## Build for multiple platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-darwin-amd64 ./
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(APP_NAME)-darwin-arm64 ./
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-linux-amd64 ./
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(APP_NAME)-linux-arm64 ./
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-windows-amd64.exe ./

## Create a release with goreleaser
release:
	goreleaser release --clean

## Create a snapshot release (no publish)
snapshot:
	goreleaser release --snapshot --clean

## Install development dependencies
dev-deps:
	go install github.com/goreleaser/goreleaser/v2@v2.13.1
	go install github.com/go-delve/delve/cmd/dlv@latest

## --- Docker Examples ---

## Start default example Redis instances (plain redis only — no redis-stack)
docker-up: docker-up-standalone docker-up-cluster

## Stop default example Redis instances
docker-down: docker-down-standalone docker-down-cluster

## Seed default example instances
docker-seed: docker-seed-standalone docker-seed-cluster

## Start / stop / seed every compose file including redis-stack
docker-up-all: docker-up-standalone docker-up-standalone-stack docker-up-cluster docker-up-cluster-stack
docker-down-all: docker-down-standalone docker-down-standalone-stack docker-down-cluster docker-down-cluster-stack
docker-seed-all: docker-seed-standalone docker-seed-standalone-stack docker-seed-cluster docker-seed-cluster-stack

## Standalone (redis:7-alpine on :6379)
docker-up-standalone:
	docker compose -f examples/standalone/docker-compose.yml up -d

docker-down-standalone:
	docker compose -f examples/standalone/docker-compose.yml down

docker-seed-standalone:
	go run ./examples/seed -flush

## Standalone Redis Stack (redis-stack on :6390)
docker-up-standalone-stack:
	docker compose -f examples/standalone-redis-stack/docker-compose.yml up -d

docker-down-standalone-stack:
	docker compose -f examples/standalone-redis-stack/docker-compose.yml down

docker-seed-standalone-stack:
	go run ./examples/seed -addr localhost:6390 -flush

## Cluster (redis:7-alpine on :6380-6385)
docker-up-cluster:
	docker compose -f examples/cluster/docker-compose.yml up -d

docker-down-cluster:
	docker compose -f examples/cluster/docker-compose.yml down

docker-seed-cluster:
	go run ./examples/seed -addr localhost:6380 -cluster -flush

## Cluster Redis Stack (redis-stack on :6386-6392)
docker-up-cluster-stack:
	docker compose -f examples/cluster-redis-stack/docker-compose.yml up -d

docker-down-cluster-stack:
	docker compose -f examples/cluster-redis-stack/docker-compose.yml down

docker-seed-cluster-stack:
	go run ./examples/seed -addr localhost:6386 -cluster -flush

## --- Demo ---

## Render the README demo GIF against an isolated demo config.
## Built without $(LDFLAGS) so main.version stays "dev" and the update banner is suppressed.
## Unset NO_COLOR so lipgloss emits ANSI colors under VHS (agent shells often set NO_COLOR=1).
demo: docker-up-standalone docker-up-cluster docker-seed-standalone docker-seed-cluster
	go build -o bin/$(APP_NAME) ./
	env -u NO_COLOR -u FORCE_COLOR COLORTERM=truecolor TERM=xterm-256color CLICOLOR_FORCE=1 FORCE_COLOR=1 vhs docs/demo.tape

## Capture static README screenshots (docs/main.png, keys-preview.png, protobuf.png, detail.png).
screenshots: docker-up-standalone docker-seed-standalone
	go build -o bin/$(APP_NAME) ./
	env -u NO_COLOR -u FORCE_COLOR COLORTERM=truecolor TERM=xterm-256color CLICOLOR_FORCE=1 FORCE_COLOR=1 vhs docs/screenshots.tape

## Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "  Build & Dev:"
	@echo "    build       - Build the application"
	@echo "    install     - Install to GOPATH/bin"
	@echo "    clean       - Clean build artifacts"
	@echo "    test        - Run tests"
	@echo "    test-cover        - Run tests with coverage"
	@echo "    test-cover-check  - Fail if any function < 100%%"
	@echo "    test-cross-device - Ubuntu 26.04 EXDEV self-update repro"
	@echo "    lint        - Run linter"
	@echo "    fmt         - Format code"
	@echo "    run         - Build and run the application"
	@echo "    start       - Alias for run"
	@echo "    build-all   - Build for multiple platforms"
	@echo "    release     - Create a release with goreleaser"
	@echo "    snapshot    - Create a snapshot release"
	@echo "    dev-deps    - Install development dependencies"
	@echo ""
	@echo "  Docker Examples:"
	@echo "    docker-up                  - Start default instances (standalone + cluster)"
	@echo "    docker-down                - Stop default instances"
	@echo "    docker-seed                - Seed default instances"
	@echo "    docker-up-all              - Start all instances including redis-stack"
	@echo "    docker-down-all            - Stop all instances including redis-stack"
	@echo "    docker-seed-all            - Seed all instances including redis-stack"
	@echo "    docker-up-standalone       - Standalone (:6379)"
	@echo "    docker-up-standalone-stack - Standalone Redis Stack (:6390)"
	@echo "    docker-up-cluster          - Cluster (:6380-6385)"
	@echo "    docker-up-cluster-stack    - Cluster Redis Stack (:6386-6392)"
	@echo ""
	@echo "  Demo:"
	@echo "    demo        - Render the README demo GIF"
	@echo "    screenshots - Capture README static screenshots"
	@echo ""
	@echo "    help        - Show this help"
