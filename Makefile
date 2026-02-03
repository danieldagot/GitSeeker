# Define the output binary name
BINARY_NAME=gs

# Installation directories
PREFIX?=$(HOME)/.local
BINDIR?=$(PREFIX)/bin
ZSH_COMPLETION_DIR?=$(PREFIX)/share/zsh/site-functions

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

# Install binary and zsh completion
install: build
	mkdir -p $(BINDIR)
	install -m 755 $(BINARY_NAME) $(BINDIR)/$(BINARY_NAME)
	if [ -f completions/_gs ]; then \
		mkdir -p $(ZSH_COMPLETION_DIR); \
		install -m 644 completions/_gs $(ZSH_COMPLETION_DIR)/_gs; \
	fi

# Uninstall binary and zsh completion
uninstall:
	rm -f $(BINDIR)/$(BINARY_NAME)
	rm -f $(ZSH_COMPLETION_DIR)/_gs

# Clean the build directory
clean:
	rm -rf $(BUILD_DIR)

.PHONY: build crossbuild install uninstall clean
