# User Story: Code Advisory Vector Search for GitHub Copilot

## Elevator Pitch

Copilot needs a significant amount of coaching to write at a **high** standard and code reviews can be hit and miss. The different models have all likely internalised common and popular style guides such as Effective Go, but it still takes back-and-forth messaging between the model and a human that often just quotes these open source documents. Instead of instructing the model in the particulars of these docs I propose a **fully self-contained** tool that any model can call upon on-demand to retrieve specific coaching advice for the specific task at-hand. This tool can be extended at build time with company specific guidelines and examples.

## As a
Software Engineer

## I want to
Query company coding standards and style guides through GitHub Copilot via MCP

## So that
I can write code that conforms to company standards without manually searching documentation, instructing copilot or consuming large portions of the context window.

## Acceptance Criteria
- [ ] Markdown documents can be chunked and embedded into a vector database using contextual retrieval strategy
- [ ] Database building is idempotent and incremental (skips existing embeddings)
- [ ] Database can be exported to a single portable file
- [ ] MCP server can load database from file or embedded binary
- [ ] MCP server provides advisory tool returning configurable number of results (default 10, max 25)
- [ ] Tool accepts task description and scope parameter
- [ ] Docker image can be built and deployed
- [ ] Can be configured in GitHub Copilot as an MCP server
- [ ] Search results are semantically relevant to coding queries with deep-linked URLs
- [ ] Server includes embedded prompt instructions accessible via CLI

## Technical Implementation

### Phase 1: Database Builder Tool

**Command:** `build-advisory-db`

**Functionality:**
- Read markdown files from JSON manifest (not directory traversal)
- Parse markdown using `github.com/yuin/goldmark`
- Chunk documents using contextual retrieval strategy (H2 context + H3 sections)
- Generate embeddings using Azure OpenAI text-embedding-3-small
- Idempotent/incremental: check SHA1 hash before calling embedding API
- Store in chromem-go vector database
- Export to single file: `advisory.db`

**Input Manifest Format (JSON):**
```json
{
  "documents": [
    {
      "path": "docs/golang/error-handling.md",
      "uri": "https://standards.company.com/golang/error-handling",
      "metadata": {
        "language": "Go",
        "category": "error-handling",
        "version": "1.0",
        "author": "Platform Team"
      }
    }
  ]
}
```

**Configuration (environment variables or flags):**
- `--manifest` / `MANIFEST_FILE`: Path to JSON manifest (required)
- `--output` / `OUTPUT_FILE`: Database output path (default: `advisory.db`)
- `--azure-endpoint` / `AZURE_OPENAI_ENDPOINT`: Azure OpenAI endpoint
- `--azure-key` / `AZURE_OPENAI_KEY`: Azure OpenAI API key
- `--model` / `EMBEDDING_MODEL`: Model name (default: `text-embedding-3-small`)

**Contextual Retrieval Chunking Strategy:**

1. **Discard H1 and preamble**: Skip document title (H1) and all content before first H2
2. **Capture H2 context**: Extract H2 heading and content up to first H3
3. **Prepend context to H3 sections**: For each H3:
   - Prepend the H2 heading + H2 content
   - Append H3 heading + H3 content
   - This creates a self-contained chunk with full context
4. **Generate document ID**: Use SHA1 hash of chunk content as unique ID
5. **Construct deep-link URL**: 
   - Slugify H3 heading (lowercase, hyphens, remove special chars)
   - Append as fragment to document URI
   - Example: `https://go.dev/doc/effective_go#package-names`
6. **Store metadata**:
   - Copy all metadata from manifest entry
   - Add `url` field with constructed deep-link
   - Add `source_file` with original path
   - Add `h2_heading` and `h3_heading` for context

**Example Chunk:**
```
## Error Handling

Always wrap errors with context using fmt.Errorf with %w verb. This preserves 
the error chain for debugging.

### Error Wrapping

Use fmt.Errorf("%w", err) to wrap errors:

    if err != nil {
        return fmt.Errorf("failed to connect: %w", err)
    }

This allows callers to use errors.Is() and errors.As() for inspection.
```

**Idempotent/Incremental Processing:**
- On startup, load existing `advisory.db` if present
- For each chunk, compute SHA1 hash of content
- Query chromem-go by ID (SHA1 hash)
- If exists: skip embedding API call
- If missing: call embedding API and insert
- This allows re-running after adding new documents without re-embedding existing content

**Key Libraries:**
- `github.com/yuin/goldmark` - Markdown parsing
- `github.com/philippgille/chromem-go` - Vector database
- Azure OpenAI Go SDK for embeddings
- `crypto/sha1` - Document ID generation

### Phase 2: MCP Server

**Server Name:** `code-advisory-mcp`

**MCP Tool Definition:**
```json
{
  "name": "fetch_advisory_for_task",
  "description": "Fetch coding standards and style guide advisories relevant to a specific development task. Returns contextual guidance with deep-links to full documentation.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "task": {
        "type": "string",
        "description": "Description of the coding task (e.g., 'implementing error handling', 'writing unit tests', 'naming a new service')"
      },
      "scope": {
        "type": "string",
        "enum": ["package", "function", "method", "struct", "interface", "variable", "constant", "test", "documentation", "general"],
        "description": "Scope of the task to help filter relevant advisories",
        "default": "general"
      },
      "n_results": {
        "type": "integer",
        "description": "Number of results to return (default: 10, max: 25)",
        "default": 10,
        "minimum": 1,
        "maximum": 25
      }
    },
    "required": ["task"]
  }
}
```

