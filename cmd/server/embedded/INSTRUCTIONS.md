# Code Advisory MCP Server - Usage Instructions

## Overview

The Code Advisory MCP server provides instant access to coding standards, style guides, and best practices for Go programming. Use the fetch_advisory_for_task tool to retrieve relevant guidelines that extend beyond your knowledge cut-off date and public training data set.

This server indexes hundreds of chunks from authoritative Go sources:
- Effective Go (Go Team)
- Go Code Review Comments (Go Team)
- Go Test Comments (Go Team)
- Private company style guides

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
- Be specific about the context and the task at hand
- Include technical terms relevant to your question

**n_results** (optional, integer, 1-25)
- Number of advisories to return
- Default: 10
- Use higher values (15-25) for comprehensive research
- The tool will cap this to collection size automatically

### Response Format

Each advisory includes:

**Source** - Document title (e.g., "Effective Go", "Company Go Style Guide")
**URL** - Direct deep-link to the relevant section
**Content** - Complete guidance with headings, code examples, and explanations

Relevance scores help you prioritize - higher percentages indicate better matches.

## Workflow Integration

### Read Workflow

Use fetch_advisory_for_task when you encounter unfamiliar code or patterns:

1. Call the tool with a detailed task description
2. Review the returned advisories and URLs
3. Apply the guidance to understand the code

### Edit Workflow

Integrate the tool into your development process:

1. **Before editing**: Query for best practices on what you're about to implement
2. **During editing**: Reference guidelines for naming, structure, and patterns
3. **After editing**: Verify your code matches recommended approaches
4. **Testing**: Get guidance on writing effective tests

Example workflow:
1. Task: "implementing a server with graceful shutdown handling OS signals"
2. Review returned advisories for signal handling patterns
3. Implement following the examples
4. Task: "writing integration tests for HTTP server with test fixtures and cleanup"
5. Apply testing best practices from results

## Detailed Examples

These examples demonstrate how to write effective task descriptions that retrieve the most relevant advisories. Each example includes the full context of what you're trying to accomplish and why the task description works well.

### Example 1: Error Handling in File Operations

**Scenario:**
You're building a data processing service that reads configuration files, validates them, processes the data, and writes output. You need to implement proper error handling throughout the pipeline so that errors are informative and can be properly handled by calling code.

**Task:**
"implementing comprehensive error handling in a file processing pipeline that reads JSON configuration files from disk, validates the schema and content, processes records according to the config, and writes results to an output file, with proper error wrapping using fmt.Errorf and %w at each step so that calling code can identify whether errors are due to file access problems, invalid JSON, validation failures, or processing errors, and include relevant context like file paths and line numbers in error messages"

**Why this works:**
- Describes the complete pipeline (read → validate → process → write)
- Specifies the error wrapping technique (fmt.Errorf with %w)
- Lists specific error categories that need distinction
- Mentions context that should be preserved (file paths, line numbers)
- Indicates the calling code needs to differentiate error types
- Shows understanding that errors flow through multiple layers

**Expected results:**
- Error wrapping patterns with %w verb
- Strategies for adding context at each layer
- When to use errors.Is vs errors.As
- Custom error types for domain-specific errors
- Error handling in defer statements
- Logging vs returning errors

**Follow-up queries you might make:**
- "creating custom error types that implement the error interface and can be checked with errors.As for file validation failures"
- "testing error paths in file processing code including simulating IO errors and permission problems"

### Example 2: Concurrent Worker Pool Implementation

**Scenario:**
You're building a web scraper that needs to fetch hundreds of URLs concurrently but with controlled parallelism to avoid overwhelming target servers or exhausting system resources. Workers should be able to shut down gracefully when the program receives an interrupt signal.

**Task:**
"designing a worker pool with exactly 10 concurrent goroutines that pull URLs from an input channel, make HTTP requests with timeout and retry logic, and send results to an output channel, with proper shutdown coordination using context cancellation when receiving SIGTERM or SIGINT, ensuring all in-flight requests complete or timeout before the program exits, and using sync.WaitGroup to track worker lifecycle"

**Why this works:**
- Specifies exact concurrency limit (10 workers)
- Details data flow (input channel → workers → output channel)
- Mentions critical features (timeouts, retries, graceful shutdown)
- Names specific primitives (context, sync.WaitGroup)
- Describes signal handling (SIGTERM, SIGINT)
- Addresses the cleanup concern (in-flight requests)
- Shows understanding of production requirements

