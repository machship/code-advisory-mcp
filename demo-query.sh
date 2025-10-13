#!/bin/bash
# Demo script for querying the advisory database

set -e

echo "=== Code Advisory Database Query Demo ==="
echo

# Check if database exists
if [ ! -d "advisory-ollama.db" ]; then
    echo "Database not found. Building it first..."
    ./demo-ollama.sh
    echo
fi

# Build the query tool if needed
if [ ! -f "bin/query-advisory-db" ]; then
    echo "Building query tool..."
    go build -o bin/query-advisory-db ./cmd/query
    echo
fi

echo "Running sample queries..."
echo

# Sample queries
queries=(
    "error wrapping with fmt.Errorf"
    "naming conventions for Go packages"
    "how to write unit tests"
    "using goroutines safely"
    "interface design best practices"
)

for i in "${!queries[@]}"; do
    query="${queries[$i]}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Query $((i+1)): \"$query\""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo
    
    ./bin/query-advisory-db --query "$query" --n 2
    
    if [ $i -lt $((${#queries[@]} - 1)) ]; then
        echo
        echo
        read -p "Press Enter to continue to next query..." -r
        echo
    fi
done

echo
echo "=== Demo Complete ==="
echo
echo "Try your own queries:"
echo "  ./bin/query-advisory-db --query \"your question here\""
echo
echo "See QUERY_TOOL.md for more examples and options."
