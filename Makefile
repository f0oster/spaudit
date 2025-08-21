.PHONY: deps deps-update gen build run test test-coverage lint lint-fast fmt fmt-check tidy-check generate-check clean clean-all install check-env db-init assets dev verify help

# Default target
.DEFAULT_GOAL := help

# Simple cross-platform (using bash everywhere)
SHELL := /bin/bash
RM = rm -rf
MKDIR = mkdir -p
BINARY_EXT = .exe

# Build configuration
BINARY_NAME = server$(BINARY_EXT)
BUILD_DIR   = bin
CMD_DIR     = cmd/server

# Tool versions
SQLC_VERSION       ?= v1.29.0
TEMPL_VERSION      ?= v0.3.920
GOLANGCI_VERSION   ?= v1.60.3

# Where to install tools
GOBIN ?= $(shell go env GOBIN)
ifeq ($(GOBIN),)
  GOBIN := $(shell go env GOPATH)/bin
endif
export GOBIN

# --- Tools & deps -------------------------------------------------------------

## deps: Install CLI tooling (sqlc, templ, goimports, staticcheck, golangci-lint)
deps:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
	go install github.com/a-h/templ/cmd/templ@$(TEMPL_VERSION)
	go install golang.org/x/tools/cmd/goimports@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VERSION)

# If you truly want to bump libraries in go.mod, do it explicitly:
## deps-update: Update library dependencies in go.mod
deps-update:
	go get -u ./...
	go mod tidy

# --- Generation ---------------------------------------------------------------

## gen: Run code generators (sqlc, templ)
gen:
	"$(GOBIN)/sqlc" generate
	"$(GOBIN)/templ" generate

# --- Build & Run --------------------------------------------------------------

## build: Build binary into ./bin
build: gen
	$(MKDIR) $(BUILD_DIR)
# 	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	go build -trimpath -buildvcs=false -ldflags "-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

## run: Run from source
run: gen
	go run ./$(CMD_DIR)

## debug: Run with delve debugger
debug: gen
	dlv debug ./$(CMD_DIR) --headless --listen=:2345 --api-version=2 --accept-multiclient

# --- Test ---------------------------------------------------------------------

## test: Run unit tests
ifeq ($(OS),Windows_NT)
test:
	go test ./...
else
test:
	go test -race ./...
endif

## test-coverage: Run tests with coverage report
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage HTML: coverage.html"

# --- Lint & Format ------------------------------------------------------------

## lint: Full lint (vet, staticcheck, golangci-lint)
lint:
	go vet ./...
	staticcheck ./...
	golangci-lint run

## lint-fast: Quick lint without golangci-lint
lint-fast:
	go vet ./...
	staticcheck ./...

## fmt: Write format & imports
fmt:
	go fmt ./...
	goimports -w .

## fmt-check: Fail if code needs formatting (CI-friendly)
fmt-check:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "$$diff = & gofmt -l .; if ($$diff) { Write-Host 'Files need gofmt:'; $$diff; exit 1 }"
	@powershell -NoProfile -Command "$$diffi = & goimports -l .; if ($$diffi) { Write-Host 'Files need goimports:'; $$diffi; exit 1 }"
else
	@diff=$$(gofmt -l .); if [ -n "$$diff" ]; then echo "Files need gofmt:"; echo "$$diff"; exit 1; fi
	@diffi=$$(goimports -l .); if [ -n "$$diffi" ]; then echo "Files need goimports:"; echo "$$diffi"; exit 1; fi
endif

