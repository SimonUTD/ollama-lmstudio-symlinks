#!/bin/bash

# Build script for ollama-symlinks
# This script reads the version from VERSION file and builds the binary

set -e

# Use a local Go build cache to avoid permission issues in restricted environments.
GOCACHE_DIR="${GOCACHE:-"$(pwd)/.gocache"}"
mkdir -p "$GOCACHE_DIR"
export GOCACHE="$GOCACHE_DIR"

# Read version from VERSION file
VERSION=$(cat VERSION)
echo "Building version: $VERSION"

# Build the binary
echo "Building binary..."
go build -ldflags="-X 'main.Version=$VERSION'" -o ollama-symlinks ./cmd/ollama-symlinks

echo "Build complete: ollama-symlinks version $VERSION"
