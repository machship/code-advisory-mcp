package mcp

import (
	"context"
	"fmt"

	"github/machship/code-advisory-mcp/internal/database"
	"github/machship/code-advisory-mcp/internal/embedder"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/philippgille/chromem-go"
)

// Handler implements the MCP server handler for code advisory tools.
type Handler struct {
	db       *database.Database
	embedder *embedder.Embedder
	dbPath   string
	backend  embedder.Backend
	model    string
}

// Config holds configuration for the MCP handler.
type Config struct {
	DBPath        string
	Backend       embedder.Backend
	Endpoint      string
	APIKey        string
	Model         string
	OllamaBaseURL string
}

// NewHandler creates a new MCP handler.
func NewHandler(cfg Config) (*Handler, error) {
	// Initialize embedder
	emb, err := embedder.New(embedder.Config{
		Backend:       cfg.Backend,
		Endpoint:      cfg.Endpoint,
		APIKey:        cfg.APIKey,
		Model:         cfg.Model,
		OllamaBaseURL: cfg.OllamaBaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// Load database
	db, err := database.New(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load database: %w", err)
	}

	return &Handler{
		db:       db,
		embedder: emb,
		dbPath:   cfg.DBPath,
		backend:  cfg.Backend,
		model:    cfg.Model,
	}, nil
}

// RegisterTools registers MCP tools with the server.
func (h *Handler) RegisterTools(s *server.MCPServer) {
	// Register fetch_advisory_for_task tool
	s.AddTool(mcp.Tool{
		Name:        "fetch_advisory_for_task",
		Description: "Fetch coding advisory and best practices for a specific programming task. Returns relevant guidelines from your company's coding standards with deep-linked URLs to full documentation.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"task": map[string]interface{}{
					"type":        "string",
					"description": "Description of the coding task or question (e.g., 'error handling in Go', 'writing unit tests', 'naming conventions')",
				},
				"scope": map[string]interface{}{
					"type":        "string",
					"description": "Optional scope to narrow results (function, method, struct, interface, package, test, all)",
					"enum":        []string{"function", "method", "struct", "interface", "package", "test", "all"},
					"default":     "all",
				},
				"n_results": map[string]interface{}{
					"type":        "integer",
					"description": "Number of results to return (1-25, default 10)",
					"minimum":     1,
					"maximum":     25,
					"default":     10,
				},
			},
			Required: []string{"task"},
		},
	}, h.fetchAdvisoryForTask)
}

// RegisterPrompts registers prompt instructions with the server.
func (h *Handler) RegisterPrompts(s *server.MCPServer) {
	s.AddPrompt(mcp.Prompt{
		Name:        "code-advisory-usage",
		Description: "Instructions for using the code advisory tool effectively",
		Arguments: []mcp.PromptArgument{
			{
				Name:        "language",
				Description: "Programming language (e.g., 'Go', 'Python')",
				Required:    false,
			},
		},
	}, h.getPromptInstructions)
}