## tidy-check: Ensure go.mod/go.sum are tidy (CI-friendly)
tidy-check:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "$$m1=(Get-FileHash go.mod).Hash; $$s1=(Get-FileHash go.sum).Hash; go mod tidy; $$m2=(Get-FileHash go.mod).Hash; $$s2=(Get-FileHash go.sum).Hash; if ($$m1 -ne $$m2) { Write-Error 'go.mod not tidy'; exit 1 }; if ($$s1 -ne $$s2) { Write-Error 'go.sum not tidy'; exit 1 }"
else
	@cp go.mod go.mod.bak && cp go.sum go.sum.bak; \
	go mod tidy; \
	diff -q go.mod go.mod.bak >/dev/null || (echo "go.mod not tidy"; rm -f go.mod.bak go.sum.bak; exit 1); \
	diff -q go.sum go.sum.bak >/dev/null || (echo "go.sum not tidy"; rm -f go.mod.bak go.sum.bak; exit 1); \
	rm -f go.mod.bak go.sum.bak
endif

## generate-check: Ensure generated code is up to date (CI-friendly)
generate-check:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "$$before = & git status --porcelain; $(MAKE) --no-print-directory gen | Out-Null; $$after = & git status --porcelain; if ($$before -ne $$after) { Write-Host 'Generated code is out of date. Run make gen and commit changes.'; & git diff -- .; exit 1 }"
else
	@status_before=$$(git status --porcelain); \
	$(MAKE) --no-print-directory gen; \
	status_after=$$(git status --porcelain); \
	if [ "$$status_before" != "$$status_after" ]; then \
		echo "Generated code is out of date. Run 'make gen' and commit changes."; \
		git diff -- .; \
		exit 1; \
	fi
endif

# --- Install / Env / DB -------------------------------------------------------

## install: Build & install to GOBIN
install: gen
	go install ./$(CMD_DIR)

## check-env: Verify .env exists (fails if missing)
check-env:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "if (!(Test-Path '$(CMD_DIR)\.env')) { Write-Error '.env missing in $(CMD_DIR) (copy .env.example)'; exit 1 } else { Write-Host '.env found' }"
else
	@test -f $(CMD_DIR)/.env && echo ".env found" || (echo ".env missing in $(CMD_DIR) (copy .env.example)"; exit 1)
endif

## db-init: Initialize database using the built binary
db-init: build check-env
	cd $(CMD_DIR) && ../$(BUILD_DIR)/$(BINARY_NAME) -init-db

# --- Dev / Assets -------------------------------------------------------------

## dev: Development mode (you can swap to a watcher here)
dev: run
# Example watcher (optional):
# dev:
#     $(GOBIN)/templ generate --watch &
#     reflex -r '\.go$|\.templ$|\.sql$' -- sh -c 'go run ./$(CMD_DIR)'

## assets: Describe frontend asset layout
assets:
	@echo "Frontend Asset Organization:"
	@echo "├── web/assets/"
	@echo "│   └── css/components.css (custom CSS components)"
	@echo ""
	@echo "Assets served from /assets/; Tailwind via CDN; HTMX only"

# --- Clean --------------------------------------------------------------------

## clean: Remove build artifacts and generated files
clean:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -Command "if (Test-Path '$(BUILD_DIR)') { Remove-Item -Recurse -Force '$(BUILD_DIR)' }"
	@powershell -NoProfile -Command "if (Test-Path 'internal\db') { Get-ChildItem -Path 'internal\db' -Filter '*_gen.go' | Remove-Item -Force }"
	@powershell -NoProfile -Command "if (Test-Path 'web') { Get-ChildItem -Path 'web' -Filter '*_templ.go' -Recurse | Remove-Item -Force }"
else
	$(RM) $(BUILD_DIR)
	find internal/db -name "*_gen.go" -delete 2>/dev/null || true
	find web -name "*_templ.go" -delete 2>/dev/null || true
endif

## clean-all: Clean everything including module cache
clean-all: clean
	go clean -modcache

# --- Meta ---------------------------------------------------------------------

## verify: Build + read-only checks (CI-ready)
verify: fmt-check tidy-check generate-check lint-fast build test
	@echo "✓ Build + checks passed"

## help: Show this help
help:
	@echo "SharePoint Audit - Available commands:"
	@grep '## ' $(MAKEFILE_LIST) | sed 's/## /  /'
