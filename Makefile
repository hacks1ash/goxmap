.PHONY: build install clean test test-cover test-bench lint vet fmt generate help

MODULE   := github.com/hacks1ash/goxmap
BIN_DIR  := bin
BINARY   := $(BIN_DIR)/goxmap
COVERAGE := $(BIN_DIR)/coverage.out

# Build goxmap binary into bin/
build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY) .

# Install goxmap to $GOPATH/bin
install:
	go install .

# Remove build artifacts
clean:
	rm -rf $(BIN_DIR)
	go clean -testcache

# Run all tests
test:
	go test ./...

# Run tests with coverage report
test-cover:
	@mkdir -p $(BIN_DIR)
	go test -coverprofile=$(COVERAGE) ./internal/...
	go tool cover -func=$(COVERAGE) | tail -1
	@echo "\nPer-package:"
	@go tool cover -func=$(COVERAGE) | grep "total" || true

# Run tests with HTML coverage report
test-cover-html: test-cover
	go tool cover -html=$(COVERAGE) -o $(BIN_DIR)/coverage.html
	@echo "Coverage report: $(BIN_DIR)/coverage.html"

# Run benchmarks (in separate module)
test-bench:
	cd benchmarks && go test -bench=. -benchmem -benchtime=3s

# Run go vet
vet:
	go vet ./...

# Run golangci-lint (if installed)
lint:
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... || \
		(echo "golangci-lint not installed. Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)

# Format all Go files
fmt:
	gofmt -s -w .

# Run go generate
generate:
	go generate ./...

# Run all checks (vet + lint + test)
check: vet lint test

# Show help
help:
	@echo "goxmap Makefile targets:"
	@echo ""
	@echo "  build          Build goxmap binary to bin/"
	@echo "  install        Install goxmap to GOPATH/bin"
	@echo "  clean          Remove build artifacts and test cache"
	@echo "  test           Run all tests"
	@echo "  test-cover     Run tests with coverage (target: 92%+)"
	@echo "  test-cover-html  Generate HTML coverage report"
	@echo "  test-bench     Run benchmarks (separate module)"
	@echo "  vet            Run go vet"
	@echo "  lint           Run golangci-lint"
	@echo "  fmt            Format Go source files"
	@echo "  generate       Run go generate"
	@echo "  check          Run vet + lint + test"
	@echo "  help           Show this help"
