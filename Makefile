.PHONY: run build test clean setup dev help

# Default target
help:
	@echo "Available commands:"
	@echo "  setup    - Install dependencies"
	@echo "  run      - Start the development server"
	@echo "  dev      - Development mode with hot reload (requires air)"
	@echo "  build    - Build the application"
	@echo "  test     - Run tests"
	@echo "  clean    - Clean build artifacts"
	@echo "  help     - Show this help message"

# Install dependencies
setup:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download
	@echo "Dependencies installed successfully!"
	@echo ""
	@echo "To get started:"
	@echo "1. Set your OpenAI API key: export OPENAI_API_KEY=your_api_key_here"
	@echo "2. Run the application: make run"

# Run the application
run:
	@echo "Starting chat2cart server..."
	go run main.go

# Development mode with hot reload (requires air)
dev:
	@if command -v air > /dev/null; then \
		echo "Starting development server with hot reload..."; \
		air; \
	else \
		echo "Air not found. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to regular run..."; \
		make run; \
	fi

# Build the application
build:
	@echo "Building chat2cart..."
	go build -o bin/chat2cart main.go
	@echo "Build complete! Binary created at bin/chat2cart"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf tmp/
	go clean
	@echo "Clean complete!"

# Install air for hot reload (optional)
install-air:
	@echo "Installing air for hot reload..."
	go install github.com/cosmtrek/air@latest
	@echo "Air installed! You can now use 'make dev' for hot reload."