**Expected results:**
- Worker pool initialization patterns
- Channel creation and sizing strategies
- Context propagation through goroutines
- Signal handling with os/signal package
- WaitGroup usage for coordination
- Proper channel closing discipline
- Avoiding goroutine leaks

**Follow-up queries you might make:**
- "implementing exponential backoff retry logic for HTTP requests with jitter to avoid thundering herd"
- "testing concurrent code for race conditions and using go test -race to detect data races"

### Example 3: Table-Driven Testing for Complex Parser

**Scenario:**
You've written a parser that converts a custom query language into SQL. The parser needs to handle various input formats, validate syntax, handle edge cases like empty strings and SQL injection attempts, and return meaningful error messages. You want comprehensive test coverage.

**Task:**
"creating table-driven tests for a query language parser that converts custom syntax to SQL, with test cases organized in a map[string]struct containing the input query string, expected SQL output, expected error type, and description of what is being tested, using t.Run for subtests so each case runs independently, covering valid queries with various operators and field types, malformed syntax that should return parse errors, edge cases like empty input and extremely long queries, and potential SQL injection patterns that should be rejected, with clear test names that describe the scenario being tested"

**Why this works:**
- Specifies the exact test structure (map[string]struct with t.Run)
- Lists all the test case fields needed
- Categorizes test scenarios (valid, malformed, edge cases, security)
- Mentions specific concerns (SQL injection)
- Notes the importance of clear test names
- Shows understanding that tests document behavior
- Indicates tests should run independently

**Expected results:**
- Table-driven test pattern with map[string]struct
- Subtest organization with t.Run
- Test naming conventions
- Error comparison techniques
- Test data organization
- Testing negative cases
- When to use helper functions

**Follow-up queries you might make:**
- "organizing test fixture data for complex test cases using golden files or embedded test data"
- "testing error messages for helpful context without depending on exact string matching"

### Example 4: HTTP Middleware Chain Design

**Scenario:**
You're building a REST API and need to implement several cross-cutting concerns: request logging with timing, authentication token validation, panic recovery, request ID propagation, and rate limiting. These need to work together cleanly without tight coupling.

**Task:**
"implementing HTTP middleware chain for a REST API where each middleware wraps the next handler and performs operations before and after the request, including authentication middleware that checks Bearer tokens and adds user context, logging middleware that records request method, path, duration, and status code, panic recovery middleware that catches panics and returns 500 errors while logging stack traces, request ID middleware that generates or extracts trace IDs from headers and adds them to context, and rate limiting middleware that uses a token bucket per client IP, with middleware composed in the correct order so authentication happens before authorization checks and logging captures the complete request lifecycle including any middleware failures"

**Why this works:**
- Lists all middleware components and their responsibilities
- Describes the wrapping pattern
- Mentions data flowing through context
- Specifies ordering requirements and why they matter
- Notes error handling at middleware level
- Shows understanding of request lifecycle
- Indicates middleware should be composable

**Expected results:**
- Middleware function signature patterns
- Context usage for request-scoped data
- Handler composition and chaining
- Middleware ordering considerations
- Error handling in middleware
- Testing middleware in isolation
- Standard library patterns (http.Handler interface)

**Follow-up queries you might make:**
- "implementing context values for request-scoped data like user ID and trace ID with type-safe accessors"
- "testing HTTP middleware in isolation by creating mock handlers and inspecting requests and responses"

### Example 5: Package Design for Database Access Layer

**Scenario:**
You're creating a data access package for your application that needs to support multiple database backends (PostgreSQL initially, but may add MySQL later). The package should hide implementation details, provide both transactional and non-transactional operations, and allow dependency injection for testing.

**Task:**
"structuring an internal data access package that defines interfaces for repository operations like Create, Read, Update, Delete on domain entities, with concrete implementations that use database/sql for PostgreSQL, where the public API accepts interfaces for database connections to allow injecting mock implementations in tests, transaction handling is explicit with a BeginTx function that returns a transaction context, and the package exports types for domain entities but keeps query building and row scanning logic internal, following the principle of accepting interfaces and returning concrete types, with clear separation between the repository interface definitions and their implementations so switching database backends would only require a new implementation package"

**Why this works:**
- Specifies package scope (internal) and responsibility (data access)
- Describes the interface design (repository pattern)
- Notes flexibility requirement (multiple backends)
- Mentions testing concern (dependency injection)
- References key principle (accept interfaces, return structs)
- Details visibility (public interfaces vs internal implementation)
- Shows architecture understanding (separation of concerns)
- Lists specific operations (CRUD, transactions)

