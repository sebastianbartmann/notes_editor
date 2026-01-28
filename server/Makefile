.PHONY: build build-web test test-coverage lint run clean

# Build the server binary
build:
	go build -o bin/server ./cmd/server

# Build web UI and copy to static directory
build-web:
	cd ../clients/web && npm install && npm run build
	rm -rf static
	cp -r ../clients/web/dist static

# Build everything (server + web UI)
build-all: build-web build

# Run all tests
test:
	go test -v ./...

# Run tests with coverage report
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

# Run the server
run:
	go run ./cmd/server

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html
