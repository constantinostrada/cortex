.PHONY: build install clean test

BINARY_NAME=cortex
VERSION=0.1.0
BUILD_FLAGS=-tags "fts5"
CGO_FLAGS=CGO_ENABLED=1 CGO_CFLAGS="-DSQLITE_ENABLE_FTS5"

build:
	$(CGO_FLAGS) go build $(BUILD_FLAGS) -o bin/$(BINARY_NAME) ./cmd/cortex

install:
	$(CGO_FLAGS) go install $(BUILD_FLAGS) ./cmd/cortex

clean:
	rm -rf bin/
	rm -f $(BINARY_NAME)

test:
	go test -v ./...

# Build for multiple platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/cortex
	GOOS=darwin GOARCH=arm64 go build -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/cortex
	GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/cortex
	GOOS=windows GOARCH=amd64 go build -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/cortex

# Run the CLI
run:
	go run ./cmd/cortex $(ARGS)

# Initialize a test environment
dev-init:
	go run ./cmd/cortex init

# Download dependencies
deps:
	go mod download
	go mod tidy
