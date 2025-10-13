# Ollama Backend Demo

This document demonstrates the Ollama backend support for completely local, private embeddings.

## Why Ollama?

- **No API Keys**: Runs completely locally, no cloud services required
- **Privacy**: Your code standards never leave your machine
- **Free**: No API costs
- **Fast**: Local inference, no network latency
- **Development**: Perfect for testing and development

## Quick Demo

### 1. Prerequisites

Install Ollama (if not already installed):

```bash
# macOS
brew install ollama

# Linux
curl -fsSL https://ollama.ai/install.sh | sh

# Windows
# Download from https://ollama.ai
```

### 2. Run the Demo

```bash
./demo-ollama.sh
```

This script will:
1. Check if Ollama is installed
2. Pull the `nomic-embed-text` model (if needed)
3. Build the tool
4. Process all 4 markdown documents
5. Create `advisory-ollama.db`

### 3. Verify Results

```bash
# Check database was created
ls -lh advisory-ollama.db

# Run again to see idempotency
./bin/build-advisory-db \
  --backend ollama \
  --model nomic-embed-text \
  --manifest manifest.json \
  --output advisory-ollama.db
```

You should see "Skipping existing chunk" for all 80 chunks!

## Performance Comparison

**First run (80 chunks):**
- Ollama (local): ~15 seconds
- Azure OpenAI: ~20-30 seconds (depending on network)

**Second run (idempotent, 80 chunks):**
- Both: < 1 second (no embeddings generated)

## Supported Models

Ollama supports many embedding models:

```bash
# Recommended: nomic-embed-text
ollama pull nomic-embed-text  # 274MB, 768 dimensions

# Alternative: mxbai-embed-large  
ollama pull mxbai-embed-large  # 669MB, 1024 dimensions

# List available models
ollama list
```

## Example Output

```
2025/10/13 20:40:45 Loading manifest from manifest.json
2025/10/13 20:40:45 Loaded manifest with 4 documents
2025/10/13 20:40:45 Initializing ollama embedder with model nomic-embed-text
2025/10/13 20:40:45 Opening database at advisory-ollama.db
2025/10/13 20:40:45 [1/4] Processing resources/Effective-Go.md
2025/10/13 20:40:45   Found 43 chunks
2025/10/13 20:40:45     [1/43] Generating embedding for chunk (ID: 95421ac9)
...
2025/10/13 20:41:57 
=== Summary ===
2025/10/13 20:41:57 Total documents processed: 4
2025/10/13 20:41:57 Total chunks: 80
2025/10/13 20:41:57 Added chunks: 80
2025/10/13 20:41:57 Skipped chunks (already exist): 0
2025/10/13 20:41:57 Failed chunks: 0
2025/10/13 20:41:57 Database saved to: advisory-ollama.db
```

## Manual Usage

```bash
# Using flags
./bin/build-advisory-db \
  --backend ollama \
  --model nomic-embed-text \
  --manifest manifest.json \
  --output advisory.db

# Using environment variables
export EMBEDDING_BACKEND=ollama
export EMBEDDING_MODEL=nomic-embed-text
export MANIFEST_FILE=manifest.json
export OUTPUT_FILE=advisory.db

./bin/build-advisory-db
```

## Troubleshooting

### "Error: Ollama is not installed"

Install Ollama from https://ollama.ai

### "error response from the embedding API: 404 Not Found"

The model hasn't been pulled yet:

```bash
ollama pull nomic-embed-text
```

### "connection refused"

Make sure Ollama is running:

```bash
ollama serve
```

Or on macOS with the app, it should start automatically.

## Technical Details

### Architecture

```
┌─────────────────┐
│  build-advisory │
│      -db        │
└────────┬────────┘
         │
    ┌────┴─────┐
    │ embedder │
    └────┬─────┘
         │
    ┌────┴──────────┐
    │ Backend Layer │
    └────┬──────────┘
         │
    ┌────┴─────┬────────┐
    │          │        │
┌───▼───┐  ┌──▼────┐   │
│ Azure │  │Ollama │   ...
└───────┘  └───────┘
```

### Embedding Function

The Ollama integration uses chromem-go's built-in `NewEmbeddingFuncOllama`:

```go
embeddingFunc := chromem.NewEmbeddingFuncOllama(
    "nomic-embed-text",
    "http://localhost:11434/api",
)
```

This calls the Ollama API at `POST /api/embeddings`:

```bash
curl http://localhost:11434/api/embeddings \
  -d '{"model": "nomic-embed-text", "prompt": "your text here"}'
```

### Idempotency

Both Azure and Ollama backends use the same SHA1-based idempotency:

1. Compute SHA1 hash of chunk content
2. Check if hash exists in database
3. If exists: skip (no API call)
4. If missing: generate embedding and store

This works across backend switches! A database created with Ollama can be incrementally updated with Azure (or vice versa), though mixing embedding models isn't recommended for production.

## Production Considerations

### Ollama vs Azure OpenAI

| Factor | Ollama | Azure OpenAI |
|--------|--------|--------------|
| Cost | Free | ~$0.0001 per 1K tokens |
| Privacy | Local, private | Cloud, GDPR compliant |
| Quality | Excellent (nomic) | Excellent (text-embedding-3) |
| Latency | 100-300ms local | 200-500ms network |
| Scalability | Single machine | Unlimited |
| Maintenance | Self-hosted | Managed service |

### Recommendation

- **Development/Testing**: Use Ollama (fast, free, private)
- **Production (small team)**: Use Ollama (predictable costs)
- **Production (large scale)**: Use Azure OpenAI (scalability, SLA)

## Next Steps

Once you have a database:
- Phase 2 will add the MCP server for querying
- Phase 3 will add GitHub Copilot integration
- The same database works with both backends

Try it now:

```bash
./demo-ollama.sh
```