// fetchAdvisoryForTask handles the fetch_advisory_for_task tool invocation.
func (h *Handler) fetchAdvisoryForTask(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters using helper methods
	task, err := request.RequireString("task")
	if err != nil {
		return mcp.NewToolResultError("task parameter is required"), nil
	}

	scope := request.GetString("scope", "all")
	nResults := request.GetInt("n_results", 10)

	// Validate bounds
	if nResults < 1 {
		nResults = 1
	} else if nResults > 25 {
		nResults = 25
	}

	// Augment task with scope if not "all"
	query := task
	if scope != "all" {
		query = fmt.Sprintf("%s for %s", task, scope)
	}

	// Generate query embedding
	queryEmbedding, err := h.embedder.Embed(ctx, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to generate embedding: %v", err)), nil
	}

	// Load database and get collection
	db, err := chromem.NewPersistentDB(h.dbPath, false)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load database: %v", err)), nil
	}

	collection := db.GetCollection("advisories", nil)
	if collection == nil {
		return mcp.NewToolResultError("advisories collection not found"), nil
	}

	// Cap n_results to collection size
	collectionSize := collection.Count()
	if nResults > collectionSize {
		nResults = collectionSize
	}

	// Query the collection
	results, err := collection.QueryEmbedding(ctx, queryEmbedding, nResults, nil, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to query collection: %v", err)), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText("No relevant advisories found for the given task."), nil
	}

	// Format results
	response := fmt.Sprintf("Found %d relevant coding advisories:\n\n", len(results))

	for i, result := range results {
		response += fmt.Sprintf("## Advisory %d (Relevance: %.1f%%)\n\n", i+1, result.Similarity*100)

		// Add metadata
		if url, ok := result.Metadata["url"]; ok {
			response += fmt.Sprintf("**URL:** %s\n\n", url)
		}
		if sourceFile, ok := result.Metadata["source_file"]; ok {
			response += fmt.Sprintf("**Source:** %s\n\n", sourceFile)
		}
		if h2, ok := result.Metadata["h2_heading"]; ok {
			if h3, ok := result.Metadata["h3_heading"]; ok && h3 != "" {
				response += fmt.Sprintf("**Section:** %s > %s\n\n", h2, h3)
			} else {
				response += fmt.Sprintf("**Section:** %s\n\n", h2)
			}
		}

		// Add content
		response += fmt.Sprintf("**Content:**\n\n%s\n\n", result.Content)

		if i < len(results)-1 {
			response += "---\n\n"
		}
	}

	return mcp.NewToolResultText(response), nil
}

// getPromptInstructions returns prompt instructions for using the code advisory tool.
func (h *Handler) getPromptInstructions(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	language, ok := request.Params.Arguments["language"]
	if !ok || language == "" {
		language = "your programming language"
	}

	instructions := fmt.Sprintf(`# Code Advisory Tool Usage Guide

## Purpose
The code advisory tool provides instant access to your organization's coding standards, style guides, and best practices. Use it to write better code that conforms to team standards without manually searching documentation.

## When to Use
- Writing new functions or methods
- Implementing error handling
- Naming variables, functions, or types
- Writing tests
- Structuring packages
- Handling concurrency
- Any coding question where you need guidance

## How to Use

### Basic Query
Ask natural language questions:
- "error handling in %s"
- "naming conventions for functions"
- "how to write unit tests"
- "struct vs interface design"

### With Scope
Narrow results to specific contexts:
- scope: "function" - for function-level guidance
- scope: "test" - for testing-related advice
- scope: "package" - for package design
- scope: "all" (default) - search everything

### Adjust Results
Control the number of results:
- n_results: 5 - quick answer
- n_results: 10 (default) - balanced
- n_results: 25 - comprehensive research

## Example Queries

**Error Handling:**
- Task: "wrapping errors with context"
- Scope: "function"
- Results: 5

**Testing:**
- Task: "table driven tests"
- Scope: "test"
- Results: 10

**Design:**
- Task: "interface design principles"
- Scope: "interface"
- Results: 10

## Response Format
Each advisory includes:
- **Relevance score** - how well it matches your query
- **URL** - direct link to full documentation
- **Source** - which guide it's from
- **Section** - document hierarchy
- **Content** - the actual guidance with code examples

## Tips for Best Results
1. Be specific: "error wrapping with %%w" > "errors"
2. Use technical terms: "goroutine lifecycle" > "threads"
3. Try different phrasings if first attempt doesn't match
4. Adjust n_results if you need more context
5. Check URLs for complete documentation

## Integration with Development
- Query before writing new code
- Reference during code reviews
- Look up unfamiliar patterns
- Verify best practices
- Share URLs with team members

The tool searches %d indexed guidelines and returns results in under 500ms.
`, language, 126) // TODO: Make chunk count dynamic

	return &mcp.GetPromptResult{
		Messages: []mcp.PromptMessage{
			{
				Role: "user",
				Content: mcp.TextContent{
					Type: "text",
					Text: instructions,
				},
			},
		},
	}, nil
}

// Close cleans up resources.
func (h *Handler) Close() error {
	if h.db != nil {
		return h.db.Close()
	}
	return nil
}
