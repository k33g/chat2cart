.PHONY: run build clean setup help

# Default target
help:
	@echo "Available commands:"
	@echo "  setup    - Install dependencies"
	@echo "  run      - Start the development server"
	@echo "  build    - Build the application"
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

# Build the application
build:
	@echo "Building chat2cart..."
	go build -o bin/chat2cart main.go
	@echo "Build complete! Binary created at bin/chat2cart"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf tmp/
	go clean
	@echo "Clean complete!"
