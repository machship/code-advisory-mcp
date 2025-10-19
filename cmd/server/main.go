package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github/machship/code-advisory-mcp/internal/embedder"
	"github/machship/code-advisory-mcp/internal/mcp"

	mcpServer "github.com/mark3labs/mcp-go/server"
	"github.com/philippgille/chromem-go"
)

//go:embed embedded/advisory.gob
var embeddedDBBytes []byte

type config struct {
	dbPath            string
	backend           string
	azureEndpoint     string
	azureKey          string
	ollamaBaseURL     string
	model             string
	useEmbedded       bool
	printInstructions bool
}

func main() {
	cfg := parseFlags()

	if cfg.printInstructions {
		printPromptInstructions()
		return
	}

	if err := run(cfg); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func parseFlags() config {
	cfg := config{}

	flag.StringVar(&cfg.dbPath, "db", getEnv("DB_PATH", ""), "Path to advisory database (empty = use embedded)")
	flag.StringVar(&cfg.backend, "backend", getEnv("EMBEDDING_BACKEND", "ollama"), "Embedding backend: azure or ollama")
	flag.StringVar(&cfg.azureEndpoint, "azure-endpoint", getEnv("AZURE_OPENAI_ENDPOINT", ""), "Azure OpenAI endpoint (required for azure backend)")
	flag.StringVar(&cfg.azureKey, "azure-key", getEnv("AZURE_OPENAI_KEY", ""), "Azure OpenAI API key (required for azure backend)")
	flag.StringVar(&cfg.ollamaBaseURL, "ollama-url", getEnv("OLLAMA_BASE_URL", "http://localhost:11434/api"), "Ollama base URL (for ollama backend)")
	flag.StringVar(&cfg.model, "model", getEnv("EMBEDDING_MODEL", "nomic-embed-text"), "Embedding model name")
	flag.BoolVar(&cfg.printInstructions, "print-instructions", false, "Print prompt instructions and exit")

	flag.Parse()

	// Use embedded database if no path provided
	cfg.useEmbedded = (cfg.dbPath == "")

	// Normalize backend
	cfg.backend = strings.ToLower(cfg.backend)

	// Validate backend-specific requirements
	switch cfg.backend {
	case "azure":
		if cfg.azureEndpoint == "" {
			log.Fatal("azure-endpoint is required for azure backend")
		}
		if cfg.azureKey == "" {
			log.Fatal("azure-key is required for azure backend")
		}
	case "ollama":
		if cfg.model == "" {
			log.Fatal("model is required for ollama backend")
		}
	default:
		log.Fatalf("unsupported backend: %s (use 'azure' or 'ollama')", cfg.backend)
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// loadEmbeddedDB loads the embedded database from memory.
func loadEmbeddedDB() (*chromem.DB, error) {
	// Create in-memory database
	db := chromem.NewDB()

	// Create a reader from the embedded bytes
	reader := bytes.NewReader(embeddedDBBytes)

	// Import from reader
	if err := db.ImportFromReader(reader, ""); err != nil {
		return nil, fmt.Errorf("failed to import embedded database: %w", err)
	}

	return db, nil
}

func run(cfg config) error {
	// Determine backend type
	var backend embedder.Backend
	switch cfg.backend {
	case "azure":
		backend = embedder.BackendAzureOpenAI
	case "ollama":
		backend = embedder.BackendOllama
	default:
		return fmt.Errorf("unsupported backend: %s", cfg.backend)
	}

	// Initialize MCP handler with either embedded DB or path
	var handler *mcp.Handler
	var err error

	if cfg.useEmbedded {
		log.Printf("Loading embedded database")
		db, dbErr := loadEmbeddedDB()
		if dbErr != nil {
			return fmt.Errorf("failed to load embedded database: %w", dbErr)
		}

		handler, err = mcp.NewHandler(mcp.Config{
			DB:            db,
			Backend:       backend,
			Endpoint:      cfg.azureEndpoint,
			APIKey:        cfg.azureKey,
			Model:         cfg.model,
			OllamaBaseURL: cfg.ollamaBaseURL,
		})
	} else {
		log.Printf("Loading external database: %s", cfg.dbPath)
		handler, err = mcp.NewHandler(mcp.Config{
			DBPath:        cfg.dbPath,
			Backend:       backend,
			Endpoint:      cfg.azureEndpoint,
			APIKey:        cfg.azureKey,
			Model:         cfg.model,
			OllamaBaseURL: cfg.ollamaBaseURL,
		})
	}

	if err != nil {
		return fmt.Errorf("failed to create MCP handler: %w", err)
	}

	// Create MCP server
	server := mcpServer.NewMCPServer(
		"code-advisory-mcp",
		"1.0.0",
	)

	// Register tools and prompts
	handler.RegisterTools(server)
	handler.RegisterPrompts(server)

	log.Printf("Starting Code Advisory MCP Server")
	if cfg.useEmbedded {
		log.Printf("  Database: embedded (extracted to temp)")
	} else {
		log.Printf("  Database: %s", cfg.dbPath)
	}
	log.Printf("  Backend: %s", cfg.backend)
	log.Printf("  Model: %s", cfg.model)

	// Start server with stdio transport
	if err := mcpServer.ServeStdio(server); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func printPromptInstructions() {
	instructions := `# Code Advisory MCP Server - Usage Instructions

## Overview

The Code Advisory MCP server provides instant access to coding standards, style guides, and best practices for Go programming. Use the fetch_advisory_for_task tool to retrieve relevant guidelines with deep-linked URLs to full documentation.

This server indexes 126 chunks from 4 authoritative Go sources:
- Effective Go (Go Team)
- Go Code Review Comments (Go Team)
- Go Test Comments (Go Team)
- Google Go Style Guide (Google)

## Tool: fetch_advisory_for_task

### When to Use

Call this tool whenever you need guidance on Go coding practices:

1. **Before writing new code** - Get best practices upfront
2. **During code reviews** - Verify adherence to standards
3. **When encountering patterns** - Understand recommended approaches
4. **While debugging** - Learn proper error handling and testing
5. **For naming decisions** - Follow established conventions
6. **When designing APIs** - Apply interface and package design principles

### Parameters

**task** (required, string)
- Detailed description of what you're trying to accomplish
- Be specific about the context and what you need guidance on
- Include technical terms relevant to your question
- Examples:
  - "implementing error wrapping with context using fmt.Errorf and %w verb for error chains"
  - "designing a public API for a package that handles concurrent HTTP requests with proper synchronization"
  - "writing table-driven unit tests for a parser function with edge cases and error conditions"
  - "choosing between interface and struct for a plugin system with dynamic loading"
  - "handling goroutine lifecycle management and graceful shutdown in a long-running service"

**scope** (optional, enum: function, method, struct, interface, package, test, all)
- Narrows results to specific code element types
- Default: "all"
- Use when you want guidance specific to a particular context
- Examples:
  - scope: "test" - for testing-related questions
  - scope: "interface" - for interface design
  - scope: "package" - for package organization

**n_results** (optional, integer, 1-25)
- Number of advisories to return
- Default: 10
- Use lower values (3-5) for quick answers
- Use higher values (15-25) for comprehensive research
- The tool will cap this to collection size automatically

### Response Format

Each advisory includes:

**Source** - Document title (e.g., "Effective Go", "Google Go Style Guide")
**URL** - Direct deep-link to the relevant section
**Content** - Complete guidance with headings, code examples, and explanations

Relevance scores help you prioritize - higher percentages indicate better matches.

## Workflow Integration

### Read Workflow

Use fetch_advisory_for_task when you encounter unfamiliar code or patterns:

1. Call the tool with a detailed task description
2. Review the returned advisories and URLs
3. Apply the guidance to understand the code
4. Visit URLs for comprehensive context if needed

Example:
- You see ` + "`defer file.Close()`" + ` in code
- Task: "when and how to use defer statements for resource cleanup and error handling"
- Apply learned patterns to similar situations

### Edit Workflow

Integrate the tool into your development process:

1. **Before editing**: Query for best practices on what you're about to implement
2. **During editing**: Reference guidelines for naming, structure, and patterns
3. **After editing**: Verify your code matches recommended approaches
4. **Testing**: Get guidance on writing effective tests

Example workflow:
1. Task: "implementing a server with graceful shutdown handling SIGTERM and SIGINT signals"
2. Review returned advisories for signal handling patterns
3. Implement following the examples
4. Task: "writing integration tests for HTTP server with test fixtures and cleanup"
5. Apply testing best practices from results

## Detailed Examples

### Example 1: Error Handling

**Task:**
"implementing comprehensive error handling in a file processing function that validates input, reads files, parses JSON, and writes output with proper error wrapping and context at each step"

**Why this works:**
- Specific operation (file processing)
- Multiple error points identified
- Mentions desired pattern (error wrapping with context)
- Clear about the complexity level

**Expected results:**
- Error wrapping with fmt.Errorf and %w
- Error checking patterns
- Context propagation
- Error vs panic guidelines

### Example 2: Concurrency

**Task:**
"designing a worker pool pattern with bounded concurrency using channels and goroutines where workers process jobs from a queue and results are collected with proper cancellation via context"

**Why this works:**
- Specific pattern (worker pool)
- Mentions primitives (channels, goroutines, context)
- Clear about requirements (bounded, cancellation)
- Indicates data flow (queue → workers → results)

**Expected results:**
- Goroutine lifecycle management
- Channel patterns
- Context usage
- Synchronization approaches

### Example 3: Testing

**Task:**
"creating table-driven tests for a URL parser that handles valid URLs, malformed input, edge cases like empty strings and special characters, with subtests for each scenario"

**Why this works:**
- Specific function type (parser)
- Mentions test structure (table-driven)
- Lists case categories
- Notes subtest usage

**Scope:** "test"

**Expected results:**
- Table-driven test patterns
- Subtest organization
- Error testing approaches
- Test naming conventions

### Example 4: Package Design

**Task:**
"structuring a new internal package for database access that provides both synchronous and asynchronous APIs while hiding implementation details and allowing for future backend changes"

**Why this works:**
- Clear purpose (database access)
- Mentions API styles needed
- Notes design goals (information hiding, flexibility)
- Indicates it's internal (affects public API decisions)

**Scope:** "package"

**Expected results:**
- Package organization principles
- Interface design for flexibility
- Internal vs exported symbols
- API consistency patterns

### Example 5: Interface Design

**Task:**
"deciding whether to accept an interface or concrete type as a parameter in a function that processes data streams and needs to support both in-memory and file-based sources"

**Why this works:**
- Specific design decision
- Context about parameter types
- Notes flexibility requirement
- Mentions concrete use cases

**Scope:** "interface"

**Expected results:**
- "Accept interfaces, return structs" principle
- io.Reader/Writer patterns
- Interface size guidelines
- Dependency inversion

### Example 6: Naming

**Task:**
"choosing appropriate names for a set of functions that initialize, configure, start, stop, and clean up a background service while following Go idioms and avoiding stuttering"

**Why this works:**
- Specific naming scenario
- Lists all functions needing names
- Mentions Go idioms
- Notes specific concern (stuttering)

**Expected results:**
- Naming conventions
- Method naming patterns
- Avoiding redundancy
- Idiomatic verb choices

### Example 7: Code Organization

**Task:**
"organizing a large main.go file that has grown to 500 lines with multiple initialization functions, HTTP handlers, and business logic into a proper package structure"

**Why this works:**
- Specific problem (large main.go)
- Quantifies the issue (500 lines)
- Lists content types
- Clear goal (proper structure)

**Scope:** "package"

**Expected results:**
- Main package patterns
- Package decomposition
- Handler organization
- Initialization approaches

## Best Practices for Task Descriptions

### Do:

✓ Include specific technical terms
✓ Mention the patterns or approaches you're considering
✓ Describe the context and constraints
✓ Note any specific concerns or requirements
✓ Reference Go primitives when relevant (channels, goroutines, interfaces, etc.)
✓ Specify error conditions or edge cases
✓ Indicate performance or concurrency requirements

### Don't:

✗ Use vague terms like "best way" or "good approach"
✗ Ask yes/no questions
✗ Omit context about what you're building
✗ Use generic descriptions
✗ Skip mentioning specific Go language features involved

### Task Quality Comparison

**Poor:**
"error handling"
- Too vague, will return generic results

**Better:**
"error handling in HTTP handlers"
- More specific, but still broad

**Best:**
"implementing error handling in HTTP middleware that wraps handler errors with request context including method, path, and user ID before logging and returning appropriate HTTP status codes"
- Specific component (HTTP middleware)
- Clear inputs (handler errors)
- Detailed operations (wrapping with context)
- Concrete data (method, path, user ID)
- Expected outcomes (logging, status codes)

## Integration with Development

### Pre-Coding Phase

1. Query for architectural patterns before starting
2. Review returned advisories for design considerations
3. Bookmark relevant URLs for reference
4. Plan implementation based on best practices

### Active Development

1. Query while writing each significant function
2. Keep advisory responses visible for quick reference
3. Use scope parameter to filter by current context
4. Apply patterns immediately

### Code Review Phase

1. Query for standards mentioned in review comments
2. Share URLs with team members
3. Use as discussion reference
4. Document decisions with guideline links

### Learning Phase

1. Use higher n_results (15-25) for comprehensive learning
2. Read through all advisories on a topic
3. Visit URLs for deep dives
4. Build mental models from examples

## Advanced Usage

### Chaining Queries

For complex scenarios, make multiple focused queries:

1. High-level design: "designing a plugin system with interface-based plugins"
2. Specific pattern: "loading and initializing plugins from a directory at startup"
3. Error handling: "handling plugin initialization errors with graceful degradation"
4. Testing: "testing plugin system with mock implementations"

### Comparative Learning

Query different approaches:

1. "implementing configuration using struct tags with validation"
2. "implementing configuration using builder pattern with method chaining"
3. Compare results to understand tradeoffs

### Troubleshooting

When results don't match expectations:

1. Rephrase with more technical detail
2. Try different scope values
3. Increase n_results to see more options
4. Break complex queries into focused ones

## Common Patterns and Queries

### Error Handling
- "wrapping errors with context preserving original error for errors.Is and errors.As"
- "creating custom error types with structured data for API responses"
- "handling errors in goroutines and communicating them through channels"

### Concurrency
- "using sync.WaitGroup to wait for multiple goroutines to complete"
- "implementing graceful shutdown with context cancellation and timeout"
- "protecting shared state with sync.Mutex or sync.RWMutex"

### Testing
- "structuring table-driven tests with setup and teardown"
- "using test fixtures and golden files for complex test data"
- "writing benchmarks with b.ResetTimer and proper iteration"

### Interfaces
- "defining minimal interfaces with single methods following interface segregation"
- "using io.Reader and io.Writer for stream processing"
- "type assertions and type switches for interface handling"

### Packages
- "organizing internal and public APIs in a package"
- "avoiding circular dependencies between packages"
- "choosing between package-level functions and methods"

## Troubleshooting

### No Relevant Results

If you get "No relevant advisories found":
- Make your task description more specific
- Include Go-specific terminology
- Check if you're asking about non-Go topics
- Verify the guidance exists in our indexed documents

### Too Many Generic Results

If results aren't specific enough:
- Add more technical detail to the task
- Use the scope parameter
- Mention specific Go language features
- Include the context of your specific use case

### Results Don't Match Question

If results seem off-topic:
- Rephrase using different technical terms
- Break complex questions into smaller queries
- Remove ambiguous language
- Focus on one specific aspect at a time

## Additional Resources

All advisories include URLs to full documentation. Visit these links for:
- Complete context around the guideline
- Additional examples
- Related topics
- Detailed explanations

The indexed sources are authoritative and actively maintained by the Go team and Google.

## Database Contents

**126 chunks** indexed across:

1. **Effective Go** (43 chunks)
   - Formatting, comments, names
   - Control structures, functions, data
   - Initialization, methods, interfaces
   - Concurrency, errors

2. **Go Code Review Comments** (32 chunks)
   - Code organization
   - Naming conventions
   - Error handling
   - Testing practices

3. **Go Test Comments** (14 chunks)
   - Test structure
   - Test naming
   - Table-driven tests
   - Subtests and helpers

4. **Google Go Style Guide** (37 chunks)
   - Style decisions
   - Best practices
   - Common patterns
   - Design principles

## Performance

- Query latency: <500ms including embedding generation
- Embeddings generated via Ollama (nomic-embed-text) or Azure OpenAI
- Results ranked by semantic similarity
- Deep-linked URLs for immediate reference

---

Remember: The more specific and detailed your task description, the better the results. Include context, mention Go-specific primitives, and describe exactly what you're trying to accomplish.
`
	fmt.Println(instructions)
}
