.PHONY: build build-windows build-linux run test clean fmt tidy help

# Default target
all: build

# Build for current platform
build:
	@echo "Building for current platform..."
	GO111MODULE=on go build -o bin/pr-tracker ./cmd

# Build for Windows
build-windows:
	@echo "Building for Windows..."
	GO111MODULE=on GOOS=windows GOARCH=amd64 go build -o bin/pr-tracker-windows.exe ./cmd

# Build for Linux
build-linux:
	@echo "Building for Linux..."
	GO111MODULE=on GOOS=linux GOARCH=amd64 go build -o bin/pr-tracker-linux ./cmd

# Build for multiple platforms
build-all: build-windows build-linux build

# Run the application
run:
	@echo "Running application..."
	GO111MODULE=on go run ./cmd

# Run tests
test:
	@echo "Running tests..."
	GO111MODULE=on go test -v ./pkg/... ./internal/... ./cmd/...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	GO111MODULE=on go test -v -coverprofile=coverage.out ./pkg/... ./internal/... ./cmd/...
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report saved to coverage.html"

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	GO111MODULE=on go test -race -v ./pkg/... ./internal/... ./cmd/...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf tmp/
	rm -f last_notification.txt

# Format code
fmt:
	@echo "Formatting code..."
	GO111MODULE=on go fmt ./...
	GO111MODULE=on goimports -w .

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	GO111MODULE=on go mod tidy

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install golang.org/x/tools/cmd/goimports@latest

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build for current platform"
	@echo "  build-windows - Build for Windows"
	@echo "  build-linux   - Build for Linux"
	@echo "  build-all     - Build for all platforms"
	@echo "  run           - Run the application"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-race     - Run tests with race detection"
	@echo "  clean         - Clean build artifacts"
	@echo "  fmt           - Format code"
	@echo "  tidy          - Tidy dependencies"
	@echo "  install-tools - Install development tools"
	@echo "  help          - Show this help" 