# Quick Start Guide

Get started with the Code Advisory Database Builder in 5 minutes.

## Prerequisites

- Go 1.23 or later
- Azure OpenAI account with `text-embedding-3-small` deployment
- Markdown documentation files

## Step 1: Clone and Build

```bash
git clone <repository-url>
cd code-advisory-mcp
go mod download
go build -o bin/build-advisory-db ./cmd/builder
```

## Step 2: Configure Environment

Copy the example environment file and configure it:

```bash
cp .env.example .env
```

Edit `.env` and set your Azure OpenAI credentials:

```env
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com
AZURE_OPENAI_KEY=your-actual-api-key
```

## Step 3: Create a Manifest

The provided `manifest.json` already includes the Go documentation from the `resources/` directory:

```json
{
  "documents": [
    {
      "path": "resources/Effective-Go.md",
      "uri": "https://go.dev/doc/effective_go",
      "metadata": {
        "language": "Go",
        "category": "best-practices"
      }
    }
  ]
}
```

## Step 4: Build the Database

Run the builder:

```bash
source .env
./bin/build-advisory-db
```

Or with explicit flags:

```bash
./bin/build-advisory-db \
  --manifest manifest.json \
  --output advisory.db \
  --azure-endpoint $AZURE_OPENAI_ENDPOINT \
  --azure-key $AZURE_OPENAI_KEY
```

You'll see output like:

```
2024/01/15 10:30:00 Loading manifest from manifest.json
2024/01/15 10:30:00 Loaded manifest with 4 documents
2024/01/15 10:30:00 Initializing embedder with model text-embedding-3-small
2024/01/15 10:30:00 Opening database at advisory.db
2024/01/15 10:30:01 [1/4] Processing resources/Effective-Go.md
2024/01/15 10:30:01   Found 23 chunks
2024/01/15 10:30:01     [1/23] Generating embedding for chunk (ID: a3f5c892)
...

=== Summary ===
Total documents processed: 4
Total chunks: 89
Added chunks: 89
Skipped chunks (already exist): 0
Failed chunks: 0
Database saved to: advisory.db
```

## Step 5: Verify the Database

The database file `advisory.db` should now exist:

```bash
ls -lh advisory.db
```

## Step 6: Add More Documents (Optional)

Edit `manifest.json` to add your own markdown files:

```json
{
  "documents": [
    {
      "path": "docs/my-guide.md",
      "uri": "https://company.com/docs/my-guide",
      "metadata": {
        "language": "Go",
        "category": "custom-guide",
        "team": "platform"
      }
    }
  ]
}
```

Then re-run the builder. It will only process new documents:

```bash
./bin/build-advisory-db
```

## Testing

Run the test suite:

```bash
mask test
```

Or with Go directly:

```bash
go test ./...
```

## Troubleshooting

### "failed to get embeddings" error

- Verify your Azure OpenAI endpoint URL is correct
- Check that your API key is valid
- Ensure the deployment name matches your model (default: `text-embedding-3-small`)

### "failed to read file" error

- Check that paths in `manifest.json` are correct
- Paths can be relative to the manifest file location or absolute

### "no chunks found" warning

- Verify your markdown files have H2 and H3 headings
- The chunker skips H1 and preamble text
- Documents with only H2 or only H3 won't generate chunks

## Next Steps

- **Phase 2**: Wait for the MCP server implementation to query this database
- **Phase 3**: Deploy with Docker and integrate with GitHub Copilot
- **Read the docs**: See [README.md](README.md) for detailed information

## Getting Help

- Check the [README.md](README.md) for detailed documentation
- Review test files for usage examples
- Open an issue on GitHub

## Performance Tips

1. **Incremental builds**: The builder is idempotent, so you can safely re-run it
2. **Batch processing**: Add multiple documents to your manifest at once
3. **Monitor costs**: Each chunk requires one embedding API call
4. **Database size**: Typical database is ~1-5MB for 100 documents
