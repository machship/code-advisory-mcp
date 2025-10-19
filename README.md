# Code Advisory MCP Server

A Model Context Protocol (MCP) server that provides instant access to coding standards, style guides, and best practices through semantic search. GitHub Copilot and other MCP clients can query your company's coding standards and get relevant guidance with deep-linked URLs.

## Overview

This project provides a complete solution for making coding standards searchable via MCP:

1. **Database Builder** - Chunks markdown documents and generates embeddings
2. **MCP Server** - Serves advisories via the `fetch_advisory_for_task` tool
3. **Embedded Database** - Self-contained binary with built-in Go standards

The server uses vector similarity search to find the most relevant coding guidelines based on natural language task descriptions.

## Features

- ✅ **Embedded Database**: Self-contained 14MB binary with 126 Go coding standard chunks
- ✅ **Multiple Backends**: Azure OpenAI or local Ollama embeddings
- ✅ **Contextual Chunking**: Smart markdown parsing preserving document structure
- ✅ **Deep-Linked Results**: Precise URL fragments to exact documentation sections
- ✅ **No Cloud Required**: Use Ollama for completely local, private operation
- ✅ **Production Ready**: All phases complete and tested

## Installation

### Prerequisites

**Required:**
- Go 1.23 or later
- Git

**Choose one embedding backend:**
- **Ollama** (recommended for local development) - No API keys needed
- **Azure OpenAI** (recommended for production) - Requires Azure account

### 1. Install Ollama (Local Backend)

Ollama provides high-quality embeddings locally without any API keys or cloud services.

**macOS:**
```bash
brew install ollama
```

**Linux:**
```bash
curl -fsSL https://ollama.ai/install.sh | sh
```

