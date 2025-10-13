package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github/machship/code-advisory-mcp/internal/embedder"

	"github.com/philippgille/chromem-go"
)

type config struct {
	dbPath        string
	query         string
	nResults      int
	backend       string
	azureEndpoint string
	azureKey      string
	ollamaBaseURL string
	model         string
}

func main() {
	cfg := parseFlags()

	if err := run(cfg); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func parseFlags() config {
	cfg := config{}

	flag.StringVar(&cfg.dbPath, "db", getEnv("DB_PATH", "advisory-ollama.db"), "Path to database")
	flag.StringVar(&cfg.query, "query", "", "Query text (required)")
	flag.IntVar(&cfg.nResults, "n", 5, "Number of results to return")
	flag.StringVar(&cfg.backend, "backend", getEnv("EMBEDDING_BACKEND", "ollama"), "Embedding backend: azure or ollama")
	flag.StringVar(&cfg.azureEndpoint, "azure-endpoint", getEnv("AZURE_OPENAI_ENDPOINT", ""), "Azure OpenAI endpoint")
	flag.StringVar(&cfg.azureKey, "azure-key", getEnv("AZURE_OPENAI_KEY", ""), "Azure OpenAI API key")
	flag.StringVar(&cfg.ollamaBaseURL, "ollama-url", getEnv("OLLAMA_BASE_URL", "http://localhost:11434/api"), "Ollama base URL")
	flag.StringVar(&cfg.model, "model", getEnv("EMBEDDING_MODEL", "nomic-embed-text"), "Embedding model name")

	flag.Parse()

	if cfg.query == "" {
		flag.Usage()
		os.Exit(1)
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func run(cfg config) error {
	ctx := context.Background()

	// Initialize embedder
	var backend embedder.Backend
	switch cfg.backend {
	case "azure":
		backend = embedder.BackendAzureOpenAI
	case "ollama":
		backend = embedder.BackendOllama
	default:
		return fmt.Errorf("unsupported backend: %s", cfg.backend)
	}

	emb, err := embedder.New(embedder.Config{
		Backend:       backend,
		Endpoint:      cfg.azureEndpoint,
		APIKey:        cfg.azureKey,
		Model:         cfg.model,
		OllamaBaseURL: cfg.ollamaBaseURL,
	})
	if err != nil {
		return fmt.Errorf("failed to create embedder: %w", err)
	}

	// Generate query embedding
	fmt.Printf("Generating embedding for query: %q\n", cfg.query)
	queryEmbedding, err := emb.Embed(ctx, cfg.query)
	if err != nil {
		return fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Load database
	fmt.Printf("Loading database from: %s\n", cfg.dbPath)
	db, err := chromem.NewPersistentDB(cfg.dbPath, false)
	if err != nil {
		return fmt.Errorf("failed to load database: %w", err)
	}

	// Get collection
	collection := db.GetCollection("advisories", nil)
	if collection == nil {
		return fmt.Errorf("collection 'advisories' not found in database")
	}

	fmt.Printf("Querying collection with %d documents...\n\n", collection.Count())

	// Query the collection
	results, err := collection.QueryEmbedding(ctx, queryEmbedding, cfg.nResults, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to query collection: %w", err)
	}

	// Display results
	if len(results) == 0 {
		fmt.Println("No results found.")
		return nil
	}

	fmt.Printf("Found %d results:\n", len(results))
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	for i, result := range results {
		fmt.Printf("Result %d (Similarity: %.4f)\n", i+1, result.Similarity)
		fmt.Printf("URL: %s\n", result.Metadata["url"])
		fmt.Printf("Source: %s\n", result.Metadata["source_file"])
		fmt.Printf("Section: %s > %s\n", result.Metadata["h2_heading"], result.Metadata["h3_heading"])
		fmt.Printf("\nContent Preview:\n%s\n", truncate(result.Content, 400))

		if i < len(results)-1 {
			fmt.Println()
			fmt.Println(strings.Repeat("-", 80))
			fmt.Println()
		}
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
