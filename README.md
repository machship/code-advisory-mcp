# Code Advisory MCP Server

A Model Context Protocol (MCP) server that provides code advisory and style guide information through semantic search. This tool enables GitHub Copilot and other MCP clients to query company coding standards and best practices on-demand.

## Overview

This project implements a vector search system for coding standards documentation. It chunks markdown documents using a contextual retrieval strategy, generates embeddings via Azure OpenAI, and stores them in a local vector database for fast semantic search.

## Features

- **Multiple Backends**: Support for both Azure OpenAI and local Ollama embeddings
- **Contextual Chunking**: Smart markdown parsing that preserves document structure and context
- **Idempotent Processing**: Skip already-embedded content to save API calls
- **Portable Database**: Single-file database that can be embedded in binaries
- **Deep-Linked Results**: Automatic URL fragment generation for precise documentation navigation
- **No Cloud Required**: Use Ollama for completely local, private embeddings

## Quick Start with Ollama (No API Keys Needed!)

The fastest way to get started is with Ollama, which runs completely locally:

```bash
# 1. Install Ollama (if not already installed)
# Visit https://ollama.ai or use: brew install ollama

# 2. Pull an embedding model
ollama pull nomic-embed-text

# 3. Run the demo script
./demo-ollama.sh
```

That's it! You now have a working advisory database with 80 chunks from Go documentation.

## Phase 1: Database Builder (Complete)

The database builder tool processes markdown documents and creates a searchable vector database.

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd code-advisory-mcp

# Install dependencies
go mod download

# Build the builder tool
go build -o bin/build-advisory-db ./cmd/builder
```

### Usage

#### 1. Create a Manifest File

Create a JSON manifest listing your markdown documents:

```json
{
  "documents": [
    {
      "path": "docs/golang/error-handling.md",
      "uri": "https://standards.company.com/golang/error-handling",
      "metadata": {
        "language": "Go",
        "category": "error-handling",
        "version": "1.0"
      }
    }
  ]
}
```

#### 2. Run the Builder

**With Ollama (Local, No API Keys):**

```bash
./bin/build-advisory-db \
  --backend ollama \
  --model nomic-embed-text \
  --manifest manifest.json \
  --output advisory.db
```

**With Azure OpenAI:**

```bash
./bin/build-advisory-db \
  --backend azure \
  --manifest manifest.json \
  --output advisory.db \
  --azure-endpoint https://your-resource.openai.azure.com \
  --azure-key your-api-key \
  --model text-embedding-3-small
```

Or using environment variables:

```bash
# For Ollama
export EMBEDDING_BACKEND=ollama
export EMBEDDING_MODEL=nomic-embed-text

# For Azure OpenAI
export EMBEDDING_BACKEND=azure
export AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com
export AZURE_OPENAI_KEY=your-api-key
export EMBEDDING_MODEL=text-embedding-3-small

# Common
export MANIFEST_FILE=manifest.json
export OUTPUT_FILE=advisory.db

./bin/build-advisory-db
```

#### 3. Re-run for Incremental Updates

The builder is idempotent. Add new documents to your manifest and re-run:

```bash
./bin/build-advisory-db --manifest manifest.json
```

Existing chunks are skipped, only new content generates embeddings.

### How It Works

#### Contextual Retrieval Chunking Strategy

1. **Discard H1 and preamble**: Skip document title and content before first H2
2. **Capture H2 context**: Extract H2 heading and content up to first H3
3. **Prepend context to H3 sections**: Each H3 chunk includes:
   - H2 heading + H2 content (context)
   - H3 heading + H3 content (specific section)
4. **Generate document ID**: SHA1 hash of chunk content
5. **Construct deep-link URL**: Slugified H3 heading appended as fragment

**Example Chunk:**

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

#### Idempotent Processing

- On startup, loads existing `advisory.db` if present
- For each chunk, computes SHA1 hash of content
- Queries chromem-go database by ID
- If exists: skip embedding API call
- If missing: call embedding API and insert

This allows re-running after adding documents without re-embedding existing content.

### Architecture

```
cmd/
  builder/        # Database builder CLI
    main.go
internal/
  chunker/        # Markdown chunking logic
    chunker.go
    chunker_test.go
  embedder/       # Azure OpenAI embedding client
    embedder.go
  database/       # chromem-go wrapper
    database.go
  manifest/       # JSON manifest parsing
    manifest.go
    manifest_test.go
```

### Testing

Run all tests:

```bash
mask test
```

Or directly with Go:

```bash
go test ./...
```

Run specific package tests:

```bash
go test ./internal/chunker -v
```

### Configuration

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--manifest` | `MANIFEST_FILE` | *(required)* | Path to JSON manifest file |
| `--output` | `OUTPUT_FILE` | `advisory.db` | Database output path |
| `--backend` | `EMBEDDING_BACKEND` | `azure` | Backend: `azure` or `ollama` |
| `--azure-endpoint` | `AZURE_OPENAI_ENDPOINT` | - | Azure OpenAI endpoint (required for azure) |
| `--azure-key` | `AZURE_OPENAI_KEY` | - | Azure OpenAI API key (required for azure) |
| `--ollama-url` | `OLLAMA_BASE_URL` | `http://localhost:11434/api` | Ollama API URL |
| `--model` | `EMBEDDING_MODEL` | `text-embedding-3-small` | Embedding model name |

**Recommended Models:**
- **Ollama**: `nomic-embed-text` (274MB, 768 dimensions, excellent quality)
- **Azure OpenAI**: `text-embedding-3-small` (best quality/cost ratio)

## Phase 2: MCP Server (Coming Soon)

The MCP server will:
- Load the vector database
- Provide `fetch_advisory_for_task` tool
- Accept task description and scope parameters
- Return top N relevant results with deep links
- Include embedded prompt instructions

## Phase 3: Deployment (Coming Soon)

- Docker containerization
- Database embedding in binary
- GitHub Copilot integration
- Production deployment guides

## Development

### Prerequisites

- Go 1.23+
- **Either**:
  - [Ollama](https://ollama.ai) for local embeddings (recommended for development)
  - Azure OpenAI account with text-embedding-3-small deployed
- [mask](https://github.com/jacobdeichert/mask) (optional, for task automation)

### Build

```bash
go build -o bin/build-advisory-db ./cmd/builder
```

### Format Code

```bash
go tool goimports -w .
```

### Run Static Analysis

```bash
mask check
```

Or manually:

```bash
go tool staticcheck ./...
go tool errcheck ./...
go tool govulncheck ./...
```

## Dependencies

- [goldmark](https://github.com/yuin/goldmark) - Markdown parsing
- [chromem-go](https://github.com/philippgille/chromem-go) - Vector database
- [openai-go](https://github.com/openai/openai-go) - OpenAI client with Azure support

## License

See LICENSE file for details.

## Contributing

Contributions are welcome! Please:
1. Run tests: `mask test`
2. Format code: `go tool goimports -w .`
3. Check for issues: `mask check`
4. Submit pull request

## Roadmap

- [x] Phase 1: Database Builder
  - [x] JSON manifest parsing
  - [x] Contextual markdown chunking
  - [x] Azure OpenAI embeddings
  - [x] Idempotent processing
  - [x] chromem-go integration
  - [x] Unit tests
- [ ] Phase 2: MCP Server
  - [ ] MCP SDK integration
  - [ ] Advisory search tool
  - [ ] Scope parameter support
  - [ ] Embedded prompt instructions
- [ ] Phase 3: Deployment
  - [ ] Docker image
  - [ ] Binary embedding
  - [ ] GitHub Copilot integration
  - [ ] Production guides
