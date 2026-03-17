#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default installation directory
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="route-switch"

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.24+ from https://golang.org/dl/"
        exit 1
    fi

    GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+')
    REQUIRED_VERSION="1.24"

    if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
        print_error "Go version $GO_VERSION is installed, but $REQUIRED_VERSION+ is required"
        exit 1
    fi

    print_status "Go version $GO_VERSION detected"
}

# Download dependencies
download_deps() {
    print_status "Downloading dependencies..."
    go mod download
    go mod tidy
    print_status "Dependencies downloaded successfully"
}

# Build the binary
build_binary() {
    print_status "Building $BINARY_NAME..."
    go build -o "$BINARY_NAME"
    print_status "Build completed successfully"
}

# Install the binary
install_binary() {
    print_status "Installing $BINARY_NAME to $INSTALL_DIR..."

    if [ ! -d "$INSTALL_DIR" ]; then
        print_warning "Directory $INSTALL_DIR does not exist. Creating it..."
        sudo mkdir -p "$INSTALL_DIR"
    fi

    if [ -w "$INSTALL_DIR" ]; then
        cp "$BINARY_NAME" "$INSTALL_DIR/"
    else
        print_warning "Elevated permissions required to install to $INSTALL_DIR"
        sudo cp "$BINARY_NAME" "$INSTALL_DIR/"
    fi

    print_status "$BINARY_NAME installed successfully to $INSTALL_DIR/$BINARY_NAME"
}

# Show usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --build-only    Only build the binary, do not install"
    echo "  --install-dir   Specify installation directory (default: /usr/local/bin)"
    echo "  --help          Show this help message"
    echo ""
    echo "Environment variables:"
    echo "  INSTALL_DIR     Installation directory (default: /usr/local/bin)"
    echo ""
    echo "Examples:"
    echo "  $0                           # Build and install to /usr/local/bin"
    echo "  $0 --build-only              # Only build, don't install"
    echo "  $0 --install-dir ~/bin       # Install to ~/bin"
    echo "  INSTALL_DIR=~/.local/bin $0  # Install to ~/.local/bin"
}

# Parse arguments
BUILD_ONLY=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --build-only)
            BUILD_ONLY=true
            shift
            ;;
        --install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Main execution
main() {
    print_status "Starting $BINARY_NAME installation..."

    # Change to script directory
    cd "$(dirname "$0")"

    check_go
    download_deps
    build_binary

    if [ "$BUILD_ONLY" = false ]; then
        install_binary
        echo ""
        print_status "Installation complete! Run '$BINARY_NAME --help' to get started."
    else
        echo ""
        print_status "Build complete! Binary available at ./$BINARY_NAME"
    fi
}

main
