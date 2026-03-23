APP_NAME := flarness
VERSION  := $(shell git describe --tags --always 2>/dev/null || echo "dev")

.PHONY: build build-all install clean test

# Build for current platform
build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/$(APP_NAME) .

# Cross-compile for all platforms
build-all:
	GOOS=darwin  GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o bin/$(APP_NAME)-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o bin/$(APP_NAME)-darwin-arm64 .
	GOOS=linux   GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o bin/$(APP_NAME)-linux-amd64 .
	GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o bin/$(APP_NAME)-windows-amd64.exe .

# Install to /usr/local/bin
install: build
	cp bin/$(APP_NAME) /usr/local/bin/

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/
