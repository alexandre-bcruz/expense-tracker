BINARY := expense-tracker
CMD    := ./cmd/api
ADDR   ?= :8080

.DEFAULT_GOAL := help
.PHONY: help run build test test-race cover vet fmt fmt-check tidy clean

help: ## List available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-11s\033[0m %s\n", $$1, $$2}'

run: ## Run the API server (override the port with ADDR=:9090)
	ADDR=$(ADDR) go run $(CMD)

build: ## Build the API binary into ./bin
	go build -o bin/$(BINARY) $(CMD)

test: ## Run all tests
	go test ./...

test-race: ## Run all tests with the race detector
	go test -race ./...

cover: ## Report test coverage
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

vet: ## Run go vet
	go vet ./...

fmt: ## Format the code
	gofmt -w .

fmt-check: ## Fail if the code is not gofmt-clean
	@out="$$(gofmt -l .)"; test -z "$$out" || { echo "not formatted:"; echo "$$out"; exit 1; }

tidy: ## Tidy go.mod
	go mod tidy

clean: ## Remove build and coverage artifacts
	rm -rf bin coverage.out
