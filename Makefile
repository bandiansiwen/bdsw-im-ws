.PHONY: test test-unit test-integration test-bench coverage

test:
	@echo "Running all tests..."
	@go test -v ./... -cover

test-unit:
	@echo "Running unit tests..."
	@go test -v ./internal/... ./pkg/... -cover

test-integration:
	@echo "Running integration tests..."
	@go test -v ./integration/... -tags=integration

test-bench:
	@echo "Running benchmark tests..."
	@go test -bench=. -benchmem ./...

coverage:
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

test-short:
	@echo "Running short tests..."
	@go test -short -v ./... -cover

clean:
	@rm -f coverage.out coverage.html

help:
	@echo "Available targets:"
	@echo "  test          - Run all tests"
	@echo "  test-unit     - Run unit tests only"
	@echo "  test-integration - Run integration tests"
	@echo "  test-bench    - Run benchmark tests"
	@echo "  coverage      - Generate coverage report"
	@echo "  test-short    - Run tests in short mode"
	@echo "  clean         - Clean generated files"