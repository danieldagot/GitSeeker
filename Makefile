# Define the output binary name
BINARY_NAME=myapp

# Define the source files
SOURCES=main.go

# Define the platforms and their respective build targets
PLATFORMS=darwin/amd64 linux/amd64 windows/amd64 linux/arm linux/arm64

# Define the build directory
BUILD_DIR=build

# Default target: build for the current platform
default: build

# Build for the current platform
build: $(SOURCES)
	go build -o $(BINARY_NAME)

# Build for all specified platforms
crossbuild: $(SOURCES)
	mkdir -p $(BUILD_DIR)
	$(foreach platform, $(PLATFORMS), \
		GOOS=$(word 1, $(subst /, , $(platform))) \
		GOARCH=$(word 2, $(subst /, , $(platform))) \
		go build -o $(BUILD_DIR)/$(BINARY_NAME)-$(word 1, $(subst /, , $(platform)))-$(word 2, $(subst /, , $(platform))) $(SOURCES);)

# Clean the build directory
clean:
	rm -rf $(BUILD_DIR)

.PHONY: build crossbuild clean
