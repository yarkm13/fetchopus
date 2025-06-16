# Variables
APP_NAME := fetchopus
DIST_DIR := dist
SRC_DIR := .

# Default target
all: build

# Build the binary
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(DIST_DIR)
	go build -o $(DIST_DIR)/$(APP_NAME) $(SRC_DIR)
	chmod +x $(DIST_DIR)/$(APP_NAME)

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	@rm -rf $(DIST_DIR)

.PHONY: all build clean
