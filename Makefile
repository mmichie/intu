# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOGENERATE=$(GOCMD) generate
GOFMT=$(GOCMD) fmt

# Binary name
BINARY_NAME=intu

# Build directory
BUILD_DIR=bin

# Main package path
MAIN_PACKAGE=./cmd/intu

# All packages
ALL_PACKAGES=./...

all: deps fmt test build

build: generate
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

test:
	$(GOTEST) $(ALL_PACKAGES)

test-verbose:
	$(GOTEST) -v $(ALL_PACKAGES)

clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

deps:
	$(GOMOD) download
	$(GOMOD) tidy

generate:
	$(GOGENERATE) $(ALL_PACKAGES)

fmt:
	$(GOFMT) $(ALL_PACKAGES)

# Cross compilation
build-linux: generate
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)

build-linux-arm64: generate
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PACKAGE)

build-windows: generate
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)

build-windows-arm64: generate
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe $(MAIN_PACKAGE)

build-darwin: generate
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)

build-darwin-arm64: generate
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)

# Build for all platforms
build-all: build-linux build-linux-arm64 build-windows build-windows-arm64 build-darwin build-darwin-arm64

.PHONY: all build test test-verbose clean run deps generate fmt build-linux build-linux-arm64 build-windows build-windows-arm64 build-darwin build-darwin-arm64 build-all
