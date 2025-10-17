package main

import (
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github/machship/code-advisory-mcp/internal/embedder"
	"github/machship/code-advisory-mcp/internal/mcp"

	mcpServer "github.com/mark3labs/mcp-go/server"
)

//go:embed embedded
var embeddedDB embed.FS

type config struct {
	dbPath        string
	backend       string
	azureEndpoint string
	azureKey      string
	ollamaBaseURL string
	model         string
	useEmbedded   bool
}

func main() {
	cfg := parseFlags()

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

// extractEmbeddedDB extracts the embedded database to a temporary directory.
func extractEmbeddedDB() (string, func() error, error) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "code-advisory-db-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	cleanup := func() error {
		return os.RemoveAll(tmpDir)
	}

	// Walk the embedded filesystem and copy to temp dir
	err = fs.WalkDir(embeddedDB, "embedded/advisory-ollama.db", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel("embedded/advisory-ollama.db", path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(tmpDir, "advisory-ollama.db", relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// Copy file
		srcFile, err := embeddedDB.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		dstFile, err := os.Create(targetPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		if _, err := io.Copy(dstFile, srcFile); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to extract database: %w", err)
	}

	dbPath := filepath.Join(tmpDir, "advisory-ollama.db")
	return dbPath, cleanup, nil
}

func run(cfg config) error {
	// Extract embedded database if needed
	var dbPath string
	var cleanupFunc func() error

	if cfg.useEmbedded {
		var err error
		dbPath, cleanupFunc, err = extractEmbeddedDB()
		if err != nil {
			return fmt.Errorf("failed to extract embedded database: %w", err)
		}
		defer func() {
			if cleanupFunc != nil {
				if err := cleanupFunc(); err != nil {
					log.Printf("Warning: failed to cleanup temp database: %v", err)
				}
			}
		}()
		log.Printf("Using embedded database")
	} else {
		dbPath = cfg.dbPath
		log.Printf("Using external database: %s", dbPath)
	}

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

	// Initialize MCP handler
	handler, err := mcp.NewHandler(mcp.Config{
		DBPath:        dbPath,
		Backend:       backend,
		Endpoint:      cfg.azureEndpoint,
		APIKey:        cfg.azureKey,
		Model:         cfg.model,
		OllamaBaseURL: cfg.ollamaBaseURL,
	})
	if err != nil {
		return fmt.Errorf("failed to create MCP handler: %w", err)
	}
	defer func() {
		if err := handler.Close(); err != nil {
			log.Printf("Warning: failed to close handler: %v", err)
		}
	}()

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
