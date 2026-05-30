.PHONY: help build build-pdfium test test-race vet cover install clean docker

BIN     := distill
CMD     := ./cmd/distill
PREFIX  ?= $(HOME)/.local

help: ## Show this help.
	@awk 'BEGIN{FS=":.*##"; printf "Targets:\n"} /^[a-zA-Z_-]+:.*##/ {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Default build (pure-Go, ~9 MB).
	go build -o $(BIN) $(CMD)

build-pdfium: ## Full-feature build with PDFium WebAssembly engine (~23 MB).
	go build -tags pdfium -o $(BIN) $(CMD)

test: ## Run all tests.
	go test ./... -count=1

test-race: ## Run all tests with the race detector.
	go test ./... -race -count=1

vet: ## Run go vet (default and pdfium tags).
	go vet ./...
	go vet -tags pdfium ./...

cover: ## Print package coverage.
	go test ./internal/converters/tests/ -coverpkg=./internal/converters/src -count=1
	go test ./internal/convert -cover -count=1
	go test ./internal/app -cover -count=1
	go test ./cmd/distill -cover -count=1

install: build-pdfium ## Build with -tags pdfium and install to $(PREFIX)/bin (default: ~/.local/bin).
	install -d $(PREFIX)/bin
	install -m 0755 $(BIN) $(PREFIX)/bin/$(BIN)
	@echo "installed: $(PREFIX)/bin/$(BIN)"

clean: ## Remove build artifacts.
	rm -f $(BIN)

docker: ## Build a distroless container image (-tags pdfium).
	docker build -t distill:latest .