**Expected results:**
- Repository interface patterns
- Package organization (internal structure)
- Interface vs concrete type decisions
- Transaction handling approaches
- Testing with interfaces
- When to export vs keep internal
- Database/sql patterns

**Follow-up queries you might make:**
- "implementing transaction handling with context.Context to propagate cancellation through database operations"
- "testing database code using interfaces and mocks without requiring a test database"

### Example 6: CLI Tool with Structured Output

**Scenario:**
You're building a command-line tool that can output results in multiple formats (human-readable tables, JSON for scripting, and CSV for spreadsheets). You want clean separation between data generation, formatting, and output so formats can be added easily.

**Task:**
"designing a CLI tool architecture using the flag package for argument parsing where the core logic generates strongly-typed result structures that are passed to formatter functions selected based on a -format flag, with formatters implementing a common interface that accepts the results and returns formatted output as strings or writes directly to io.Writer, supporting formats including human-readable tables using text/tabwriter, JSON with proper marshaling and indentation, and CSV with escaping and headers, with the main function coordinating flag parsing, calling the core logic, selecting the appropriate formatter, and handling any formatting errors, keeping formatters completely independent from business logic so new output formats can be added without touching core code"

**Why this works:**
- Describes full CLI architecture (parsing → logic → formatting → output)
- Specifies exact libraries (flag, text/tabwriter)
- Lists all output formats and their requirements
- Shows design principle (separation of concerns)
- Mentions interface for extensibility
- Details error handling responsibilities
- Indicates main function should be minimal
- Notes that formats should be independent

**Expected results:**
- Main package organization patterns
- Command-line flag patterns
- Interface definitions for formatters
- Separation of logic and presentation
- Using io.Writer for flexible output
- Table formatting techniques
- JSON marshaling with struct tags

**Follow-up queries you might make:**
- "implementing flag validation and printing usage help with custom formatting and examples"
- "writing tests for CLI tools that capture stdout and test different output formats"

### Example 7: Graceful Server Shutdown

**Scenario:**
You have an HTTP server that handles long-running requests (up to 30 seconds) and you need to implement graceful shutdown so that when you deploy a new version, in-flight requests complete successfully but new requests are rejected, with a maximum shutdown timeout.

**Task:**
"implementing graceful shutdown for an http.Server that listens for SIGTERM and SIGINT signals using the os/signal package, when receiving a signal the server stops accepting new connections, allows in-flight HTTP requests up to 30 seconds to complete using Server.Shutdown with a context timeout, logs the shutdown process including number of active connections and whether shutdown completed or timed out, and if shutdown doesn't complete within the timeout, forcibly closes remaining connections using Server.Close and exits with an error code, with the main function setting up signal handling in a separate goroutine that coordinates shutdown, the server running in another goroutine with error returned via a channel, and proper cleanup of any background workers or database connections when shutdown begins"

**Why this works:**
- Specifies the exact shutdown behavior required
- Names the APIs involved (Server.Shutdown, signal.Notify)
- Details the timeout strategy (30 seconds for requests, forced close)
- Describes goroutine coordination
- Mentions logging requirements
- Lists cleanup responsibilities
- Shows understanding of production deployment
- Notes both graceful and forced shutdown paths

**Expected results:**
- Signal handling patterns
- http.Server shutdown sequence
- Context usage for timeouts
- Goroutine coordination
- Error handling during shutdown
- Cleanup patterns
- Exit code handling

**Follow-up queries you might make:**
- "implementing health check endpoints that return unhealthy status during shutdown to help load balancers"
- "testing graceful shutdown behavior including timeout scenarios"


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

## Integration with Development

### Pre-Coding Phase

1. Query for architectural patterns before starting
2. Review returned advisories for design considerations
3. Bookmark relevant URLs for reference
4. Plan implementation based on best practices

### Active Development

1. Query while writing each significant function
2. Keep advisory responses visible for quick reference
3. Apply patterns immediately

### Code Review Phase

1. Query for standards mentioned in review comments
2. Share URLs with team members
3. Use as discussion reference
4. Document decisions with guideline links

### Learning Phase

1. Use higher n_results (15-25) for comprehensive learning
2. Read through all advisories on a topic
3. Build mental models from examples

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
2. Increase n_results to see more options
3. Break complex queries into focused ones

---

Remember: The more specific and detailed your task description, the better the results. Include context, mention Go-specific primitives, and describe exactly what you're trying to accomplish.
