.PHONY: all build dev test lint clean deps

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)"

all: deps lint test build

deps:
	go mod download
	cd web && npm ci

build:
	@mkdir -p build
	go build $(LDFLAGS) -o build/keldris-server ./cmd/keldris-server
	go build $(LDFLAGS) -o build/keldris-agent ./cmd/keldris-agent
	cd web && npm run build

build-agent-all:
	@mkdir -p build
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o build/keldris-agent-linux-amd64 ./cmd/keldris-agent
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o build/keldris-agent-linux-arm64 ./cmd/keldris-agent
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o build/keldris-agent-darwin-amd64 ./cmd/keldris-agent
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o build/keldris-agent-darwin-arm64 ./cmd/keldris-agent
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o build/keldris-agent-windows-amd64.exe ./cmd/keldris-agent

dev:
	@trap 'kill 0' EXIT; \
	go run ./cmd/keldris-server & \
	cd web && npm run dev & \
	wait

test:
	go test -race -cover ./...

lint:
	go vet ./...
	staticcheck ./...
	cd web && npx @biomejs/biome check .

fmt:
	gofmt -w .
	cd web && npx @biomejs/biome check --write .

clean:
	rm -rf build coverage.out
	cd web && rm -rf dist

docker-up:
	docker compose -f docker/docker-compose.yml up -d

docker-down:
	docker compose -f docker/docker-compose.yml down

hooks:
	lefthook install