**Functionality:**
- Load chromem-go database from file or embedded binary
- Accept MCP tool calls via stdio (using official `github.com/modelcontextprotocol/go-sdk/mcp`)
- Generate query embedding using same Azure model
- Perform vector similarity search
- Return top N results with metadata and deep-link URLs
- Handle errors gracefully (return empty results rather than crashing)
- Provide embedded prompt instructions via CLI flag

**Configuration (environment variables):**
- `DB_PATH`: Path to advisory.db file (if not embedded)
- `AZURE_OPENAI_ENDPOINT`: Azure OpenAI endpoint
- `AZURE_OPENAI_KEY`: Azure OpenAI API key
- `EMBEDDING_MODEL`: Model name (must match builder)

**Database Embedding (Build-time):**
- Use `go:embed` directive to embed database file into binary
- Database path passed at build time via linker flag:
  ```bash
  go build -tags embed \
    -ldflags "-X main.embeddedDBPath=/path/to/advisory.db" \
    -o code-advisory-mcp
  ```
- At runtime, check build tag:
  - If `embed` tag: extract embedded DB to temp file or read directly
  - If not embedded: read from `DB_PATH` environment variable
- Embedded mode: no external dependencies at runtime
- External mode: more flexible for updates without recompilation

**Response Format:**
```json
{
  "results": [
    {
      "content": "## Error Handling\n\nAlways wrap errors...\n\n### Error Wrapping\n\nUse fmt.Errorf...",
      "metadata": {
        "url": "https://standards.company.com/golang/error-handling#error-wrapping",
        "source_file": "docs/golang/error-handling.md",
        "h2_heading": "Error Handling",
        "h3_heading": "Error Wrapping",
        "language": "Go",
        "category": "error-handling",
        "similarity_score": 0.92
      }
    }
  ]
}
```

**Embedded Prompt Instructions:**
- Store prompt instructions as embedded string in binary
- Access via CLI flag: `--print-instructions` or `--instructions`
- Instructions should guide LLM on:
  - When to use the advisory tool
  - How to interpret results
  - How to present URLs to users
  - Best practices for task description and scope selection

**Example prompt instructions:**
```
# Code Advisory Tool Usage

Use the `fetch_advisory_for_task` tool when:
- Writing new code that should conform to company standards
- Reviewing code for standards compliance
- User asks about coding conventions, best practices, or style guidelines
- Refactoring code to meet standards
- User references "our standards", "company guidelines", or "best practices"

Task description:
- Use natural language describing what you're implementing
- Be specific: "implementing error handling for database operations" not just "errors"
- Include context: "naming a new HTTP client package" rather than "naming"
- Examples: "writing unit tests for repository layer", "documenting a public API"

Scope selection:
- package: When creating or organizing packages/modules
- function/method: When implementing function logic or signatures
- struct/interface: When defining data structures or contracts
- variable/constant: When naming variables or constants
- test: When writing tests
- documentation: When writing comments, READMEs, or docs
- general: When scope is unclear or task spans multiple scopes

Presenting results:
- Summarize the guidance from returned chunks
- Always include the URL for users to read full context
- If multiple results, highlight the most relevant
- Mention the source document for credibility
- Apply the guidance to the specific code being written

Example usage:
User: "I need to add error handling to this database function"
Assistant: [calls fetch_advisory_for_task with task="implementing error handling for database operations" scope="function"]
Assistant: "According to our coding standards, you should wrap errors with context 
using fmt.Errorf with %w verb. This preserves the error chain for debugging. For 
database operations specifically, include the operation type and relevant identifiers 
in the error message. [Full details](https://standards.company.com/golang/error-handling#error-wrapping)"
```

**CLI Interface:**
```bash
# Run MCP server (stdio mode)
code-advisory-mcp

# Print embedded prompt instructions
code-advisory-mcp --print-instructions

# Run with external database
DB_PATH=/path/to/advisory.db code-advisory-mcp
```

### Phase 3: Deployment

**Docker Image:**
- Multi-stage build: builder + runtime
- Base image: `golang:1.23-alpine` (builder), `alpine:latest` (runtime)
- Build with embedded database using linker flags
- Expose MCP server on stdio (not HTTP)
- Health check: validate DB loads successfully

**Dockerfile structure:**
```dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /build
COPY . .

# Copy database file into build context
ARG DB_FILE=advisory.db
COPY ${DB_FILE} /build/advisory.db

# Build with embedded DB
RUN go build -tags embed \
    -ldflags "-X main.embeddedDBPath=/build/advisory.db" \
    -o /mcp-server ./cmd/server

FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

COPY --from=builder /mcp-server /usr/local/bin/code-advisory-mcp

ENTRYPOINT ["/usr/local/bin/code-advisory-mcp"]
```

