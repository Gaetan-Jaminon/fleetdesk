.PHONY: test test-report build lint check clean

test:           ## Run unit tests
	go test ./... -race -v

test-report:    ## Run unit tests with coverage report
	go test ./... -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

build:          ## Build binary
	go build -o fleetdesk .

lint:           ## Run linter
	golangci-lint run

check:          ## Run all checks before PR (build + test + lint)
	$(MAKE) build
	$(MAKE) test
	$(MAKE) lint

clean:          ## Remove build artifacts
	rm -f fleetdesk coverage.out coverage.html