**Windows:**
Download from [ollama.ai](https://ollama.ai)

**Start Ollama and pull the embedding model:**
```bash
# Start Ollama service (if not already running)
ollama serve

# Pull the embedding model (274MB, 768 dimensions)
ollama pull nomic-embed-text
```

### 2. Install the MCP Server

The server includes an embedded database with 126 Go coding standard chunks, so it works immediately after installation without any additional setup.

**Option A: Install from source**
```bash
# Clone the repository
git clone https://github.com/machship/code-advisory-mcp.git
cd code-advisory-mcp

# Install dependencies
go mod download

# Build the MCP server (includes embedded database)
go build -o bin/code-advisory-mcp ./cmd/server

# Install to your PATH (optional)
sudo cp bin/code-advisory-mcp /usr/local/bin/
```

**Option B: Install using go install**
```bash
go install github.com/machship/code-advisory-mcp/cmd/server@latest
```

The server binary is self-contained and includes:
- 126 chunks from Go documentation (770KB)
- Comprehensive usage instructions (19KB)
- Total binary size: ~14MB

### 3. Verify Installation

```bash
# Check the server binary
./bin/code-advisory-mcp --help

# View usage instructions
./bin/code-advisory-mcp --print-instructions

# Test the embedded database (optional)
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"0.1.0","capabilities":{}}}' | ./bin/code-advisory-mcp
```

You should see the MCP server help with available flags and the initialize response.

## Quick Start

### Test the Server Locally

The server includes an embedded database with 126 chunks from official Go documentation.

```bash
# Start the MCP server with embedded database
./bin/code-advisory-mcp

# Or specify a custom database
./bin/code-advisory-mcp --db /path/to/custom.db
```

The server communicates via JSON-RPC over stdin/stdout (MCP protocol).

### Configure with MCP Clients

The server works with any MCP-compatible client. Configuration depends on your client:

#### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS):

```json
{
  "mcpServers": {
    "code-advisory": {
      "command": "/usr/local/bin/code-advisory-mcp",
      "args": [],
      "env": {
        "EMBEDDING_BACKEND": "ollama",
        "EMBEDDING_MODEL": "nomic-embed-text"
      }
    }
  }
}
```

#### VS Code with Cline Extension

Add to Cline's MCP settings:

```json
{
  "code-advisory": {
    "command": "/usr/local/bin/code-advisory-mcp",
    "args": [],
    "env": {
      "EMBEDDING_BACKEND": "ollama",
      "EMBEDDING_MODEL": "nomic-embed-text"
    }
  }
}
```

#### GitHub Copilot (via CLI)

GitHub Copilot CLI doesn't currently support custom MCP servers directly. You can:

1. Use the query tool standalone:
   ```bash
   ./bin/query-advisory-db --query "your question"
   ```

2. Or integrate via a wrapper script that Copilot can call

#### With Azure OpenAI

Replace the `env` section in any configuration:

```json
"env": {
  "EMBEDDING_BACKEND": "azure",
  "AZURE_OPENAI_ENDPOINT": "https://your-resource.openai.azure.com",
  "AZURE_OPENAI_KEY": "your-api-key",
  "EMBEDDING_MODEL": "text-embedding-3-small"
}
```

Restart your MCP client after configuration changes.


## Usage

### Using the MCP Tool

Once configured with an MCP client, you can ask for coding guidance:

**Example prompts:**
- "How should I handle errors in Go?"
- "What's the best way to structure table-driven tests?"
- "Show me the proper way to use interfaces in Go"
- "How do I implement graceful shutdown for an HTTP server?"

Copilot will use the `fetch_advisory_for_task` tool to retrieve relevant guidelines from your standards.

### Tool Parameters

**fetch_advisory_for_task** accepts:

- `task` (required, string): Detailed description of what you're trying to accomplish
- `n_results` (optional, integer 1-25, default 10): Number of advisories to return

**Example tool call:**
```json
{
  "task": "implementing error wrapping with context using fmt.Errorf and %w verb for error chains in a file processing pipeline",
  "n_results": 5
}
```

For detailed examples and best practices, run:
```bash
./bin/code-advisory-mcp --print-instructions
```

## Building Custom Databases

The server includes an embedded database with Go standards, but you can build custom databases with your own documentation.

### 1. Build the Database Builder

```bash
go build -o bin/build-advisory-db ./cmd/builder
```

### 2. Create a Manifest

Create `manifest.json` with your documents:

```json
{
  "documents": [
    {
      "path": "docs/error-handling.md",
      "uri": "https://standards.company.com/error-handling",
      "chunking_strategy": "h2_h3",
      "metadata": {
        "title": "Error Handling Guide",
        "language": "Go",
        "category": "best-practices",
        "version": "1.0",
        "author": "Engineering Team"
      }
    }
  ]
}
```

**Chunking strategies:**
- `h2_h3` - Split on H2 and H3 headings (preserves H2 context in H3 chunks)
- `h2` - Split on H2 headings only

### 3. Build the Database

**With Ollama:**
```bash
./bin/build-advisory-db \
  --backend ollama \
  --model nomic-embed-text \
  --manifest manifest.json \
  --output custom.db
```

**With Azure OpenAI:**
```bash
./bin/build-advisory-db \
  --backend azure \
  --azure-endpoint https://your-resource.openai.azure.com \
  --azure-key your-api-key \
  --model text-embedding-3-small \
  --manifest manifest.json \
  --output custom.db
```

### 4. Use Your Custom Database

```bash
./bin/code-advisory-mcp --db custom.db
```

Or embed it in the binary by replacing `cmd/server/embedded/advisory.gob`:

```bash
# Build export tool
go build -o bin/export-db ./cmd/export

# Export your database to single file
./bin/export-db --input custom.db --output cmd/server/embedded/advisory.gob

# Rebuild server with embedded database
go build -o bin/code-advisory-mcp ./cmd/server
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `EMBEDDING_BACKEND` | `ollama` | Backend: `azure` or `ollama` |
| `EMBEDDING_MODEL` | `nomic-embed-text` | Model name |
| `AZURE_OPENAI_ENDPOINT` | - | Azure endpoint (required for azure backend) |
| `AZURE_OPENAI_KEY` | - | Azure API key (required for azure backend) |
| `OLLAMA_BASE_URL` | `http://localhost:11434/api` | Ollama API URL |
| `DB_PATH` | (embedded) | Path to external database file |

### Command-Line Flags

```bash
./bin/code-advisory-mcp --help
```

| Flag | Description |
|------|-------------|
| `--db` | Path to advisory database (empty = use embedded) |
| `--backend` | Embedding backend: azure or ollama |
| `--azure-endpoint` | Azure OpenAI endpoint |
| `--azure-key` | Azure OpenAI API key |
| `--ollama-url` | Ollama base URL |
| `--model` | Embedding model name |
| `--print-instructions` | Print usage instructions and exit |

## Development

### Prerequisites

- Go 1.23+
- Ollama (for local development) or Azure OpenAI
- [mask](https://github.com/jacobdeichert/mask) (optional task runner)

### Build All Tools

```bash
# Build MCP server
go build -o bin/code-advisory-mcp ./cmd/server

# Build database builder
go build -o bin/build-advisory-db ./cmd/builder

# Build query tool (for testing)
go build -o bin/query-advisory-db ./cmd/query

# Build export tool
go build -o bin/export-db ./cmd/export
```

Or use mask:
```bash
mask build
```

### Testing

```bash
# Run all tests
mask test

# Or directly
go test ./...

# Run tests with race detector
go test -race ./...

# Run specific package tests
go test ./internal/chunker -v
```

### Code Quality

```bash
# Format code
go tool goimports -w .

# Run static analysis
go tool staticcheck ./...

# Check for unchecked errors
go tool errcheck ./...

# Check for vulnerabilities
go tool govulncheck ./...
```

Or use mask:
```bash
mask check
```

## Architecture

```
cmd/
  server/         # MCP server (main binary)
  builder/        # Database builder tool
  query/          # Query tool for testing
  export/         # Database export tool

internal/
  chunker/        # Markdown chunking logic
  embedder/       # Embedding backend clients
  database/       # chromem-go wrapper
  manifest/       # Manifest parsing
  mcp/            # MCP handler and tools

cmd/server/embedded/
  advisory.gob         # Embedded database (770KB)
  INSTRUCTIONS.md      # Usage instructions (19KB)
```

## How It Works

### Contextual Chunking

Documents are split using a contextual retrieval strategy:

1. **Discard H1 and preamble** - Skip document title
2. **Capture H2 context** - Extract H2 heading and preamble
3. **Prepend context to H3 sections** - Each H3 chunk includes H2 context
4. **Generate deep-link URLs** - Slugified headings as URL fragments

**Example chunk:**
```markdown
## Error Handling

Always wrap errors with context using fmt.Errorf with %w verb.

### Error Wrapping

Use fmt.Errorf("%w", err) to wrap errors:

if err != nil {
    return fmt.Errorf("failed to connect: %w", err)
}
```

URL: `https://go.dev/doc/effective_go#error-wrapping`

### Vector Search

1. Query text is embedded using the same model
2. Semantic similarity search finds top N chunks
3. Results include source, URL, and full content
4. Relevance scores help prioritize results

### Embedded Database

The server binary includes a 770KB database with 126 chunks:
- **Effective Go** (43 chunks)
- **Go Code Review Comments** (32 chunks)
- **Go Test Comments** (14 chunks)
- **Google Go Style Guide** (37 chunks)

No external files needed - the binary is completely self-contained.

## Troubleshooting

### Ollama Connection Issues

```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# Start Ollama if needed
ollama serve

# Verify model is downloaded
ollama list | grep nomic-embed-text
```

### MCP Server Not Appearing in Copilot

1. Check the configuration file path for your IDE
2. Verify the binary path is absolute
3. Check environment variables are set correctly
4. Restart your IDE
5. Check IDE logs for MCP connection errors

### Building Custom Database

If database building fails:
- Verify manifest.json is valid JSON
- Check markdown files exist at specified paths
- Ensure Ollama is running for ollama backend
- Verify Azure credentials for azure backend
- Check network connectivity

## Embedded Database Contents

The included database contains:

| Source | Chunks | Topics |
|--------|--------|--------|
| Effective Go | 43 | Formatting, naming, control structures, data, methods, interfaces, concurrency, errors |
| Code Review Comments | 32 | Code organization, naming, error handling, testing, documentation |
| Test Comments | 14 | Test structure, naming, table-driven tests, subtests, helpers |
| Google Go Style Guide | 37 | Style decisions, best practices, common patterns, design principles |

**Total:** 126 chunks covering comprehensive Go coding standards

## License

See LICENSE file for details.

## Contributing

Contributions welcome! Please:
1. Run tests: `mask test`
2. Format code: `go tool goimports -w .`
3. Run checks: `mask check`
4. Submit pull request

## Project Status

✅ **Phase 1: Database Builder** - Complete
- JSON manifest parsing
- Contextual markdown chunking  
- Azure OpenAI and Ollama embeddings
- Idempotent processing
- chromem-go integration

✅ **Phase 2: MCP Server** - Complete
- MCP SDK integration
- `fetch_advisory_for_task` tool
- Embedded database support
- Comprehensive usage instructions
- Production ready

✅ **Phase 3: Deployment** - Complete  
- Self-contained binary with embedded database
- GitHub Copilot integration
- Installation and usage documentation
- Development and deployment guides

## Roadmap

Future enhancements:
- [ ] Web UI for database management
- [ ] Support for more embedding backends (OpenAI, Anthropic)
- [ ] Multi-language support beyond Go
- [ ] Prometheus metrics
- [ ] Docker images
- [ ] Caching layer for frequently queried tasks
- [ ] Admin API for database management