**Build and push:**
```bash
# Build database
go run ./cmd/builder \
  --manifest=docs/manifest.json \
  --output=advisory.db

# Build Docker image with embedded DB
docker build \
  --build-arg DB_FILE=advisory.db \
  -t company/code-advisory-mcp:latest \
  .

# Push to registry
docker push company/code-advisory-mcp:latest
```

**GitHub Copilot Configuration:**
Add to `.github/copilot-mcp.json` or Copilot settings:
```json
{
  "mcpServers": {
    "code-advisory": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "company/code-advisory-mcp:latest"],
      "env": {
        "AZURE_OPENAI_ENDPOINT": "${AZURE_OPENAI_ENDPOINT}",
        "AZURE_OPENAI_KEY": "${AZURE_OPENAI_KEY}"
      }
    }
  }
}
```

## Implementation Notes

### Package Structure
```
.
├── cmd/
│   ├── builder/       # Database builder tool
│   │   └── main.go
│   └── server/        # MCP server
│       └── main.go
├── internal/
│   ├── chunker/       # Markdown chunking logic
│   ├── embedder/      # Azure OpenAI embedding client
│   ├── database/      # chromem-go wrapper
│   └── mcp/           # MCP tool handlers
├── docs/              # Sample markdown files
│   └── manifest.json  # Document manifest
├── Dockerfile
├── go.mod
└── README.md
```

### Embedding Model Consistency
**CRITICAL:** Builder and MCP server MUST use identical embedding models and dimensions. Model mismatch will produce garbage results.

### chromem-go Advantages
- Pure Go (no CGO, no sqlite-vec complications)
- Simple persistence model
- Built-in similarity search
- Easy to embed in binary

### Chunking Considerations
- Preserve heading hierarchy for context
- Include file path metadata for traceability
- H2 context prepended to every H3 chunk ensures self-contained results
- Monitor token counts to stay within model limits (8000 tokens)

### Scope Parameter Usage
- Query embeddings should incorporate scope information
- Consider augmenting task with scope: `task + " in " + scope + " context"`
- Scope can help filter results if metadata includes code element types
- Not strictly required but helps improve result relevance

### Error Handling
- Builder: fail fast on embedding errors, log which files failed
- MCP Server: never crash, return empty results on error
- Log all errors for debugging
- Validate n_results is within bounds (1-25)

### Testing Strategy
- Unit tests: chunking logic, metadata extraction, URL construction
- Integration tests: end-to-end with sample markdown files
- MCP protocol compliance: test with MCP inspector tool
- Verify idempotent behavior: run builder twice, check no duplicate API calls
- Load tests: ensure reasonable performance with large databases

### Security Considerations
- Never log API keys
- Validate input task length (prevent abuse)
- Sanitize file paths in metadata
- Consider rate limiting for MCP tool calls
- Validate scope enum values

### Performance Targets
- Database build: < 1 minute for 100 markdown files
- Incremental rebuild: < 10 seconds for 10 new files
- Query latency: < 500ms for search + embedding
- Docker image size: < 50MB

### Future Enhancements (Out of Scope)
- Metadata-based filtering (by language, category, version)
- Analytics on query patterns and frequently accessed advisories
- Caching for repeated queries
- Web UI for testing queries
- Support for non-markdown formats (RST, AsciiDoc)
- Multi-database support (different teams/languages)

## Dependencies
- Go 1.23+
- `github.com/yuin/goldmark` - Markdown parsing
- `github.com/philippgille/chromem-go` - Vector database
- `github.com/modelcontextprotocol/go-sdk/mcp` - Official MCP SDK
- Azure OpenAI account with text-embedding-3-small deployed
- Docker for containerization
- GitHub Copilot with MCP support

## Estimates
- Phase 1 (Builder): 1 day
  - Manifest parsing and file loading: 0.5 hour (copilot/claude)
  - Contextual chunking implementation: 1 hour (copilot/claude)
  - Idempotent/incremental logic: 1 hour (copilot/claude)
  - URL construction and metadata handling: 0.5 hour (copilot/claude)
  - Testing: 5 hours (human)
- Phase 2 (MCP Server): 1 day
  - MCP SDK integration and tool handler implementation: 0.5 hour  (copilot/claude)
  - Embedded DB and linker flags: 0.5 hour  (copilot/claude)
  - Prompt instructions: 2 hour  (human/copilot/claude)
  - Testing: 5 hour (human)
- Phase 3 (Deployment): 1 day
  - Dockerfile and build process
  - Documentation
  - Integration testing with Copilot
- **Total: 3 days**

## Definition of Done
- [ ] Code compiles and passes tests
- [ ] Builder successfully processes sample markdown directory
- [ ] Builder is idempotent (second run skips existing embeddings)
- [ ] MCP server responds correctly to fetch_advisory_for_task queries
- [ ] Scope parameter properly influences results
- [ ] Docker image builds and runs with embedded database
- [ ] Successfully integrated with GitHub Copilot in test environment
- [ ] Prompt instructions accessible via CLI and properly formatted
- [ ] Documentation written (README with usage examples)
- [ ] Code reviewed and merged
