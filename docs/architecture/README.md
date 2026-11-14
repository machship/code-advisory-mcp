---
lastReviewed: 2026-11-14
---
# code-advisory-mcp

A Model Context Protocol server that gives coding assistants (GitHub Copilot, Claude) MachShip's
Go coding standards on demand. An assistant calls the `fetch_advisory_for_task` tool with a
plain-language task; the server embeds it, runs a vector search over chunked style guides and
returns the closest sections with deep links to the source. Owned by uplift; Dev Only.

## Where it runs

**On a developer's laptop only.** It is pure tooling: nothing is deployed to k8s, Azure or AWS,
and there is no image or CI workflow (the Maskfile mentions a Dockerfile that does not exist).
A developer builds `bin/code-advisory-mcp` from `cmd/server` and registers it in their MCP
client, which starts it as a child process and talks to it over **stdio** (`ServeStdio`). There
is no network listener.

| Binary | What it does |
|---|---|
| `cmd/server` | The MCP server. Uses the database embedded in the binary (`cmd/server/embedded/advisory.gob`) unless `-db` points at another one |
| `cmd/builder` | Chunks the markdown in `manifest.json` (`resources/`) and generates embeddings into a chromem-go database |
| `cmd/export` | Exports that database to the `.gob` file the server embeds |
| `cmd/query` | Command-line query tool for testing |

Indexed sources (`manifest.json`, `resources/`): Effective Go, Go Code Review Comments, Go Test
Comments, and Google's Go style guide (best practices, decisions).

## What it touches

- **Embedding backend**, needed per query to embed the task text (`EMBEDDING_BACKEND`):
  - `ollama` (default): a local Ollama at `OLLAMA_BASE_URL` (default localhost:11434), model `nomic-embed-text`.
  - `azure`: an Azure OpenAI deployment the developer supplies via `AZURE_OPENAI_ENDPOINT` and
    `AZURE_OPENAI_KEY` (environment or flags). TODO(confirm) which Azure OpenAI resource, if any,
    MachShip developers use; none is named in the repo, so no dependency is declared.
- Optional query logs to `QUERY_LOG_DIR` on the local disk.

No MachShip service, database or secret store is involved.

## Context

See [diagrams/context.md](diagrams/context.md).

## Key decisions

- Ship the vector database inside the binary so it works with no setup beyond an embedding backend.
- chromem-go (in-process vector store) instead of a hosted vector database.
- Ollama as the default backend so it can run with no cloud account or API key.

## Lifecycle notes

`in-development` (GitHub Dev Only). Last change Jul 2026. MachShip-specific standards are not
indexed yet; only public Go guides.
