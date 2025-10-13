package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github/machship/code-advisory-mcp/internal/chunker"
	"github/machship/code-advisory-mcp/internal/database"
	"github/machship/code-advisory-mcp/internal/embedder"
	"github/machship/code-advisory-mcp/internal/manifest"
)

type config struct {
	manifestFile   string
	outputFile     string
	backend        string
	azureEndpoint  string
	azureKey       string
	ollamaBaseURL  string
	embeddingModel string
}

func main() {
	cfg := parseFlags()

	if err := run(cfg); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func parseFlags() config {
	cfg := config{}

	flag.StringVar(&cfg.manifestFile, "manifest", getEnv("MANIFEST_FILE", ""), "Path to JSON manifest file (required)")
	flag.StringVar(&cfg.outputFile, "output", getEnv("OUTPUT_FILE", "advisory.db"), "Database output path")
	flag.StringVar(&cfg.backend, "backend", getEnv("EMBEDDING_BACKEND", "azure"), "Embedding backend: azure or ollama")
	flag.StringVar(&cfg.azureEndpoint, "azure-endpoint", getEnv("AZURE_OPENAI_ENDPOINT", ""), "Azure OpenAI endpoint (required for azure backend)")
	flag.StringVar(&cfg.azureKey, "azure-key", getEnv("AZURE_OPENAI_KEY", ""), "Azure OpenAI API key (required for azure backend)")
	flag.StringVar(&cfg.ollamaBaseURL, "ollama-url", getEnv("OLLAMA_BASE_URL", "http://localhost:11434/api"), "Ollama base URL (for ollama backend)")
	flag.StringVar(&cfg.embeddingModel, "model", getEnv("EMBEDDING_MODEL", "text-embedding-3-small"), "Embedding model name")

	flag.Parse()

	if cfg.manifestFile == "" {
		log.Fatal("manifest file is required")
	}

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
		if cfg.embeddingModel == "" {
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

func run(cfg config) error {
	ctx := context.Background()

	// Load manifest
	log.Printf("Loading manifest from %s", cfg.manifestFile)
	m, err := manifest.Load(cfg.manifestFile)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}
	log.Printf("Loaded manifest with %d documents", len(m.Documents))

	// Initialize embedder
	log.Printf("Initializing %s embedder with model %s", cfg.backend, cfg.embeddingModel)

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
		Model:         cfg.embeddingModel,
		OllamaBaseURL: cfg.ollamaBaseURL,
	})
	if err != nil {
		return fmt.Errorf("failed to create embedder: %w", err)
	}

	// Initialize or load database
	log.Printf("Opening database at %s", cfg.outputFile)
	db, err := database.New(cfg.outputFile)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Warning: failed to close database: %v", err)
		}
	}()

	// Process each document
	totalChunks := 0
	skippedChunks := 0
	addedChunks := 0
	failedChunks := 0

	for i, doc := range m.Documents {
		log.Printf("[%d/%d] Processing %s", i+1, len(m.Documents), doc.Path)

		// Read file content
		content, err := os.ReadFile(doc.Path)
		if err != nil {
			log.Printf("  ERROR: failed to read file: %v", err)
			failedChunks++
			continue
		}

		// Get base directory for relative path resolution
		baseDir := filepath.Dir(cfg.manifestFile)
		absPath := doc.Path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(baseDir, absPath)
		}

		// Chunk the document
		chunks, err := chunker.ChunkDocument(content, doc.URI, absPath, doc.Metadata)
		if err != nil {
			log.Printf("  ERROR: failed to chunk document: %v", err)
			failedChunks++
			continue
		}

		log.Printf("  Found %d chunks", len(chunks))

		// Process each chunk
		for j, chunk := range chunks {
			totalChunks++

			// Check if chunk already exists (idempotent)
			exists, err := db.Exists(ctx, chunk.ID)
			if err != nil {
				log.Printf("  ERROR: failed to check if chunk exists: %v", err)
				failedChunks++
				continue
			}

			if exists {
				log.Printf("    [%d/%d] Skipping existing chunk (ID: %s)", j+1, len(chunks), chunk.ID[:8])
				skippedChunks++
				continue
			}

			// Generate embedding
			log.Printf("    [%d/%d] Generating embedding for chunk (ID: %s)", j+1, len(chunks), chunk.ID[:8])
			embedding, err := emb.Embed(ctx, chunk.Content)
			if err != nil {
				log.Printf("    ERROR: failed to generate embedding: %v", err)
				failedChunks++
				continue
			}

			// Prepare metadata
			metadata := make(map[string]string)
			for k, v := range chunk.Metadata {
				metadata[k] = v
			}
			metadata["url"] = chunk.URL
			metadata["source_file"] = chunk.SourceFile
			metadata["h2_heading"] = chunk.H2Heading
			metadata["h3_heading"] = chunk.H3Heading
			metadata["id"] = chunk.ID

			// Add to database
			err = db.Add(ctx, database.Document{
				ID:        chunk.ID,
				Content:   chunk.Content,
				Embedding: embedding,
				Metadata:  metadata,
			})
			if err != nil {
				log.Printf("    ERROR: failed to add chunk to database: %v", err)
				failedChunks++
				continue
			}

			addedChunks++
		}
	}

	log.Println("\n=== Summary ===")
	log.Printf("Total documents processed: %d", len(m.Documents))
	log.Printf("Total chunks: %d", totalChunks)
	log.Printf("Added chunks: %d", addedChunks)
	log.Printf("Skipped chunks (already exist): %d", skippedChunks)
	log.Printf("Failed chunks: %d", failedChunks)
	log.Printf("Database saved to: %s", cfg.outputFile)

	if failedChunks > 0 {
		return fmt.Errorf("%d chunks failed to process", failedChunks)
	}

	return nil
}
