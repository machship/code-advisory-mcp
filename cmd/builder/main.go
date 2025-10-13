package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github/machship/code-advisory-mcp/internal/chunker"
	"github/machship/code-advisory-mcp/internal/database"
	"github/machship/code-advisory-mcp/internal/embedder"
	"github/machship/code-advisory-mcp/internal/manifest"
)

type config struct {
	manifestFile   string
	outputFile     string
	azureEndpoint  string
	azureKey       string
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
	flag.StringVar(&cfg.azureEndpoint, "azure-endpoint", getEnv("AZURE_OPENAI_ENDPOINT", ""), "Azure OpenAI endpoint")
	flag.StringVar(&cfg.azureKey, "azure-key", getEnv("AZURE_OPENAI_KEY", ""), "Azure OpenAI API key")
	flag.StringVar(&cfg.embeddingModel, "model", getEnv("EMBEDDING_MODEL", "text-embedding-3-small"), "Embedding model name")

	flag.Parse()

	if cfg.manifestFile == "" {
		log.Fatal("manifest file is required")
	}
	if cfg.azureEndpoint == "" {
		log.Fatal("azure-endpoint is required")
	}
	if cfg.azureKey == "" {
		log.Fatal("azure-key is required")
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
	log.Printf("Initializing embedder with model %s", cfg.embeddingModel)
	emb, err := embedder.New(cfg.azureEndpoint, cfg.azureKey, cfg.embeddingModel)
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
