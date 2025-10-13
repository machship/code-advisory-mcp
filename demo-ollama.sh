#!/bin/bash
# Demo script for building advisory database with Ollama

set -e

echo "=== Code Advisory Database Builder - Ollama Demo ==="
echo

# Check if Ollama is available
if ! command -v ollama &> /dev/null; then
    echo "Error: Ollama is not installed or not in PATH"
    echo "Please install Ollama from https://ollama.ai"
    exit 1
fi

# Check if nomic-embed-text model is available
if ! ollama list | grep -q "nomic-embed-text"; then
    echo "Pulling nomic-embed-text model (this may take a few minutes)..."
    ollama pull nomic-embed-text
    echo
fi

# Build the tool if needed
if [ ! -f "bin/build-advisory-db" ]; then
    echo "Building build-advisory-db tool..."
    go build -o bin/build-advisory-db ./cmd/builder
    echo
fi

# Run the builder
echo "Building advisory database with Ollama..."
echo

./bin/build-advisory-db \
    --backend ollama \
    --model nomic-embed-text \
    --manifest manifest.json \
    --output advisory-ollama.db

echo
echo "=== Demo Complete ==="
echo
echo "Database created: advisory-ollama.db"
echo
echo "Try running again to see idempotency in action:"
echo "  ./bin/build-advisory-db --backend ollama --model nomic-embed-text --manifest manifest.json --output advisory-ollama.db"
