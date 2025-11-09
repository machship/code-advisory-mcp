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
2. Review the returned advisories
3. Apply the guidance to understand the code

### Edit Workflow

Integrate the tool into your development process:

1. **Before editing**: Query for best practices on what you're about to implement
2. **During editing**: Reference guidelines for naming, structure, and patterns
3. **After editing**: Verify your code matches recommended approaches
4. **Testing**: Get guidance on writing effective tests

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

Note: your task descriptions must be more detailed (see Detailed Examples).

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

## Final Thoughts

The more specific and detailed your task description, the better the results. Include context, mention Go-specific primitives, and describe exactly what you're trying to accomplish.
