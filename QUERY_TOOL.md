# Query Tool - Test Your Advisory Database

A simple CLI tool to query the advisory database and test semantic search.

## Quick Start

### 1. Build the Query Tool

```bash
go build -o bin/query-advisory-db ./cmd/query
```

### 2. Query the Database

```bash
./bin/query-advisory-db --query "how to handle errors in Go"
```

## Usage

```bash
./bin/query-advisory-db [flags]
```

### Flags

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--query` | - | *(required)* | Query text |
| `--db` | `DB_PATH` | `advisory-ollama.db` | Path to database |
| `--n` | - | `5` | Number of results |
| `--backend` | `EMBEDDING_BACKEND` | `ollama` | Backend: `azure` or `ollama` |
| `--model` | `EMBEDDING_MODEL` | `nomic-embed-text` | Embedding model |
| `--ollama-url` | `OLLAMA_BASE_URL` | `http://localhost:11434/api` | Ollama URL |
| `--azure-endpoint` | `AZURE_OPENAI_ENDPOINT` | - | Azure endpoint |
| `--azure-key` | `AZURE_OPENAI_KEY` | - | Azure API key |

## Examples

### Basic Query

```bash
./bin/query-advisory-db --query "error wrapping"
```

### More Results

```bash
./bin/query-advisory-db --query "naming conventions" --n 10
```

### Query Azure Database

```bash
./bin/query-advisory-db \
  --db advisory.db \
  --backend azure \
  --azure-endpoint $AZURE_OPENAI_ENDPOINT \
  --azure-key $AZURE_OPENAI_KEY \
  --model text-embedding-3-small \
  --query "testing best practices"
```

### Use Environment Variables

```bash
export DB_PATH=advisory-ollama.db
export EMBEDDING_BACKEND=ollama
export EMBEDDING_MODEL=nomic-embed-text

./bin/query-advisory-db --query "package structure"
```

## Example Output

```
Generating embedding for query: "error wrapping with fmt.Errorf"
Loading database from: advisory-ollama.db
Querying collection with 80 documents...

Found 2 results:
================================================================================

Result 1 (Similarity: 0.6939)
URL: https://google.github.io/styleguide/go#error-wrapping
Source: resources/Go-Coding-Style-Guide.md
Section: Error Handling > Error Wrapping

Content Preview:
## Error Handling

Error Wrapping
Use `%w` verb with `fmt.Errorf` to wrap errors for proper error chain handling.
...
```

## Sample Queries to Try

### Error Handling
```bash
./bin/query-advisory-db --query "how to handle errors"
./bin/query-advisory-db --query "error wrapping with %w"
./bin/query-advisory-db --query "checking for specific errors"
```

### Naming
```bash
./bin/query-advisory-db --query "variable naming conventions"
./bin/query-advisory-db --query "how to name packages"
./bin/query-advisory-db --query "interface naming"
```

### Testing
```bash
./bin/query-advisory-db --query "writing unit tests"
./bin/query-advisory-db --query "table driven tests"
./bin/query-advisory-db --query "test naming"
```

### Formatting
```bash
./bin/query-advisory-db --query "code formatting"
./bin/query-advisory-db --query "line length"
./bin/query-advisory-db --query "gofmt usage"
```

### Concurrency
```bash
./bin/query-advisory-db --query "goroutines"
./bin/query-advisory-db --query "channels"
./bin/query-advisory-db --query "mutex"
```

## How It Works

1. **Generate Query Embedding**: Uses the same embedder as the builder to convert your query text into a vector
2. **Load Database**: Opens the chromem-go database file
3. **Semantic Search**: Uses cosine similarity to find the most relevant chunks
4. **Display Results**: Shows top N results with similarity scores and metadata

## Understanding Results

### Similarity Score

- **> 0.7**: Highly relevant, very similar content
- **0.6 - 0.7**: Relevant, good match
- **0.5 - 0.6**: Somewhat relevant, related topic
- **< 0.5**: Less relevant, may be tangential

### Result Fields

- **URL**: Deep link to exact section in documentation
- **Source**: Original markdown file path
- **Section**: H2 > H3 heading hierarchy
- **Content Preview**: First 400 characters of the chunk

## Tips for Better Results

1. **Be Specific**: "error wrapping with fmt.Errorf" > "errors"
2. **Use Technical Terms**: The database contains technical documentation
3. **Try Variations**: If first query doesn't work, rephrase
4. **Adjust Result Count**: Use `--n` to get more or fewer results
5. **Read Full Context**: URLs link to complete documentation

## Troubleshooting

### "collection 'advisories' not found"

The database doesn't exist or wasn't created with the builder:

```bash
./demo-ollama.sh
```

### "failed to generate query embedding"

Check that Ollama is running:

```bash
ollama list
ollama pull nomic-embed-text
```

### Results Don't Make Sense

Make sure you're using the **same embedding model** that was used to build the database:

```bash
# If built with nomic-embed-text:
./bin/query-advisory-db --model nomic-embed-text --query "..."

# If built with Azure text-embedding-3-small:
./bin/query-advisory-db \
  --backend azure \
  --model text-embedding-3-small \
  --query "..."
```

## Integration with MCP (Phase 2)

This query tool demonstrates the core functionality that will be exposed via MCP in Phase 2:
- Generate query embeddings
- Search vector database
- Return top N results with metadata
- Format results for consumption

The MCP server will wrap this functionality and expose it to GitHub Copilot and other MCP clients.

## Performance

- **Query Embedding**: ~100-300ms (Ollama local)
- **Database Search**: ~10-50ms (depends on collection size)
- **Total**: Usually under 500ms

## Next Steps

Once you've tested queries and verified results are good:
1. Add more documents to your manifest
2. Re-build the database with incremental updates
3. Test that new content is findable
4. Wait for Phase 2 MCP server for GitHub Copilot integration
