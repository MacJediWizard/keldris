.PHONY: all build dev test lint clean deps swagger swagger-fmt

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)"

all: deps lint test build

deps:
	go mod download
	go install honnef.co/go/tools/cmd/staticcheck@latest
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

# Code signing configuration (set these environment variables for signed builds)
# APPLE_DEVELOPER_ID - Apple Developer ID for macOS codesign (e.g., "Developer ID Application: Your Name (TEAMID)")
# APPLE_KEYCHAIN_PROFILE - Notarization keychain profile name
# WINDOWS_SIGN_CERT - Path to Windows code signing certificate (.pfx)
# WINDOWS_SIGN_PASS - Password for Windows code signing certificate

build-agent-signed: build-agent-all sign-macos sign-windows
	@echo "Signed builds complete"

sign-macos:
ifdef APPLE_DEVELOPER_ID
	@echo "Signing macOS binaries..."
	codesign --sign "$(APPLE_DEVELOPER_ID)" --options runtime --timestamp build/keldris-agent-darwin-amd64
	codesign --sign "$(APPLE_DEVELOPER_ID)" --options runtime --timestamp build/keldris-agent-darwin-arm64
ifdef APPLE_KEYCHAIN_PROFILE
	@echo "Notarizing macOS binaries..."
	@for arch in amd64 arm64; do \
		zip -j build/keldris-agent-darwin-$$arch.zip build/keldris-agent-darwin-$$arch; \
		xcrun notarytool submit build/keldris-agent-darwin-$$arch.zip --keychain-profile "$(APPLE_KEYCHAIN_PROFILE)" --wait; \
		rm build/keldris-agent-darwin-$$arch.zip; \
	done
endif
else
	@echo "Skipping macOS signing (APPLE_DEVELOPER_ID not set)"
endif

sign-windows:
ifdef WINDOWS_SIGN_CERT
	@echo "Signing Windows binary..."
	signtool sign /f "$(WINDOWS_SIGN_CERT)" /p "$(WINDOWS_SIGN_PASS)" /tr http://timestamp.digicert.com /td sha256 /fd sha256 build/keldris-agent-windows-amd64.exe
else
	@echo "Skipping Windows signing (WINDOWS_SIGN_CERT not set)"
endif

dev:
	@trap 'kill 0' EXIT; \
	go run ./cmd/keldris-server & \
	cd web && npm run dev & \
	wait

test:
	go test -race -cover ./...
	cd web && npx vitest run

lint:
	go vet ./...
	@which staticcheck > /dev/null && staticcheck ./... || echo "staticcheck not installed, skipping"
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

swagger:
	swag init -g cmd/keldris-server/main.go -o docs/api --parseInternal

swagger-fmt:
	swag fmt
