# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOGENERATE=$(GOCMD) generate

# Binary name
BINARY_NAME=intu

# Build directory
BUILD_DIR=bin

# Main package path
MAIN_PACKAGE=.

# All packages
ALL_PACKAGES=./...

all: test build

build:
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

generate:
	$(GOGENERATE) $(ALL_PACKAGES)

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)

# Build for all platforms
build-all: build-linux build-windows build-darwin

.PHONY: all build test test-verbose clean run deps generate build-linux build-windows build-darwin build-all
