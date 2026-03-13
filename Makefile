# Makefile for route-switch

# Build the application
build:
	go build -o route-switch

# Run all tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -cover ./...

# Run the example script
example:
	./example.sh

# Clean build artifacts
clean:
	rm -f route-switch

# Install dependencies
deps:
	go mod tidy

# Build and run the application
run: build
	./route-switch

.PHONY: build test test-coverage example clean deps run